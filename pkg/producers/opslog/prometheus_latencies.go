// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and prysm contributors
//
// SPDX-License-Identifier: Apache-2.0

package opslog

import "github.com/prometheus/client_golang/prometheus"

var (
	// Detailed latency histogram (no pod label to reduce cardinality)
	requestsDurationHistogram = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "radosgw_requests_duration",
			Help:    "Histogram for request latencies with full detail",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"user", "tenant", "bucket", "method"},
	)

	// Aggregated latency histograms
	requestsDurationPerUserHistogram = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "radosgw_requests_duration_per_user",
			Help:    "Histogram for request latencies aggregated per user (all buckets combined)",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"user", "tenant", "method"},
	)

	requestsDurationPerBucketHistogram = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "radosgw_requests_duration_per_bucket",
			Help:    "Histogram for request latencies aggregated per bucket (all users combined)",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"tenant", "bucket", "method"},
	)

	requestsDurationPerTenantHistogram = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "radosgw_requests_duration_per_tenant",
			Help:    "Histogram for request latencies aggregated per tenant (all users and buckets combined)",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"tenant", "method"},
	)

	requestsDurationPerMethodHistogram = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "radosgw_requests_duration_per_method",
			Help:    "Histogram for request latencies aggregated per method (global)",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method"},
	)

	requestsDurationPerBucketAndMethodHistogram = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "radosgw_requests_duration_per_bucket_and_method",
			Help:    "Histogram for request latencies aggregated per bucket and method (all users combined)",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"tenant", "bucket", "method"},
	)
)

// Latency observation function - called during request processing
var latencyObs func(user, tenant, bucket, method string, seconds float64)

// Flag to prevent duplicate registration
var latencyMetricsRegistered bool

func registerLatencyMetrics(metricsConfig *MetricsConfig) {
	// Prevent duplicate registration
	if latencyMetricsRegistered {
		return
	}

	registeredAny := false

	// Register detailed histogram if enabled
	if metricsConfig.TrackLatencyDetailed {
		prometheus.MustRegister(requestsDurationHistogram)
		registeredAny = true
	}

	// Conditional registrations for aggregated histograms
	if metricsConfig.TrackLatencyPerUser {
		prometheus.MustRegister(requestsDurationPerUserHistogram)
		registeredAny = true
	}

	if metricsConfig.TrackLatencyPerBucket {
		prometheus.MustRegister(requestsDurationPerBucketHistogram)
		registeredAny = true
	}

	if metricsConfig.TrackLatencyPerTenant {
		prometheus.MustRegister(requestsDurationPerTenantHistogram)
		registeredAny = true
	}

	if metricsConfig.TrackLatencyPerMethod {
		prometheus.MustRegister(requestsDurationPerMethodHistogram)
		registeredAny = true
	}

	if metricsConfig.TrackLatencyPerBucketAndMethod {
		prometheus.MustRegister(requestsDurationPerBucketAndMethodHistogram)
		registeredAny = true
	}

	// Set up the latency observation function based on config
	if registeredAny {
		latencyObs = createLatencyObsFunction(metricsConfig)
	} else {
		// No-op function if no latency tracking is enabled
		latencyObs = func(user, tenant, bucket, method string, seconds float64) {}
	}

	latencyMetricsRegistered = true
}

// createLatencyObsFunction creates the actual observation function based on config
func createLatencyObsFunction(metricsConfig *MetricsConfig) func(string, string, string, string, float64) {
	return func(user, tenant, bucket, method string, seconds float64) {
		// The user and tenant parameters are already extracted - use them directly
		// Do NOT extract again!
		// Observe detailed histogram if enabled
		if metricsConfig.TrackLatencyDetailed {
			requestsDurationHistogram.With(prometheus.Labels{
				"user":   user,   // Use directly
				"tenant": tenant, // Use directly
				"bucket": bucket,
				"method": method,
			}).Observe(seconds)
		}

		// Conditional observations based on config
		if metricsConfig.TrackLatencyPerUser {
			requestsDurationPerUserHistogram.With(prometheus.Labels{
				"user":   user,   // Use directly
				"tenant": tenant, // Use directly
				"method": method,
			}).Observe(seconds)
		}

		if metricsConfig.TrackLatencyPerBucket {
			requestsDurationPerBucketHistogram.With(prometheus.Labels{
				"tenant": tenant, // Use directly
				"bucket": bucket,
				"method": method,
			}).Observe(seconds)
		}

		if metricsConfig.TrackLatencyPerTenant {
			requestsDurationPerTenantHistogram.With(prometheus.Labels{
				"tenant": tenant, // Use directly
				"method": method,
			}).Observe(seconds)
		}

		if metricsConfig.TrackLatencyPerMethod {
			requestsDurationPerMethodHistogram.With(prometheus.Labels{
				"method": method,
			}).Observe(seconds)
		}

		if metricsConfig.TrackLatencyPerBucketAndMethod {
			requestsDurationPerBucketAndMethodHistogram.With(prometheus.Labels{
				"tenant": tenant, // Use directly
				"bucket": bucket,
				"method": method,
			}).Observe(seconds)
		}
	}
}
