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

	// Publish detailed status metrics from dedicated storage
	if metricsConfig.TrackRequestsByStatusDetailed {
		diffMetrics.RequestsByStatusDetailed.Range(func(key, count any) bool {
			parts := strings.Split(key.(string), "|")
			if len(parts) != 3 {
				return true
			}

			user, bucket, httpStatus := parts[0], parts[1], parts[2]
			userStr, tenantStr := extractUserAndTenant(user)
			requestCount := float64(count.(*atomic.Uint64).Load())

			if requestCount > 0 {
				requestsByStatusDetailedCounter.With(prometheus.Labels{
					"pod":    cfg.PodName,
					"user":   userStr,
					"tenant": tenantStr,
					"bucket": bucket,
					"status": httpStatus,
				}).Add(requestCount)
			}
			return true
		})
	}

	// Publish per-user status metrics from dedicated storage
	if metricsConfig.TrackRequestsByStatusPerUser {
		diffMetrics.RequestsByStatusPerUser.Range(func(key, count any) bool {
			parts := strings.Split(key.(string), "|")
			if len(parts) != 2 {
				return true
			}

			user, httpStatus := parts[0], parts[1]
			userStr, tenantStr := extractUserAndTenant(user)
			requestCount := float64(count.(*atomic.Uint64).Load())

			if requestCount > 0 {
				requestsByStatusPerUserCounter.With(prometheus.Labels{
					"pod":    cfg.PodName,
					"user":   userStr,
					"tenant": tenantStr,
					"status": httpStatus,
				}).Add(requestCount)
			}
			return true
		})
	}

	// Publish per-bucket status metrics from dedicated storage
	if metricsConfig.TrackRequestsByStatusPerBucket {
		diffMetrics.RequestsByStatusPerBucket.Range(func(key, count any) bool {
			parts := strings.Split(key.(string), "|")
			if len(parts) != 3 {
				return true
			}

			tenant, bucket, httpStatus := parts[0], parts[1], parts[2]
			requestCount := float64(count.(*atomic.Uint64).Load())

			if requestCount > 0 {
				requestsByStatusPerBucketCounter.With(prometheus.Labels{
					"pod":    cfg.PodName,
					"tenant": tenant,
					"bucket": bucket,
					"status": httpStatus,
				}).Add(requestCount)
			}
			return true
		})
	}

	// Publish per-tenant status metrics from dedicated storage
	if metricsConfig.TrackRequestsByStatusPerTenant {
		diffMetrics.RequestsByStatusPerTenant.Range(func(key, count any) bool {
			parts := strings.Split(key.(string), "|")
			if len(parts) != 2 {
				return true
			}

			tenant, httpStatus := parts[0], parts[1]
			requestCount := float64(count.(*atomic.Uint64).Load())

			if requestCount > 0 {
				requestsByStatusPerTenantCounter.With(prometheus.Labels{
					"pod":    cfg.PodName,
					"tenant": tenant,
					"status": httpStatus,
				}).Add(requestCount)
			}
			return true
		})
	}
}
