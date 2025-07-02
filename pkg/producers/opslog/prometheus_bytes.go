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

func publishBytesCounters(diffMetrics *Metrics, cfg OpsLogConfig) {
	metricsConfig := cfg.MetricsConfig

	// Publish detailed bytes sent metrics from dedicated storage
	if metricsConfig.TrackBytesSentDetailed {
		diffMetrics.BytesSentDetailed.Range(func(key, bytes any) bool {
			parts := strings.Split(key.(string), "|")
			if len(parts) != 2 {
				log.Warn().Msgf("Invalid key format in BytesSentDetailed: %v", key)
				return true
			}

			user, bucket := parts[0], parts[1]
			userStr, tenantStr := extractUserAndTenant(user)
			totalBytes := float64(bytes.(*atomic.Uint64).Load())

			if totalBytes > 0 {
				bytesSentCounter.With(prometheus.Labels{
					"pod":    cfg.PodName,
					"user":   userStr,
					"tenant": tenantStr,
					"bucket": bucket,
				}).Add(totalBytes)
			}
			return true
		})
	}

	// Publish detailed bytes received metrics from dedicated storage
	if metricsConfig.TrackBytesReceivedDetailed {
		diffMetrics.BytesReceivedDetailed.Range(func(key, bytes any) bool {
			parts := strings.Split(key.(string), "|")
			if len(parts) != 2 {
				log.Warn().Msgf("Invalid key format in BytesReceivedDetailed: %v", key)
				return true
			}

			user, bucket := parts[0], parts[1]
			userStr, tenantStr := extractUserAndTenant(user)
			totalBytes := float64(bytes.(*atomic.Uint64).Load())

			if totalBytes > 0 {
				bytesReceivedCounter.With(prometheus.Labels{
					"pod":    cfg.PodName,
					"user":   userStr,
					"tenant": tenantStr,
					"bucket": bucket,
				}).Add(totalBytes)
			}
			return true
		})
	}

	// Publish per-user bytes sent metrics from dedicated storage
	if metricsConfig.TrackBytesSentPerUser {
		diffMetrics.BytesSentPerUser.Range(func(key, bytes any) bool {
			user := key.(string)
			userStr, tenantStr := extractUserAndTenant(user)
			totalBytes := float64(bytes.(*atomic.Uint64).Load())

			if totalBytes > 0 {
				bytesSentPerUserCounter.With(prometheus.Labels{
					"pod":    cfg.PodName,
					"user":   userStr,
					"tenant": tenantStr,
				}).Add(totalBytes)
			}
			return true
		})
	}

	// Publish per-user bytes received metrics from dedicated storage
	if metricsConfig.TrackBytesReceivedPerUser {
		diffMetrics.BytesReceivedPerUser.Range(func(key, bytes any) bool {
			user := key.(string)
			userStr, tenantStr := extractUserAndTenant(user)
			totalBytes := float64(bytes.(*atomic.Uint64).Load())

			if totalBytes > 0 {
				bytesReceivedPerUserCounter.With(prometheus.Labels{
					"pod":    cfg.PodName,
					"user":   userStr,
					"tenant": tenantStr,
				}).Add(totalBytes)
			}
			return true
		})
	}

	// Publish per-bucket bytes sent metrics from dedicated storage
	if metricsConfig.TrackBytesSentPerBucket {
		diffMetrics.BytesSentPerBucket.Range(func(key, bytes any) bool {
			parts := strings.Split(key.(string), "|")
			if len(parts) != 2 {
				log.Warn().Msgf("Invalid key format in BytesSentPerBucket: %v", key)
				return true
			}

			tenant, bucket := parts[0], parts[1]
			totalBytes := float64(bytes.(*atomic.Uint64).Load())

			if totalBytes > 0 {
				bytesSentPerBucketCounter.With(prometheus.Labels{
					"pod":    cfg.PodName,
					"tenant": tenant,
					"bucket": bucket,
				}).Add(totalBytes)
			}
			return true
		})
	}

	// Publish per-bucket bytes received metrics from dedicated storage
	if metricsConfig.TrackBytesReceivedPerBucket {
		diffMetrics.BytesReceivedPerBucket.Range(func(key, bytes any) bool {
			parts := strings.Split(key.(string), "|")
			if len(parts) != 2 {
				log.Warn().Msgf("Invalid key format in BytesReceivedPerBucket: %v", key)
				return true
			}

			tenant, bucket := parts[0], parts[1]
			totalBytes := float64(bytes.(*atomic.Uint64).Load())

			if totalBytes > 0 {
				bytesReceivedPerBucketCounter.With(prometheus.Labels{
					"pod":    cfg.PodName,
					"tenant": tenant,
					"bucket": bucket,
				}).Add(totalBytes)
			}
			return true
		})
	}

	// Publish per-tenant bytes sent metrics from dedicated storage
	if metricsConfig.TrackBytesSentPerTenant {
		diffMetrics.BytesSentPerTenant.Range(func(key, bytes any) bool {
			tenant := key.(string)
			totalBytes := float64(bytes.(*atomic.Uint64).Load())

			if totalBytes > 0 {
				bytesSentPerTenantCounter.With(prometheus.Labels{
					"pod":    cfg.PodName,
					"tenant": tenant,
				}).Add(totalBytes)
			}
			return true
		})
	}

	// Publish per-tenant bytes received metrics from dedicated storage
	if metricsConfig.TrackBytesReceivedPerTenant {
		diffMetrics.BytesReceivedPerTenant.Range(func(key, bytes any) bool {
			tenant := key.(string)
			totalBytes := float64(bytes.(*atomic.Uint64).Load())

			if totalBytes > 0 {
				bytesReceivedPerTenantCounter.With(prometheus.Labels{
					"pod":    cfg.PodName,
					"tenant": tenant,
				}).Add(totalBytes)
			}
			return true
		})
	}
}
