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
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"os"
	"time"

	retryablehttp "github.com/hashicorp/go-retryablehttp"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"sigs.k8s.io/release-utils/version"

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
		cfg.EncoderConfig.EncodeTime = zapcore.RFC3339TimeEncoder
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

	go runProbers(ctx, frequency, oneTime)

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

func runProbers(ctx context.Context, freq int, runOnce bool) {
	for {
		hasErr := false

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
		if runWriteProber {
			if err := fulcioWriteEndpoint(ctx); err != nil {
				hasErr = true
				Logger.Errorf("error running fulcio write prober: %v", err)
			}
			if err := rekorWriteEndpoint(ctx); err != nil {
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
	return nil
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
