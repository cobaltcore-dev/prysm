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
	// Detailed method metrics
	requestsByMethodCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "radosgw_requests_by_method",
			Help: "Number of requests grouped by HTTP method with full detail",
		},
		[]string{"pod", "user", "tenant", "bucket", "method"},
	)

	// Aggregated method metrics - per user (all buckets combined)
	requestsByMethodPerUserCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "radosgw_requests_by_method_per_user",
			Help: "Number of requests by method aggregated per user (all buckets combined)",
		},
		[]string{"pod", "user", "tenant", "method"},
	)

	// Aggregated method metrics - per bucket (all users combined)
	requestsByMethodPerBucketCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "radosgw_requests_by_method_per_bucket",
			Help: "Number of requests by method aggregated per bucket (all users combined)",
		},
		[]string{"pod", "tenant", "bucket", "method"},
	)

	// Aggregated method metrics - per tenant (all users and buckets combined)
	requestsByMethodPerTenantCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "radosgw_requests_by_method_per_tenant",
			Help: "Number of requests by method aggregated per tenant (all users and buckets combined)",
		},
		[]string{"pod", "tenant", "method"},
	)

	// Global method metrics (all users, buckets, tenants combined)
	requestsByMethodGlobalCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "radosgw_requests_by_method_global",
			Help: "Number of requests by method globally (all users, buckets, tenants combined)",
		},
		[]string{"pod", "method"},
	)
)

func registerMethodMetrics(metricsConfig *MetricsConfig) {
	// Register detailed method counter if enabled
	if metricsConfig.TrackRequestsByMethodDetailed {
		prometheus.MustRegister(requestsByMethodCounter)
	}

	// Conditional registrations for aggregated metrics
	if metricsConfig.TrackRequestsByMethodPerUser {
		prometheus.MustRegister(requestsByMethodPerUserCounter)
	}

	if metricsConfig.TrackRequestsByMethodPerBucket {
		prometheus.MustRegister(requestsByMethodPerBucketCounter)
	}

	if metricsConfig.TrackRequestsByMethodPerTenant {
		prometheus.MustRegister(requestsByMethodPerTenantCounter)
	}

	if metricsConfig.TrackRequestsByMethodGlobal {
		prometheus.MustRegister(requestsByMethodGlobalCounter)
	}
}

func publishMethodMetrics(diffMetrics *Metrics, cfg OpsLogConfig) {
	metricsConfig := cfg.MetricsConfig

	// Process RequestsByMethod data
	// Key format: "user|bucket|method"
	diffMetrics.RequestsByMethod.Range(func(key, count any) bool {
		parts := strings.Split(key.(string), "|")
		if len(parts) != 3 {
			log.Warn().Msgf("Invalid key format in RequestsByMethod: %v", key)
			return true
		}

		user, bucket, method := parts[0], parts[1], parts[2]
		userStr, tenantStr := extractUserAndTenant(user)
		requestCount := float64(count.(*atomic.Uint64).Load())

		if requestCount <= 0 {
			return true
		}

		// Detailed metric - only if enabled
		if metricsConfig.TrackRequestsByMethodDetailed {
			requestsByMethodCounter.With(prometheus.Labels{
				"pod":    cfg.PodName,
				"user":   userStr,
				"tenant": tenantStr,
				"bucket": bucket,
				"method": method,
			}).Add(requestCount)
		}

		// Aggregated metrics based on config
		if metricsConfig.TrackRequestsByMethodPerUser {
			requestsByMethodPerUserCounter.With(prometheus.Labels{
				"pod":    cfg.PodName,
				"user":   userStr,
				"tenant": tenantStr,
				"method": method,
			}).Add(requestCount)
		}

		if metricsConfig.TrackRequestsByMethodPerBucket {
			requestsByMethodPerBucketCounter.With(prometheus.Labels{
				"pod":    cfg.PodName,
				"tenant": tenantStr,
				"bucket": bucket,
				"method": method,
			}).Add(requestCount)
		}

		if metricsConfig.TrackRequestsByMethodPerTenant {
			requestsByMethodPerTenantCounter.With(prometheus.Labels{
				"pod":    cfg.PodName,
				"tenant": tenantStr,
				"method": method,
			}).Add(requestCount)
		}

		if metricsConfig.TrackRequestsByMethodGlobal {
			requestsByMethodGlobalCounter.With(prometheus.Labels{
				"pod":    cfg.PodName,
				"method": method,
			}).Add(requestCount)
		}

		return true
	})
}
