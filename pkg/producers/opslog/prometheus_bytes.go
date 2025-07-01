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
	// Detailed bytes metrics
	bytesSentCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "radosgw_bytes_sent",
			Help: "Total bytes sent with full detail",
		},
		[]string{"pod", "user", "tenant", "bucket"},
	)

	bytesReceivedCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "radosgw_bytes_received",
			Help: "Total bytes received with full detail",
		},
		[]string{"pod", "user", "tenant", "bucket"},
	)

	// Aggregated bytes metrics - per user
	bytesSentPerUserCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "radosgw_bytes_sent_per_user",
			Help: "Total bytes sent aggregated per user (all buckets combined)",
		},
		[]string{"pod", "user", "tenant"},
	)

	bytesReceivedPerUserCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "radosgw_bytes_received_per_user",
			Help: "Total bytes received aggregated per user (all buckets combined)",
		},
		[]string{"pod", "user", "tenant"},
	)

	// Aggregated bytes metrics - per bucket
	bytesSentPerBucketCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "radosgw_bytes_sent_per_bucket",
			Help: "Total bytes sent aggregated per bucket (all users combined)",
		},
		[]string{"pod", "tenant", "bucket"},
	)

	bytesReceivedPerBucketCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "radosgw_bytes_received_per_bucket",
			Help: "Total bytes received aggregated per bucket (all users combined)",
		},
		[]string{"pod", "tenant", "bucket"},
	)

	// Aggregated bytes metrics - per tenant
	bytesSentPerTenantCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "radosgw_bytes_sent_per_tenant",
			Help: "Total bytes sent aggregated per tenant (all users and buckets combined)",
		},
		[]string{"pod", "tenant"},
	)

	bytesReceivedPerTenantCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "radosgw_bytes_received_per_tenant",
			Help: "Total bytes received aggregated per tenant (all users and buckets combined)",
		},
		[]string{"pod", "tenant"},
	)
)

func registerBytesMetrics(metricsConfig *MetricsConfig) {
	// Register detailed metrics if enabled
	if metricsConfig.TrackBytesSentDetailed {
		prometheus.MustRegister(bytesSentCounter)
	}

	if metricsConfig.TrackBytesReceivedDetailed {
		prometheus.MustRegister(bytesReceivedCounter)
	}

	// Conditional registrations for aggregated metrics
	if metricsConfig.TrackBytesSentPerUser {
		prometheus.MustRegister(bytesSentPerUserCounter)
	}

	if metricsConfig.TrackBytesReceivedPerUser {
		prometheus.MustRegister(bytesReceivedPerUserCounter)
	}

	if metricsConfig.TrackBytesSentPerBucket {
		prometheus.MustRegister(bytesSentPerBucketCounter)
	}

	if metricsConfig.TrackBytesReceivedPerBucket {
		prometheus.MustRegister(bytesReceivedPerBucketCounter)
	}

	if metricsConfig.TrackBytesSentPerTenant {
		prometheus.MustRegister(bytesSentPerTenantCounter)
	}

	if metricsConfig.TrackBytesReceivedPerTenant {
		prometheus.MustRegister(bytesReceivedPerTenantCounter)
	}
}

// Updated publishing logic for bytes counters
func publishBytesCounters(diffMetrics *Metrics, cfg OpsLogConfig) {
	metricsConfig := cfg.MetricsConfig

	// Process BytesSentByBucket data
	if metricsConfig.TrackBytesSentDetailed || metricsConfig.TrackBytesSentPerUser ||
		metricsConfig.TrackBytesSentPerBucket || metricsConfig.TrackBytesSentPerTenant {
		diffMetrics.BytesSentPerBucket.Range(func(key, bytes any) bool {
			parts := strings.Split(key.(string), "|")
			if len(parts) != 2 {
				log.Warn().Msgf("Invalid key format in BytesSentByBucket: %v", key)
				return true
			}

			user, bucket := parts[0], parts[1]
			userStr, tenantStr := extractUserAndTenant(user)
			totalBytes := float64(bytes.(*atomic.Uint64).Load())

			// Detailed metric
			if metricsConfig.TrackBytesSentDetailed {
				bytesSentCounter.With(prometheus.Labels{
					"pod":    cfg.PodName,
					"user":   userStr,
					"tenant": tenantStr,
					"bucket": bucket,
				}).Add(totalBytes)
			}

			// Aggregated metrics based on config
			if metricsConfig.TrackBytesSentPerUser {
				bytesSentPerUserCounter.With(prometheus.Labels{
					"pod":    cfg.PodName,
					"user":   userStr,
					"tenant": tenantStr,
				}).Add(totalBytes)
			}

			if metricsConfig.TrackBytesSentPerBucket {
				bytesSentPerBucketCounter.With(prometheus.Labels{
					"pod":    cfg.PodName,
					"tenant": tenantStr,
					"bucket": bucket,
				}).Add(totalBytes)
			}

			// Tenant-level aggregation
			if metricsConfig.TrackBytesSentPerTenant {
				bytesSentPerTenantCounter.With(prometheus.Labels{
					"pod":    cfg.PodName,
					"tenant": tenantStr,
				}).Add(totalBytes)
			}

			return true
		})
	}

	// Process BytesReceivedByBucket data
	if metricsConfig.TrackBytesReceivedDetailed || metricsConfig.TrackBytesReceivedPerUser ||
		metricsConfig.TrackBytesReceivedPerBucket || metricsConfig.TrackBytesReceivedPerTenant {
		diffMetrics.BytesReceivedPerBucket.Range(func(key, bytes any) bool {
			parts := strings.Split(key.(string), "|")
			if len(parts) != 2 {
				log.Warn().Msgf("Invalid key format in BytesReceivedByBucket: %v", key)
				return true
			}

			user, bucket := parts[0], parts[1]
			userStr, tenantStr := extractUserAndTenant(user)
			totalBytes := float64(bytes.(*atomic.Uint64).Load())

			// Detailed metric
			if metricsConfig.TrackBytesReceivedDetailed {
				bytesReceivedCounter.With(prometheus.Labels{
					"pod":    cfg.PodName,
					"user":   userStr,
					"tenant": tenantStr,
					"bucket": bucket,
				}).Add(totalBytes)
			}

			// Aggregated metrics based on config
			if metricsConfig.TrackBytesReceivedPerUser {
				bytesReceivedPerUserCounter.With(prometheus.Labels{
					"pod":    cfg.PodName,
					"user":   userStr,
					"tenant": tenantStr,
				}).Add(totalBytes)
			}

			if metricsConfig.TrackBytesReceivedPerBucket {
				bytesReceivedPerBucketCounter.With(prometheus.Labels{
					"pod":    cfg.PodName,
					"tenant": tenantStr,
					"bucket": bucket,
				}).Add(totalBytes)
			}

			// Tenant-level aggregation
			if metricsConfig.TrackBytesReceivedPerTenant {
				bytesReceivedPerTenantCounter.With(prometheus.Labels{
					"pod":    cfg.PodName,
					"tenant": tenantStr,
				}).Add(totalBytes)
			}

			return true
		})
	}

	// Process BytesSentByUser data (if it's different from bucket data)
	if metricsConfig.TrackBytesSentPerUser {
		diffMetrics.BytesSentPerUser.Range(func(user, bytes any) bool {
			userStr, tenantStr := extractUserAndTenant(user.(string))
			totalBytes := float64(bytes.(*atomic.Uint64).Load())

			bytesSentPerUserCounter.With(prometheus.Labels{
				"pod":    cfg.PodName,
				"user":   userStr,
				"tenant": tenantStr,
			}).Add(totalBytes)

			return true
		})
	}

	// Process BytesReceivedByUser data (if it's different from bucket data)
	if metricsConfig.TrackBytesReceivedPerUser {
		diffMetrics.BytesReceivedByUser.Range(func(user, bytes any) bool {
			userStr, tenantStr := extractUserAndTenant(user.(string))
			totalBytes := float64(bytes.(*atomic.Uint64).Load())

			bytesReceivedPerUserCounter.With(prometheus.Labels{
				"pod":    cfg.PodName,
				"user":   userStr,
				"tenant": tenantStr,
			}).Add(totalBytes)

			return true
		})
	}
}
