package main

import "github.com/prometheus/client_golang/prometheus"

var (
	endpointLabel   = "endpoint"
	hostLabel       = "host"
	statusCodeLabel = "status_code"
)

var (
	// Track latency for each endpoint
	endpointLatenciesSummary = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Name:       "api_endpoint_latency",
			Help:       "API endpoint latency distributions.",
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001, .999: 0.0001},
		},
		[]string{endpointLabel, hostLabel, statusCodeLabel},
	)

	endpointLatenciesHistogram = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "api_endpoint_latency_histogram",
		Help:    "API endpoint latency distribution across Rekor and Fulcio (milliseconds)",
		Buckets: []float64{0.0, 200.0, 400.0, 600.0, 800.0, 1000.0},
	},
		[]string{endpointLabel, hostLabel, statusCodeLabel})
)
