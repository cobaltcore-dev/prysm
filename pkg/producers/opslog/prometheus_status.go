// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and prysm contributors
//
// SPDX-License-Identifier: Apache-2.0

package opslog

import (
	"strings"
	"sync/atomic"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	// Detailed status metrics with user/bucket breakdown
	requestsByStatusDetailedCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "radosgw_requests_by_status_detailed",
			Help: "Number of requests grouped by HTTP status code with full detail",
		},
		[]string{"pod", "user", "tenant", "bucket", "status"},
	)

	// Aggregated status metrics - per user (all buckets combined)
	requestsByStatusPerUserCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "radosgw_requests_by_status_per_user",
			Help: "Number of requests by status aggregated per user (all buckets combined)",
		},
		[]string{"pod", "user", "tenant", "status"},
	)

	// Aggregated status metrics - per bucket (all users combined)
	requestsByStatusPerBucketCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "radosgw_requests_by_status_per_bucket",
			Help: "Number of requests by status aggregated per bucket (all users combined)",
		},
		[]string{"pod", "tenant", "bucket", "status"},
	)

	// Aggregated status metrics - per tenant (all users and buckets combined)
	requestsByStatusPerTenantCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "radosgw_requests_by_status_per_tenant",
			Help: "Number of requests by status aggregated per tenant (all users and buckets combined)",
		},
		[]string{"pod", "tenant", "status"},
	)
)

func registerStatusMetrics(metricsConfig *MetricsConfig) {
	// Register detailed status counter if enabled
	if metricsConfig.TrackRequestsByStatusDetailed {
		prometheus.MustRegister(requestsByStatusDetailedCounter)
	}

	// Conditional registrations for aggregated metrics
	if metricsConfig.TrackRequestsByStatusPerUser {
		prometheus.MustRegister(requestsByStatusPerUserCounter)
	}

	if metricsConfig.TrackRequestsByStatusPerBucket {
		prometheus.MustRegister(requestsByStatusPerBucketCounter)
	}

	if metricsConfig.TrackRequestsByStatusPerTenant {
		prometheus.MustRegister(requestsByStatusPerTenantCounter)
	}
}

func publishStatusMetrics(diffMetrics *Metrics, cfg OpsLogConfig) {
	metricsConfig := cfg.MetricsConfig

	// Detailed status tracking from RequestsByUser
	// Key format: "user|bucket|method|http_status"
	diffMetrics.RequestsByUser.Range(func(key, requestCount any) bool {
		parts := strings.Split(key.(string), "|")
		if len(parts) != 4 {
			return true
		}

		user, bucket, _, httpStatus := parts[0], parts[1], parts[2], parts[3]
		userStr, tenantStr := extractUserAndTenant(user)
		rqCount := float64(requestCount.(*atomic.Uint64).Load())

		if rqCount <= 0 {
			return true
		}

		// Detailed status metric - only if enabled
		if metricsConfig.TrackRequestsByStatusDetailed {
			requestsByStatusDetailedCounter.With(prometheus.Labels{
				"pod":    cfg.PodName,
				"user":   userStr,
				"tenant": tenantStr,
				"bucket": bucket,
				"status": httpStatus,
			}).Add(rqCount)
		}

		// Aggregated metrics based on config
		if metricsConfig.TrackRequestsByStatusPerUser {
			requestsByStatusPerUserCounter.With(prometheus.Labels{
				"pod":    cfg.PodName,
				"user":   userStr,
				"tenant": tenantStr,
				"status": httpStatus,
			}).Add(rqCount)
		}

		if metricsConfig.TrackRequestsByStatusPerBucket {
			requestsByStatusPerBucketCounter.With(prometheus.Labels{
				"pod":    cfg.PodName,
				"tenant": tenantStr,
				"bucket": bucket,
				"status": httpStatus,
			}).Add(rqCount)
		}

		if metricsConfig.TrackRequestsByStatusPerTenant {
			requestsByStatusPerTenantCounter.With(prometheus.Labels{
				"pod":    cfg.PodName,
				"tenant": tenantStr,
				"status": httpStatus,
			}).Add(rqCount)
		}

		return true
	})
}
