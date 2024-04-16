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
	"fmt"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
)

var (
	endpointLabel   = "endpoint"
	hostLabel       = "host"
	statusCodeLabel = "status_code"
	methodLabel     = "method"
	verifiedLabel   = "verified"
)

var (
	// Track latency for each endpoint
	endpointLatenciesSummary = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Name:       "api_endpoint_latency",
			Help:       "API endpoint latency distributions (milliseconds).",
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001, .999: 0.0001},
		},
		[]string{endpointLabel, hostLabel, statusCodeLabel, methodLabel},
	)

	endpointLatenciesHistogram = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "api_endpoint_latency_histogram",
		Help:    "API endpoint latency distribution across Rekor and Fulcio (milliseconds)",
		Buckets: []float64{0.0, 200.0, 400.0, 600.0, 800.0, 1000.0},
	},
		[]string{endpointLabel, hostLabel, statusCodeLabel, methodLabel})

	verificationCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "verification",
			Help: "Rekor verification correctness counter",
		},
		[]string{verifiedLabel},
	)
)

func exportDataToPrometheus(resp *http.Response, host, endpoint, method string, latency int64) {
	statusCode := resp.StatusCode
	labels := prometheus.Labels{
		endpointLabel:   endpoint,
		statusCodeLabel: fmt.Sprintf("%d", statusCode),
		hostLabel:       host,
		methodLabel:     method,
	}
	endpointLatenciesSummary.With(labels).Observe(float64(latency))
	endpointLatenciesHistogram.With(labels).Observe(float64(latency))

	Logger.With(zap.Int("status", statusCode), zap.Int("bytes", int(resp.ContentLength)), zap.Duration("latency", time.Duration(latency)*time.Millisecond)).Infof("[DEBUG] %v %v", method, host+endpoint)
}

func exportGrpcDataToPrometheus(statusCode codes.Code, host string, endpoint string, method string, latency int64) {
	labels := prometheus.Labels{
		endpointLabel:   endpoint,
		statusCodeLabel: fmt.Sprintf("%d", statusCode),
		hostLabel:       host,
		methodLabel:     method,
	}
	endpointLatenciesSummary.With(labels).Observe(float64(latency))
	endpointLatenciesHistogram.With(labels).Observe(float64(latency))
	Logger.With(zap.Int32("status", int32(statusCode)), zap.Duration("latency", time.Duration(latency)*time.Millisecond)).Infof("[DEBUG] %v %v %v", method, endpoint, host)
}

// NewVersionCollector returns a collector that exports metrics about current version
// information.
func NewVersionCollector(program string) prometheus.Collector {
	return prometheus.NewGaugeFunc(
		prometheus.GaugeOpts{
			Namespace: program,
			Name:      "build_info",
			Help: fmt.Sprintf(
				"A metric with a constant '1' value labeled by version, revision, branch, and goversion from which %s was built.",
				program,
			),
			ConstLabels: prometheus.Labels{
				"version":    versionInfo.GitVersion,
				"revision":   versionInfo.GitCommit,
				"build_date": versionInfo.BuildDate,
				"goversion":  versionInfo.GoVersion,
			},
		},
		func() float64 { return 1 },
	)
}
