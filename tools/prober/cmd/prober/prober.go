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
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	retryablehttp "github.com/hashicorp/go-retryablehttp"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	fulciopb "github.com/sigstore/fulcio/pkg/generated/protobuf"
	prototrustroot "github.com/sigstore/protobuf-specs/gen/pb-go/trustroot/v1"
	"github.com/sigstore/sigstore-go/pkg/root"
	"github.com/sigstore/sigstore-go/pkg/sign"
	"github.com/sigstore/sigstore-go/pkg/tuf"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	insec "google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"sigs.k8s.io/release-utils/version"

	"github.com/sigstore/cosign/v2/cmd/cosign/cli/options"
	_ "github.com/sigstore/cosign/v2/pkg/providers/all"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var retryableClient *retryablehttp.Client

// this is copied from sigstore/rekor/openapi.yaml here without imports to keep this light
type InactiveShards struct {
	TreeID   string `json:"treeID"`
	TreeSize int    `json:"treeSize"`
}

type LogInfo struct {
	TreeSize       int              `json:"treeSize"`
	InactiveShards []InactiveShards `json:"inactiveShards"`
}

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
	scPath  string
	trPath  string
	staging bool

	frequency   int
	logStyle    string
	addr        string
	grpcPort    int
	disableGrpc bool

	retries        uint
	oneTime        bool
	runWriteProber bool

	rekorV2URL string

	versionInfo version.Info
)

type attemptCtxKey string

func init() {
	flag.StringVar(&scPath, "signing-config", "", "Path to the signing config")
	flag.StringVar(&trPath, "trusted-root", "", "Path to the trusted root")

	flag.BoolVar(&staging, "staging", false, "Whether to use the public instance staging environment (otherwise use the public instance production environment). For private deployments, use the signing-config and trusted-root flags.")

	flag.IntVar(&frequency, "frequency", 10, "How often to run probers (in seconds)")
	flag.StringVar(&logStyle, "logStyle", "prod", "Log style to use (dev or prod)")
	flag.StringVar(&addr, "addr", ":8080", "Port to expose prometheus to")
	flag.IntVar(&grpcPort, "grpc-port", 0, "Port for Fulcio gRPC endpoint")
	flag.BoolVar(&disableGrpc, "disable-grpc", false, "Whether to disable Fulcio gRPC testing (overrides grpc-port)")

	flag.UintVar(&retries, "retry", 4, "Maximum number of retries before marking HTTP request as failed")
	flag.BoolVar(&oneTime, "one-time", false, "Whether to run only one time and exit")
	flag.BoolVar(&runWriteProber, "write-prober", false, "Whether to run the probers for the write endpoints")

	flag.StringVar(&rekorV2URL, "rekor-v2-url", "", "Set to the Rekor v2 URL to run probers against (will take precedence over any instances listed in the signing config)")

	var rekorV1RequestsJSON string
	flag.StringVar(&rekorV1RequestsJSON, "rekor-requests", "[]", "Additional rekor requests (JSON array)")

	var fulcioRequestsJSON string
	flag.StringVar(&fulcioRequestsJSON, "fulcio-requests", "[]", "Additional fulcio requests (JSON array)")

	flag.Parse()

	ConfigureLogger(logStyle)
	retryableClient = retryablehttp.NewClient()
	retryableClient.Logger = Logger
	retryableClient.RetryMax = int(retries)
	retryableClient.RequestLogHook = func(_ retryablehttp.Logger, r *http.Request, attempt int) {
		ctx := context.WithValue(r.Context(), attemptCtxKey("attempt_number"), attempt)
		*r = *r.WithContext(ctx)
		Logger.Debugf("attempt #%d for %v %v", attempt, r.Method, r.URL)
	}
	retryableClient.ResponseLogHook = func(_ retryablehttp.Logger, r *http.Response) {
		attempt := r.Request.Context().Value(attemptCtxKey("attempt_number"))
		Logger.With(zap.Int("bytes", int(r.ContentLength))).Debugf("attempt #%d result: %d", attempt, r.StatusCode)
	}

	var rekorV1FlagRequests []ReadProberCheck
	if err := json.Unmarshal([]byte(rekorV1RequestsJSON), &rekorV1FlagRequests); err != nil {
		log.Fatal("Failed to parse rekor-requests: ", err)
	}

	var fulcioFlagRequests []ReadProberCheck
	if err := json.Unmarshal([]byte(fulcioRequestsJSON), &fulcioFlagRequests); err != nil {
		log.Fatal("Failed to parse fulcio-requests: ", err)
	}

	ShardlessRekorEndpoints = append(ShardlessRekorEndpoints, rekorV1FlagRequests...)
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

	var err error

	var signingConfig *root.SigningConfig
	var trustedRoot *root.TrustedRoot
	switch {
	case scPath != "" && trPath != "":
		signingConfig, err = root.NewSigningConfigFromPath(scPath)
		if err != nil {
			log.Fatal("Failed to load signing config: ", err)
		}
		trustedRoot, err = root.NewTrustedRootFromPath(trPath)
		if err != nil {
			log.Fatal("Failed to load trusted root: ", err)
		}
	case scPath == "" && trPath == "":
		if staging {
			opts := tuf.DefaultOptions()
			opts.Root = tuf.StagingRoot()
			opts.RepositoryBaseURL = tuf.StagingMirror
			signingConfig, err = root.FetchSigningConfigWithOptions(opts)
			if err != nil {
				log.Fatal("Failed to fetch staging signing config: ", err)
			}
			trustedRoot, err = root.FetchTrustedRootWithOptions(opts)
			if err != nil {
				log.Fatal("Failed to fetch staging trusted root: ", err)
			}
		} else {
			signingConfig, err = root.FetchSigningConfig()
			if err != nil {
				log.Fatal("Failed to fetch prod signing config: ", err)
			}
			trustedRoot, err = root.FetchTrustedRoot()
			if err != nil {
				log.Fatal("Failed to fetch prod trusted root: ", err)
			}
		}
	default:
		log.Fatal("Must specify both --signing-config and --trusted-root, or neither")
	}

	rekorV1Services, err := root.SelectServices(signingConfig.RekorLogURLs(), root.ServiceConfiguration{Selector: prototrustroot.ServiceSelector_ALL}, []uint32{1}, time.Now())
	if err != nil {
		log.Fatal("Failed to select Rekor v1 services: ", err)
	}

	var rekorV2Services []root.Service
	if rekorV2URL != "" {
		rekorV2Services = []root.Service{{URL: rekorV2URL, MajorAPIVersion: 2}}
	} else {
		rekorV2Services, err = root.SelectServices(signingConfig.RekorLogURLs(), root.ServiceConfiguration{Selector: prototrustroot.ServiceSelector_ALL}, []uint32{2}, time.Now())
		if err != nil {
			rekorV2Services = nil
		}
	}

	fulcioService, err := root.SelectService(signingConfig.FulcioCertificateAuthorityURLs(), sign.FulcioAPIVersions, time.Now())
	if err != nil {
		log.Fatal("Failed to select Fulcio service: ", err)
	}

	fulcioGrpcURL := fulcioService.URL
	if strings.HasPrefix(fulcioGrpcURL, "https://") {
		fulcioGrpcURL = strings.TrimPrefix(fulcioGrpcURL, "https://")
	} else if strings.HasPrefix(fulcioGrpcURL, "http://") {
		fulcioGrpcURL = strings.TrimPrefix(fulcioGrpcURL, "http://")
	}
	if grpcPort != 0 {
		host, _, err := net.SplitHostPort(fulcioGrpcURL)
		if err != nil {
			host = fulcioGrpcURL
		}
		fulcioGrpcURL = net.JoinHostPort(host, strconv.Itoa(grpcPort))
	}

	tsaServices, err := root.SelectServices(signingConfig.TimestampAuthorityURLs(), root.ServiceConfiguration{Selector: prototrustroot.ServiceSelector_ALL}, sign.TimestampAuthorityAPIVersions, time.Now())
	if err != nil {
		log.Fatal("Failed to select TSA services: ", err)
	}

	var fulcioClient fulciopb.CAClient
	if !disableGrpc {
		var err error
		fulcioClient, err = NewFulcioGrpcClient(fulcioGrpcURL)
		if err != nil {
			Logger.Fatalf("error creating fulcio grpc client %v", err)
		}
	}
	go runProbers(ctx, frequency, oneTime, fulcioClient, rekorV1Services, rekorV2Services, fulcioService, fulcioGrpcURL, tsaServices, trustedRoot)
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

func NewFulcioGrpcClient(fulcioGrpcURL string) (fulciopb.CAClient, error) {
	grpcHostname := fulcioGrpcURL
	if idx := strings.Index(fulcioGrpcURL, ":"); idx != -1 {
		grpcHostname = fulcioGrpcURL[:idx]
	}
	opts := []grpc.DialOption{grpc.WithUserAgent(options.UserAgent())}

	// Use insecure transport for local testing
	if strings.HasPrefix(grpcHostname, "localhost") {
		opts = append(opts, grpc.WithTransportCredentials(insec.NewCredentials()))
	} else {
		transportCreds := credentials.NewTLS(&tls.Config{MinVersion: tls.VersionTLS12, ServerName: grpcHostname})
		opts = append(opts, grpc.WithTransportCredentials(transportCreds))
	}

	conn, err := grpc.NewClient(fulcioGrpcURL, opts...)
	if err != nil {
		return nil, err
	}
	return fulciopb.NewCAClient(conn), nil
}

func runProbers(ctx context.Context, freq int, runOnce bool, fulcioGrpcClient fulciopb.CAClient, rekorV1Services []root.Service, rekorV2Services []root.Service, fulcioService root.Service, fulcioGrpcURL string, tsaServices []root.Service, trustedRoot *root.TrustedRoot) {
	for {
		hasErr := false

		for _, s := range rekorV1Services {
			// populate shard-specific reads from Rekor endpoint
			rekorEndpointsUnderTest, logInfo, err := determineRekorShardCoverage(s.URL)
			if err != nil {
				hasErr = true
				Logger.Errorf("error determining shard coverage: %v", err)
			}

			if logInfo != nil && logInfo.TreeSize >= 2 {
				rekorEndpointsUnderTest = append(rekorEndpointsUnderTest, ReadProberCheck{
					Endpoint: "/api/v1/log/proof",
					Method:   "GET",
					Queries:  map[string]string{"firstSize": "1", "lastSize": strconv.Itoa(logInfo.TreeSize)},
				})
			}

			rekorEndpointsUnderTest = append(rekorEndpointsUnderTest, ShardlessRekorEndpoints...)

			for _, r := range rekorEndpointsUnderTest {
				if _, err := observeRequest(s.URL, r); err != nil {
					hasErr = true
					Logger.Errorf("error running request %s: %v", r.Endpoint, err)
				}
			}
		}

		for _, s := range rekorV2Services {
			for _, r := range RekorV2ReadEndpoints {
				if _, err := observeRequest(s.URL, r); err != nil {
					hasErr = true
					Logger.Errorf("error running request %s: %v", r.Endpoint, err)
				}
			}
		}

		for _, r := range FulcioEndpoints {
			if _, err := observeRequest(fulcioService.URL, r); err != nil {
				hasErr = true
				Logger.Errorf("error running request %s: %v", r.Endpoint, err)
			}
		}

		for _, s := range tsaServices {
			for _, r := range TSAEndpoints {
				if _, err := observeRequest(s.URL, r); err != nil {
					hasErr = true
					Logger.Errorf("error running request %s: %v", r.Endpoint, err)
				}
			}
		}

		// Performing requests for GetTrustBundle against Fulcio gRPC API
		if fulcioGrpcClient != nil {
			if err := observeGrpcGetTrustBundleRequest(ctx, fulcioGrpcClient, fulcioGrpcURL); err != nil {
				hasErr = true
				Logger.Errorf("error running request %s: %v", "GetTrustBundle", err)
			}
		}

		if runWriteProber {
			priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
			if err != nil {
				Logger.Fatalf("failed to generate key: %v", err)
			}

			cert, err := fulcioWriteEndpoint(ctx, priv, fulcioService, trustedRoot)
			if err != nil {
				hasErr = true
				Logger.Errorf("error running fulcio v2 write prober: %v", err)
			}
			_, err = fulcioWriteLegacyEndpoint(ctx, priv, fulcioService)
			if err != nil {
				hasErr = true
				Logger.Errorf("error running fulcio v1 write prober: %v", err)
			}
			if err := rekorV1WriteEndpoint(ctx, cert, priv, rekorV1Services, trustedRoot); err != nil {
				hasErr = true
				Logger.Errorf("error running rekor write prober: %v", err)
			}
			if err := tsaWriteEndpoint(ctx, priv, tsaServices, trustedRoot); err != nil {
				hasErr = true
				Logger.Errorf("error running tsa write prober: %v", err)
			}
			if len(rekorV2Services) > 0 {
				if err := rekorV2WriteEndpoint(ctx, cert, priv, rekorV2Services); err != nil {
					hasErr = true
					Logger.Errorf("error running rekor v2 write prober: %v", err)
				}
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

func observeRequest(host string, r ReadProberCheck) ([]byte, error) {
	req, err := httpRequest(host, r)
	if err != nil {
		return nil, err
	}

	s := time.Now()
	resp, err := retryableClient.Do(req)
	latency := time.Since(s).Milliseconds()

	if err != nil {
		return nil, err
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

	var respBuffer bytes.Buffer
	if _, err := io.Copy(&respBuffer, resp.Body); err != nil {
		return nil, fmt.Errorf("error reading response: %w", err)
	}
	if resp.StatusCode >= 300 {
		return respBuffer.Bytes(), fmt.Errorf("error response: status: %s, body: %s", resp.Status, respBuffer.String())
	}
	return respBuffer.Bytes(), nil
}

func observeGrpcGetTrustBundleRequest(ctx context.Context, fulcioGrpcClient fulciopb.CAClient, fulcioGrpcURL string) error {
	s := time.Now()
	_, err := fulcioGrpcClient.GetTrustBundle(ctx, &fulciopb.GetTrustBundleRequest{})

	latency := time.Since(s).Milliseconds()
	exportGrpcDataToPrometheus(status.Code(err), "grpc://"+fulcioGrpcURL, "GetTrustBundle", "GET", latency)
	return err
}

func httpRequest(host string, r ReadProberCheck) (*retryablehttp.Request, error) {
	req, err := retryablehttp.NewRequest(r.Method, host+r.Endpoint, bytes.NewBuffer(r.Body))
	if err != nil {
		return nil, err
	}

	setHeaders(req, "", r)
	q := req.URL.Query()
	for k, v := range r.Queries {
		q.Add(k, v)
	}
	req.URL.RawQuery = q.Encode()
	return req, nil
}

// determineRekorShardCoverage adds shard-specific reads to ensure we have coverage across all backing logs
func determineRekorShardCoverage(rekorURL string) ([]ReadProberCheck, *LogInfo, error) {
	req, err := retryablehttp.NewRequest("GET", rekorURL+"/api/v1/log", nil)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid request for loginfo: %w", err)
	}

	setHeaders(req, "", ReadProberCheck{})
	resp, err := retryableClient.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("unexpected error getting loginfo endpoint: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, nil, fmt.Errorf("unexpected response code received from loginfo endpoint: %w", err)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("reading loginfo body: %w", err)
	}

	var logInfo LogInfo
	if err := json.Unmarshal(bodyBytes, &logInfo); err != nil {
		return nil, nil, fmt.Errorf("parsing loginfo: %w", err)
	}

	// if there's no entries, then we're done
	if logInfo.TreeSize == 0 {
		return nil, &logInfo, nil
	}

	// extract relevant endpoints based on index math
	indicesToFetch := make([]int, 0, len(logInfo.InactiveShards)+1)
	offset := 0

	// inactive shards should come first in computation; choose random index within shard
	for _, shard := range logInfo.InactiveShards {
		if shard.TreeSize > 0 {
			indicesToFetch = append(indicesToFetch, offset+mrand.IntN(shard.TreeSize)) // #nosec G404
		}
		offset += shard.TreeSize
	}

	// one final index chosen from active shard
	if activeSize := logInfo.TreeSize - offset; activeSize > 0 {
		indicesToFetch = append(indicesToFetch, offset+mrand.IntN(activeSize)) // #nosec G404
	}

	shardSpecificEndpoints := make([]ReadProberCheck, len(indicesToFetch))
	// convert indices into ReadProberChecks
	for i, index := range indicesToFetch {
		shardSpecificEndpoints[i] = ReadProberCheck{
			Method:   "GET",
			Endpoint: "/api/v1/log/entries",
			Queries:  map[string]string{"logIndex": strconv.Itoa(index)},
		}
	}

	return shardSpecificEndpoints, &logInfo, nil
}
