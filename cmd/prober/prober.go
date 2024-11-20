// Copyright 2022 The Sigstore Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	mrand "math/rand/v2"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	retryablehttp "github.com/hashicorp/go-retryablehttp"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	fulciopb "github.com/sigstore/fulcio/pkg/generated/protobuf"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/status"
	"sigs.k8s.io/release-utils/version"

	"github.com/sigstore/cosign/v2/cmd/cosign/cli/options"
	_ "github.com/sigstore/cosign/v2/pkg/providers/all"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var retryableClient *retryablehttp.Client

type proberLogger struct {
	*zap.SugaredLogger
}

func (p proberLogger) Printf(msg string, args ...interface{}) {
	p.Infof(msg, args...)
}

var Logger proberLogger

func ConfigureLogger(location string) {
	cfg := zap.NewProductionConfig()
	switch location {
	case "prod":
		cfg.EncoderConfig.LevelKey = "severity"
		cfg.EncoderConfig.MessageKey = "message"
		cfg.EncoderConfig.TimeKey = "time"
		cfg.EncoderConfig.EncodeLevel = encodeLevel()
		cfg.EncoderConfig.EncodeTime = zapcore.RFC3339NanoTimeEncoder
		cfg.EncoderConfig.EncodeDuration = zapcore.MillisDurationEncoder
		cfg.EncoderConfig.EncodeCaller = zapcore.ShortCallerEncoder
	default:
		cfg = zap.NewDevelopmentConfig()
		cfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}
	logger, err := cfg.Build()
	if err != nil {
		log.Fatalln("createLogger", err)
	}
	Logger = proberLogger{logger.Sugar()}
}

func encodeLevel() zapcore.LevelEncoder {
	return func(l zapcore.Level, enc zapcore.PrimitiveArrayEncoder) {
		switch l {
		case zapcore.DebugLevel:
			enc.AppendString("DEBUG")
		case zapcore.InfoLevel:
			enc.AppendString("INFO")
		case zapcore.WarnLevel:
			enc.AppendString("WARNING")
		case zapcore.ErrorLevel:
			enc.AppendString("ERROR")
		case zapcore.DPanicLevel:
			enc.AppendString("CRITICAL")
		case zapcore.PanicLevel:
			enc.AppendString("ALERT")
		case zapcore.FatalLevel:
			enc.AppendString("EMERGENCY")
		}
	}
}

var (
	logStyle       string
	frequency      int
	retries        uint
	addr           string
	rekorURL       string
	fulcioURL      string
	fulcioGrpcURL  string
	oneTime        bool
	runWriteProber bool
	versionInfo    version.Info
)

type attemptCtxKey string

func init() {
	flag.IntVar(&frequency, "frequency", 10, "How often to run probers (in seconds)")
	flag.StringVar(&logStyle, "logStyle", "prod", "log style to use (dev or prod)")
	flag.StringVar(&addr, "addr", ":8080", "Port to expose prometheus to")

	flag.StringVar(&rekorURL, "rekor-url", "https://rekor.sigstore.dev", "Set to the Rekor URL to run probers against")
	flag.StringVar(&fulcioURL, "fulcio-url", "https://fulcio.sigstore.dev", "Set to the Fulcio URL to run probers against")
	flag.StringVar(&fulcioGrpcURL, "fulcio-grpc-url", "fulcio.sigstore.dev", "Set to the Fulcio GRPC URL to run probers against")

	flag.BoolVar(&oneTime, "one-time", false, "Whether to run only one time and exit.")
	flag.BoolVar(&runWriteProber, "write-prober", false, " [Kubernetes only] run the probers for the write endpoints.")

	var rekorRequestsJSON string
	flag.StringVar(&rekorRequestsJSON, "rekor-requests", "[]", "Additional rekor requests (JSON array.)")

	var fulcioRequestsJSON string
	flag.StringVar(&fulcioRequestsJSON, "fulcio-requests", "[]", "Additional fulcio requests (JSON array.)")

	flag.UintVar(&retries, "retry", 4, "maximum number of retries before marking HTTP request as failed")

	flag.Parse()

	ConfigureLogger(logStyle)
	retryableClient = retryablehttp.NewClient()
	retryableClient.Logger = Logger
	retryableClient.RetryMax = int(retries)
	retryableClient.RequestLogHook = func(_ retryablehttp.Logger, r *http.Request, attempt int) {
		ctx := context.WithValue(r.Context(), attemptCtxKey("attempt_number"), attempt)
		*r = *r.WithContext(ctx)
		Logger.Infof("attempt #%d for %v %v", attempt, r.Method, r.URL)
	}
	retryableClient.ResponseLogHook = func(_ retryablehttp.Logger, r *http.Response) {
		attempt := r.Request.Context().Value(attemptCtxKey("attempt_number"))
		Logger.With(zap.Int("bytes", int(r.ContentLength))).Infof("attempt #%d result: %d", attempt, r.StatusCode)
	}

	var rekorFlagRequests []ReadProberCheck
	if err := json.Unmarshal([]byte(rekorRequestsJSON), &rekorFlagRequests); err != nil {
		log.Fatal("Failed to parse rekor-requests: ", err)
	}

	var fulcioFlagRequests []ReadProberCheck
	if err := json.Unmarshal([]byte(fulcioRequestsJSON), &fulcioFlagRequests); err != nil {
		log.Fatal("Failed to parse fulcio-requests: ", err)
	}

	RekorEndpoints = append(RekorEndpoints, rekorFlagRequests...)
	FulcioEndpoints = append(FulcioEndpoints, fulcioFlagRequests...)
}

func main() {
	ctx := context.Background()
	versionInfo = version.GetVersionInfo()
	Logger.Infof("running prober Version: %s GitCommit: %s BuildDate: %s", versionInfo.GitVersion, versionInfo.GitCommit, versionInfo.BuildDate)

	reg := prometheus.NewRegistry()
	reg.MustRegister(endpointLatenciesSummary, endpointLatenciesHistogram, verificationCounter)
	reg.MustRegister(NewVersionCollector("sigstore_prober"))

	// Ensure that we report zeroed failures on verifications.  This allows us to
	// detect on alert on the "never seen" --> "seen once" transition.
	verificationCounter.With(prometheus.Labels{verifiedLabel: "false"}).Add(0)
	verificationCounter.With(prometheus.Labels{verifiedLabel: "true"}).Add(0)

	if fulcioClient, err := NewFulcioGrpcClient(); err != nil {
		Logger.Fatalf("error creating fulcio grpc client %v", err)
	} else {
		go runProbers(ctx, frequency, oneTime, fulcioClient)
	}
	// Expose the registered metrics via HTTP.
	http.Handle("/metrics", promhttp.HandlerFor(
		reg,
		promhttp.HandlerOpts{
			// Opt into OpenMetrics to support exemplars.
			EnableOpenMetrics: true,
		},
	))
	Logger.Infof("Starting Prometheus Server on port %s", addr)
	/* #nosec G114 */
	Logger.Fatal(http.ListenAndServe(addr, nil))
}

func NewFulcioGrpcClient() (fulciopb.CAClient, error) {
	grpcHostname := fulcioGrpcURL
	if idx := strings.Index(fulcioGrpcURL, ":"); idx != -1 {
		grpcHostname = fulcioGrpcURL[:idx]
	}
	opts := []grpc.DialOption{grpc.WithUserAgent(options.UserAgent())}
	transportCreds := credentials.NewTLS(&tls.Config{MinVersion: tls.VersionTLS12, ServerName: grpcHostname})
	opts = append(opts, grpc.WithTransportCredentials(transportCreds))
	conn, err := grpc.NewClient(fulcioGrpcURL, opts...)
	if err != nil {
		return nil, err
	}
	return fulciopb.NewCAClient(conn), nil
}

func runProbers(ctx context.Context, freq int, runOnce bool, fulcioGrpcClient fulciopb.CAClient) {
	for {
		hasErr := false

		// populate shard-specific reads from Rekor endpoint
		if err := determineRekorShardCoverage(rekorURL); err != nil {
			hasErr = true
			Logger.Errorf("error determining shard coverage: %v", err)
		}

		for _, r := range RekorEndpoints {
			if err := observeRequest(rekorURL, r); err != nil {
				hasErr = true
				Logger.Errorf("error running request %s: %v", r.Endpoint, err)
			}
		}
		for _, r := range FulcioEndpoints {
			if err := observeRequest(fulcioURL, r); err != nil {
				hasErr = true
				Logger.Errorf("error running request %s: %v", r.Endpoint, err)
			}
		}

		// Performing requests for GetTrustBundle against Fulcio gRPC API
		if err := observeGrcpGetTrustBundleRequest(ctx, fulcioGrpcClient); err != nil {
			hasErr = true
			Logger.Errorf("error running request %s: %v", "GetTrustBundle", err)
		}

		if runWriteProber {
			priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
			if err != nil {
				Logger.Fatalf("failed to generate key: %v", err)
			}

			cert, err := fulcioWriteEndpoint(ctx, priv)
			if err != nil {
				hasErr = true
				Logger.Errorf("error running fulcio v2 write prober: %v", err)
			}
			_, err = fulcioWriteLegacyEndpoint(ctx, priv)
			if err != nil {
				hasErr = true
				Logger.Errorf("error running fulcio v1 write prober: %v", err)
			}
			if err := rekorWriteEndpoint(ctx, cert, priv); err != nil {
				hasErr = true
				Logger.Errorf("error running rekor write prober: %v", err)
			}
		}

		if runOnce {
			if hasErr {
				Logger.Fatal("Failed")
			} else {
				Logger.Info("Complete")
				os.Exit(0)
			}
		}

		time.Sleep(time.Duration(freq) * time.Second)
	}
}

func observeRequest(host string, r ReadProberCheck) error {
	req, err := httpRequest(host, r)
	if err != nil {
		return err
	}

	s := time.Now()
	resp, err := retryableClient.Do(req)
	latency := time.Since(s).Milliseconds()

	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Report the normalized SLO endpoint to prometheus if
	// one is specified. This allows us to report metrics for
	// "/api/v1/log/entries/{entryUUID}" instead of
	// "/api/v1/log/entries/<literal uuid>.
	sloEndpoint := r.SLOEndpoint
	if sloEndpoint == "" {
		sloEndpoint = r.Endpoint
	}
	exportDataToPrometheus(resp, host, sloEndpoint, r.Method, latency)

	// right we're not doing anything with the body, but let's at least read it all from the server
	if _, err := io.Copy(io.Discard, resp.Body); err != nil {
		return fmt.Errorf("error reading response: %w", err)
	}

	return nil
}

func observeGrcpGetTrustBundleRequest(ctx context.Context, fulcioGrpcClient fulciopb.CAClient) error {
	s := time.Now()
	_, err := fulcioGrpcClient.GetTrustBundle(ctx, &fulciopb.GetTrustBundleRequest{})

	latency := time.Since(s).Milliseconds()
	exportGrpcDataToPrometheus(status.Code(err), "grpc://"+fulcioGrpcURL, "GetTrustBundle", "GET", latency)
	return err
}

func httpRequest(host string, r ReadProberCheck) (*retryablehttp.Request, error) {
	req, err := retryablehttp.NewRequest(r.Method, host+r.Endpoint, bytes.NewBuffer([]byte(r.Body)))
	if err != nil {
		return nil, err
	}

	setHeaders(req, "")
	q := req.URL.Query()
	for k, v := range r.Queries {
		q.Add(k, v)
	}
	req.URL.RawQuery = q.Encode()
	return req, nil
}

// determineRekorShardCoverage adds shard-specific reads to ensure we have coverage across all backing logs
func determineRekorShardCoverage(rekorURL string) error {
	req, err := retryablehttp.NewRequest("GET", rekorURL+"/api/v1/log", nil)
	if err != nil {
		return fmt.Errorf("invalid request for loginfo: %w", err)
	}

	setHeaders(req, "")
	resp, err := retryableClient.Do(req)
	if err != nil {
		return fmt.Errorf("unexpected error getting loginfo endpoint: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected response code received from loginfo endpoint: %w", err)
	}

	// this is copied from sigstore/rekor/openapi.yaml here without imports to keep this light
	type InactiveShards struct {
		TreeID   string `json:"treeID"`
		TreeSize int    `json:"treeSize"`
	}

	type LogInfo struct {
		TreeSize       int              `json:"treeSize"`
		InactiveShards []InactiveShards `json:"inactiveShards"`
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading loginfo body: %w", err)
	}

	var logInfo LogInfo
	if err := json.Unmarshal(bodyBytes, &logInfo); err != nil {
		return fmt.Errorf("parsing loginfo: %w", err)
	}

	// if there's no entries, then we're done
	if logInfo.TreeSize == 0 {
		return nil
	}

	// extract relevant endpoints based on index math
	indicesToFetch := make([]int, len(logInfo.InactiveShards)+1)
	offset := 0

	// inactive shards should come first in computation; choose random index within shard
	for i, shard := range logInfo.InactiveShards {
		indicesToFetch[i] = offset + mrand.IntN(shard.TreeSize-offset) // #nosec G404
		offset += shard.TreeSize
	}

	// one final index chosen from active shard
	indicesToFetch[len(indicesToFetch)-1] = offset + mrand.IntN(logInfo.TreeSize) // #nosec G404

	// convert indices into ReadProberChecks
	for _, index := range indicesToFetch {
		RekorEndpoints = append(RekorEndpoints, ReadProberCheck{
			Method:   "GET",
			Endpoint: "/api/v1/log/entries",
			Queries:  map[string]string{"logIndex": strconv.Itoa(index)},
		})
	}

	return nil
}
