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

	// Timeout-specific error metrics
	timeoutErrorsCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "radosgw_timeout_errors",
			Help: "Total number of timeout errors by type (408, 504, 598, 499)",
		},
		[]string{"pod", "user", "tenant", "bucket", "timeout_type"},
	)

	// Enhanced error categorization metrics
	errorsByCategoryCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "radosgw_errors_by_category",
			Help: "Errors categorized by type (client, server, timeout, connection)",
		},
		[]string{"pod", "tenant", "bucket", "error_category", "http_status"},
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

	if metricsConfig.TrackTimeoutErrors {
		prometheus.MustRegister(timeoutErrorsCounter)
	}

	if metricsConfig.TrackErrorsByCategory {
		prometheus.MustRegister(errorsByCategoryCounter)
	}
}

func publishErrorCounters(diffMetrics *Metrics, cfg OpsLogConfig) {
	metricsConfig := cfg.MetricsConfig

	// Publish detailed error metrics from dedicated storage
	if metricsConfig.TrackErrorsDetailed {
		diffMetrics.ErrorsDetailed.Range(func(key, count any) bool {
			parts := strings.Split(key.(string), "|")
			if len(parts) != 3 {
				log.Warn().Msgf("Invalid key format in ErrorsDetailed: %v", key)
				return true
			}

			user, bucket, status := parts[0], parts[1], parts[2]
			userStr, tenantStr := extractUserAndTenant(user)
			errorCount := float64(count.(*atomic.Uint64).Load())

			// Always publish the metric, even if errorCount is 0
			// This ensures the metric is visible in Prometheus with value 0
			errorsDetailedCounter.With(prometheus.Labels{
				"pod":         cfg.PodName,
				"user":        userStr,
				"tenant":      tenantStr,
				"bucket":      bucket,
				"http_status": status,
			}).Add(errorCount)
			return true
		})
	}

	// Publish per-user error metrics from dedicated storage
	if metricsConfig.TrackErrorsPerUser {
		diffMetrics.ErrorsPerUser.Range(func(key, count any) bool {
			parts := strings.Split(key.(string), "|")
			if len(parts) != 2 {
				log.Warn().Msgf("Invalid key format in ErrorsPerUser: %v", key)
				return true
			}

			user, status := parts[0], parts[1]
			userStr, tenantStr := extractUserAndTenant(user)
			errorCount := float64(count.(*atomic.Uint64).Load())

			// Always publish the metric, even if errorCount is 0
			errorsPerUserCounter.With(prometheus.Labels{
				"pod":         cfg.PodName,
				"user":        userStr,
				"tenant":      tenantStr,
				"http_status": status,
			}).Add(errorCount)
			return true
		})
	}

	// Publish per-bucket error metrics from dedicated storage
	if metricsConfig.TrackErrorsPerBucket {
		diffMetrics.ErrorsPerBucket.Range(func(key, count any) bool {
			parts := strings.Split(key.(string), "|")
			if len(parts) != 3 {
				log.Warn().Msgf("Invalid key format in ErrorsPerBucket: %v", key)
				return true
			}

			tenant, bucket, status := parts[0], parts[1], parts[2]
			errorCount := float64(count.(*atomic.Uint64).Load())

			// Always publish the metric, even if errorCount is 0
			errorsPerBucketCounter.With(prometheus.Labels{
				"pod":         cfg.PodName,
				"tenant":      tenant,
				"bucket":      bucket,
				"http_status": status,
			}).Add(errorCount)
			return true
		})
	}

	// Publish per-tenant error metrics from dedicated storage
	if metricsConfig.TrackErrorsPerTenant {
		diffMetrics.ErrorsPerTenant.Range(func(key, count any) bool {
			parts := strings.Split(key.(string), "|")
			if len(parts) != 2 {
				log.Warn().Msgf("Invalid key format in ErrorsPerTenant: %v", key)
				return true
			}

			tenant, status := parts[0], parts[1]
			errorCount := float64(count.(*atomic.Uint64).Load())

			// Always publish the metric, even if errorCount is 0
			errorsPerTenantCounter.With(prometheus.Labels{
				"pod":         cfg.PodName,
				"tenant":      tenant,
				"http_status": status,
			}).Add(errorCount)
			return true
		})
	}

	// Publish per-status error metrics from dedicated storage
	if metricsConfig.TrackErrorsPerStatus {
		diffMetrics.ErrorsPerStatus.Range(func(key, count any) bool {
			status := key.(string)
			errorCount := float64(count.(*atomic.Uint64).Load())

			// Always publish the metric, even if errorCount is 0
			errorsPerStatusCounter.With(prometheus.Labels{
				"pod":         cfg.PodName,
				"http_status": status,
			}).Add(errorCount)
			return true
		})
	}

	// Publish per-IP error metrics from dedicated storage
	if metricsConfig.TrackErrorsByIP {
		diffMetrics.ErrorsPerIP.Range(func(key, count any) bool {
			parts := strings.Split(key.(string), "|")
			if len(parts) != 3 {
				log.Warn().Msgf("Invalid key format in ErrorsPerIP: %v", key)
				return true
			}

			ip, tenant, status := parts[0], parts[1], parts[2]
			errorCount := float64(count.(*atomic.Uint64).Load())

			// Always publish the metric, even if errorCount is 0
			errorsPerIPCounter.With(prometheus.Labels{
				"pod":         cfg.PodName,
				"ip":          ip,
				"tenant":      tenant,
				"http_status": status,
			}).Add(errorCount)
			return true
		})
	}

	// Publish timeout error metrics
	if metricsConfig.TrackTimeoutErrors {
		diffMetrics.TimeoutErrors.Range(func(key, count any) bool {
			parts := strings.Split(key.(string), "|")
			if len(parts) != 3 {
				log.Warn().Msgf("Invalid key format in TimeoutErrors: %v", key)
				return true
			}

			user, bucket, timeoutType := parts[0], parts[1], parts[2]
			userStr, tenantStr := extractUserAndTenant(user)
			errorCount := float64(count.(*atomic.Uint64).Load())

			// Always publish the metric, even if errorCount is 0
			timeoutErrorsCounter.With(prometheus.Labels{
				"pod":          cfg.PodName,
				"user":         userStr,
				"tenant":       tenantStr,
				"bucket":       bucket,
				"timeout_type": timeoutType,
			}).Add(errorCount)
			return true
		})
	}

	// Publish errors by category
	if metricsConfig.TrackErrorsByCategory {
		diffMetrics.ErrorsByCategory.Range(func(key, count any) bool {
			parts := strings.Split(key.(string), "|")
			if len(parts) != 4 {
				log.Warn().Msgf("Invalid key format in ErrorsByCategory: %v", key)
				return true
			}

			tenant, bucket, category, status := parts[0], parts[1], parts[2], parts[3]
			errorCount := float64(count.(*atomic.Uint64).Load())

			// Always publish the metric, even if errorCount is 0
			errorsByCategoryCounter.With(prometheus.Labels{
				"pod":            cfg.PodName,
				"tenant":         tenant,
				"bucket":         bucket,
				"error_category": category,
				"http_status":    status,
			}).Add(errorCount)
			return true
		})
	}
}

// IsTimeoutError checks if the HTTP status code indicates a timeout error
func IsTimeoutError(status string) bool {
	return status == "408" || // Request Timeout
		status == "504" || // Gateway Timeout
		status == "598" || // Network read timeout error
		status == "499" // Client Closed Request (nginx specific)
}

// GetTimeoutType returns the specific type of timeout error
func GetTimeoutType(status string) string {
	switch status {
	case "408":
		return "request_timeout"
	case "504":
		return "gateway_timeout"
	case "598":
		return "network_read_timeout"
	case "499":
		return "client_closed_request"
	default:
		return "unknown_timeout"
	}
}

// CategorizeHTTPError categorizes HTTP error status codes
func CategorizeHTTPError(status string) string {
	// Check for timeout errors first
	if IsTimeoutError(status) {
		return "timeout"
	}

	// Connection errors
	if status == "502" || status == "503" {
		return "connection"
	}

	// Client errors (4xx)
	if len(status) > 0 && status[0] == '4' {
		return "client"
	}

	// Server errors (5xx)
	if len(status) > 0 && status[0] == '5' {
		return "server"
	}

	return "unknown"
}
