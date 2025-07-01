// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and prysm contributors
//
// SPDX-License-Identifier: Apache-2.0

package opslog

import (
	"strings"
	"sync/atomic"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog/log"
)

var (
	// Detailed error metrics
	errorsDetailedCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "radosgw_errors_detailed",
			Help: "Total number of errors with full detail",
		},
		[]string{"pod", "user", "tenant", "bucket", "http_status"},
	)

	// Aggregated error metrics - per user (all buckets combined)
	errorsPerUserCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "radosgw_errors_per_user",
			Help: "Total errors aggregated per user (all buckets combined)",
		},
		[]string{"pod", "user", "tenant", "http_status"},
	)

	// Aggregated error metrics - per bucket (all users combined)
	errorsPerBucketCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "radosgw_errors_per_bucket",
			Help: "Total errors aggregated per bucket (all users combined)",
		},
		[]string{"pod", "tenant", "bucket", "http_status"},
	)

	// Aggregated error metrics - per tenant (all users and buckets combined)
	errorsPerTenantCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "radosgw_errors_per_tenant",
			Help: "Total errors aggregated per tenant (all users and buckets combined)",
		},
		[]string{"pod", "tenant", "http_status"},
	)

	// Aggregated error metrics - per status code (global)
	errorsPerStatusCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "radosgw_errors_per_status",
			Help: "Total errors aggregated per HTTP status code (global)",
		},
		[]string{"pod", "http_status"},
	)

	// IP-based error metrics
	errorsPerIPCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "radosgw_errors_per_ip",
			Help: "Total errors aggregated per IP (all buckets combined)",
		},
		[]string{"pod", "ip", "tenant", "http_status"},
	)
)

func registerErrorMetrics(metricsConfig *MetricsConfig) {
	// Register detailed error counter if enabled
	if metricsConfig.TrackErrorsDetailed {
		prometheus.MustRegister(errorsDetailedCounter)
	}

	// Conditional registrations based on config
	if metricsConfig.TrackErrorsPerUser {
		prometheus.MustRegister(errorsPerUserCounter)
	}

	if metricsConfig.TrackErrorsPerBucket {
		prometheus.MustRegister(errorsPerBucketCounter)
	}

	if metricsConfig.TrackErrorsPerTenant {
		prometheus.MustRegister(errorsPerTenantCounter)
	}

	if metricsConfig.TrackErrorsPerStatus {
		prometheus.MustRegister(errorsPerStatusCounter)
	}

	if metricsConfig.TrackErrorsByIP {
		prometheus.MustRegister(errorsPerIPCounter)
	}
}

func publishErrorCounters(diffMetrics *Metrics, cfg OpsLogConfig) {
	metricsConfig := cfg.MetricsConfig

	// Process ErrorsByUserAndBucket data
	diffMetrics.ErrorsByUserAndBucket.Range(func(key, count any) bool {
		parts := strings.Split(key.(string), "|")
		if len(parts) != 3 {
			log.Warn().Msgf("Invalid key format in ErrorsByUserAndBucket: %v", key)
			return true
		}

		user, bucket, status := parts[0], parts[1], parts[2]

		// Exclude HTTP status codes in the 2xx range
		if strings.HasPrefix(status, "2") {
			return true
		}

		userStr, tenantStr := extractUserAndTenant(user)
		errorCount := float64(count.(*atomic.Uint64).Load())

		if errorCount <= 0 {
			return true
		}

		// Detailed error metric - only if enabled
		if metricsConfig.TrackErrorsDetailed {
			errorsDetailedCounter.With(prometheus.Labels{
				"pod":         cfg.PodName,
				"user":        userStr,
				"tenant":      tenantStr,
				"bucket":      bucket,
				"http_status": status,
			}).Add(errorCount)
		}

		// Aggregated metrics based on config
		if metricsConfig.TrackErrorsPerUser {
			errorsPerUserCounter.With(prometheus.Labels{
				"pod":         cfg.PodName,
				"user":        userStr,
				"tenant":      tenantStr,
				"http_status": status,
			}).Add(errorCount)
		}

		if metricsConfig.TrackErrorsPerBucket {
			errorsPerBucketCounter.With(prometheus.Labels{
				"pod":         cfg.PodName,
				"tenant":      tenantStr,
				"bucket":      bucket,
				"http_status": status,
			}).Add(errorCount)
		}

		if metricsConfig.TrackErrorsPerTenant {
			errorsPerTenantCounter.With(prometheus.Labels{
				"pod":         cfg.PodName,
				"tenant":      tenantStr,
				"http_status": status,
			}).Add(errorCount)
		}

		return true
	})

	// Process ErrorsByIPAndBucket data
	if metricsConfig.TrackErrorsByIP {
		diffMetrics.ErrorsByIPAndBucket.Range(func(key, count any) bool {
			parts := strings.Split(key.(string), "|")
			if len(parts) != 4 {
				log.Warn().Msgf("Invalid key format in ErrorsByIPAndBucket: %v", key)
				return true
			}
			ip, user, _, status := parts[0], parts[1], parts[2], parts[3] // Use _ for unused bucket

			// Exclude HTTP status codes in the 2xx range
			if strings.HasPrefix(status, "2") {
				return true
			}

			_, tenantStr := extractUserAndTenant(user)
			errorCount := float64(count.(*atomic.Uint64).Load())

			if errorCount <= 0 {
				return true
			}

			errorsPerIPCounter.With(prometheus.Labels{
				"pod":         cfg.PodName,
				"ip":          ip,
				"tenant":      tenantStr,
				"http_status": status,
			}).Add(errorCount)

			return true
		})
	}

	// Process RequestsPerStatusCode for global status aggregation
	if metricsConfig.TrackErrorsPerStatus {
		diffMetrics.RequestsPerStatusCode.Range(func(status, count any) bool {
			statusStr := status.(string)

			// Exclude HTTP status codes in the 2xx range
			if strings.HasPrefix(statusStr, "2") {
				return true
			}

			requestCount := float64(count.(*atomic.Uint64).Load())

			if requestCount <= 0 {
				return true
			}

			errorsPerStatusCounter.With(prometheus.Labels{
				"pod":         cfg.PodName,
				"http_status": statusStr,
			}).Add(requestCount)

			return true
		})
	}
}
