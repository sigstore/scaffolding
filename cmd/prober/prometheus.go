package main

import "github.com/prometheus/client_golang/prometheus"

var (
	// Track latency for each endpoint
	endpointLatencies = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Name:       "api_endpoint_latency",
			Help:       "API endpoint latency distributions.",
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001, .999: 0.0001},
		},
		[]string{"endpoint"},
	)
)
