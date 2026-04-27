// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and prysm contributors
//
// SPDX-License-Identifier: Apache-2.0

package opslog

import "github.com/prometheus/client_golang/prometheus"

var (
	sliRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "radosgw_bucket_sli_requests_total",
			Help: "Bucket GET/LIST requests for SLI and SLO evaluation",
		},
		[]string{"tenant", "bucket", "operation", "status_class"},
	)

	sliRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "radosgw_bucket_sli_request_duration_seconds",
			Help:    "Latency histogram for bucket GET/LIST requests used for SLI and SLO evaluation",
			Buckets: []float64{0.05, 0.1, 0.2, 0.3, 0.5, 1, 2, 5, 10},
		},
		[]string{"tenant", "bucket", "operation"},
	)
)

func registerSLIMetrics() {
	prometheus.MustRegister(sliRequestsTotal)
	prometheus.MustRegister(sliRequestDuration)
}
