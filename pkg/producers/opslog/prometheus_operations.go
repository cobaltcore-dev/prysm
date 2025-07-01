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

// prometheus_operation.go - Operation-based metrics

var (
	// Detailed operation metrics
	requestsByOperationCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "radosgw_requests_by_operation",
			Help: "Number of requests grouped by operation with full detail",
		},
		[]string{"pod", "user", "tenant", "bucket", "operation", "method"},
	)

	// Aggregated operation metrics - per user (all buckets combined)
	requestsByOperationPerUserCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "radosgw_requests_by_operation_per_user",
			Help: "Number of requests by operation aggregated per user (all buckets combined)",
		},
		[]string{"pod", "user", "tenant", "operation", "method"},
	)

	// Aggregated operation metrics - per bucket (all users combined)
	requestsByOperationPerBucketCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "radosgw_requests_by_operation_per_bucket",
			Help: "Number of requests by operation aggregated per bucket (all users combined)",
		},
		[]string{"pod", "tenant", "bucket", "operation", "method"},
	)

	// Aggregated operation metrics - per tenant (all users and buckets combined)
	requestsByOperationPerTenantCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "radosgw_requests_by_operation_per_tenant",
			Help: "Number of requests by operation aggregated per tenant (all users and buckets combined)",
		},
		[]string{"pod", "tenant", "operation", "method"},
	)

	// Global operation metrics (all users, buckets, tenants combined)
	requestsByOperationGlobalCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "radosgw_requests_by_operation_global",
			Help: "Number of requests by operation globally (all users, buckets, tenants combined)",
		},
		[]string{"pod", "operation", "method"},
	)
)

func registerOperationMetrics(metricsConfig *MetricsConfig) {
	// Register detailed operation counter if enabled
	if metricsConfig.TrackRequestsByOperationDetailed {
		prometheus.MustRegister(requestsByOperationCounter)
	}

	// Conditional registrations for aggregated metrics
	if metricsConfig.TrackRequestsByOperationPerUser {
		prometheus.MustRegister(requestsByOperationPerUserCounter)
	}

	if metricsConfig.TrackRequestsByOperationPerBucket {
		prometheus.MustRegister(requestsByOperationPerBucketCounter)
	}

	if metricsConfig.TrackRequestsByOperationPerTenant {
		prometheus.MustRegister(requestsByOperationPerTenantCounter)
	}
	if metricsConfig.TrackRequestsByOperationGlobal {
		prometheus.MustRegister(requestsByOperationGlobalCounter)
	}
}

func publishOperationMetrics(diffMetrics *Metrics, cfg OpsLogConfig) {
	metricsConfig := cfg.MetricsConfig

	// Process RequestsByOperation data
	diffMetrics.RequestsByOperation.Range(func(key, count any) bool {
		parts := strings.Split(key.(string), "|")
		if len(parts) != 4 {
			log.Warn().Msgf("Invalid key format in RequestsByOperation: %v", key)
			return true
		}

		user, bucket, operation, method := parts[0], parts[1], parts[2], parts[3]
		userStr, tenantStr := extractUserAndTenant(user)
		requestCount := float64(count.(*atomic.Uint64).Load())

		if requestCount <= 0 {
			return true
		}

		// Detailed metric - only if enabled
		if metricsConfig.TrackRequestsByOperationDetailed {
			requestsByOperationCounter.With(prometheus.Labels{
				"pod":       cfg.PodName,
				"user":      userStr,
				"tenant":    tenantStr,
				"bucket":    bucket,
				"operation": operation,
				"method":    method,
			}).Add(requestCount)
		}

		// Aggregated metrics based on config
		if metricsConfig.TrackRequestsByOperationPerUser {
			requestsByOperationPerUserCounter.With(prometheus.Labels{
				"pod":       cfg.PodName,
				"user":      userStr,
				"tenant":    tenantStr,
				"operation": operation,
				"method":    method,
			}).Add(requestCount)
		}

		if metricsConfig.TrackRequestsByOperationPerBucket {
			requestsByOperationPerBucketCounter.With(prometheus.Labels{
				"pod":       cfg.PodName,
				"tenant":    tenantStr,
				"bucket":    bucket,
				"operation": operation,
				"method":    method,
			}).Add(requestCount)
		}

		if metricsConfig.TrackRequestsByOperationPerTenant {
			requestsByOperationPerTenantCounter.With(prometheus.Labels{
				"pod":       cfg.PodName,
				"tenant":    tenantStr,
				"operation": operation,
				"method":    method,
			}).Add(requestCount)
		}

		if metricsConfig.TrackRequestsByOperationGlobal {
			requestsByOperationGlobalCounter.With(prometheus.Labels{
				"pod":       cfg.PodName,
				"operation": operation,
				"method":    method,
			}).Add(requestCount)
		}

		return true
	})
}
