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
	// Detailed IP-based gauges
	requestsByIPGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "radosgw_requests_by_ip",
			Help: "Total number of requests per IP and user",
		},
		[]string{"pod", "user", "tenant", "ip"},
	)

	requestsByIPBucketMethodTenantGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "radosgw_requests_by_ip_bucket_method_tenant",
			Help: "Total requests grouped by IP, bucket, method, and tenant",
		},
		[]string{"pod", "ip", "bucket", "method", "tenant"},
	)

	bytesSentByIPGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "radosgw_bytes_sent_by_ip",
			Help: "Total bytes sent per IP and user",
		},
		[]string{"pod", "user", "tenant", "ip"},
	)

	bytesReceivedByIPGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "radosgw_bytes_received_by_ip",
			Help: "Total bytes received per IP and user",
		},
		[]string{"pod", "user", "tenant", "ip"},
	)

	// Aggregated IP-based gauges - per IP (all users combined)
	requestsPerIPGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "radosgw_requests_per_ip",
			Help: "Total requests aggregated per IP (all users combined)",
		},
		[]string{"pod", "tenant", "ip"},
	)

	bytesSentPerIPGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "radosgw_bytes_sent_per_ip",
			Help: "Total bytes sent aggregated per IP (all users combined)",
		},
		[]string{"pod", "tenant", "ip"},
	)

	bytesReceivedPerIPGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "radosgw_bytes_received_per_ip",
			Help: "Total bytes received aggregated per IP (all users combined)",
		},
		[]string{"pod", "tenant", "ip"},
	)

	// Aggregated IP-based gauges - per tenant (all IPs combined)
	requestsPerTenantFromIPGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "radosgw_requests_per_tenant_from_ip",
			Help: "Total requests aggregated per tenant from all IPs",
		},
		[]string{"pod", "tenant"},
	)

	bytesSentPerTenantFromIPGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "radosgw_bytes_sent_per_tenant_from_ip",
			Help: "Total bytes sent aggregated per tenant from all IPs",
		},
		[]string{"pod", "tenant"},
	)

	bytesReceivedPerTenantFromIPGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "radosgw_bytes_received_per_tenant_from_ip",
			Help: "Total bytes received aggregated per tenant from all IPs",
		},
		[]string{"pod", "tenant"},
	)
)

func registerIPMetrics(metricsConfig *MetricsConfig) {

	// Independent registrations for each flag
	if metricsConfig.TrackRequestsByIPDetailed {
		prometheus.MustRegister(requestsByIPGauge)
	}

	if metricsConfig.TrackRequestsByIPPerTenant {
		prometheus.MustRegister(requestsPerIPGauge)
	}

	if metricsConfig.TrackRequestsByIPBucketMethodTenant {
		prometheus.MustRegister(requestsByIPBucketMethodTenantGauge)
	}

	if metricsConfig.TrackRequestsByIPGlobalPerTenant {
		prometheus.MustRegister(requestsPerTenantFromIPGauge)
	}

	if metricsConfig.TrackBytesSentByIPDetailed {
		prometheus.MustRegister(bytesSentByIPGauge)
	}

	if metricsConfig.TrackBytesSentByIPPerTenant {
		prometheus.MustRegister(bytesSentPerIPGauge)
	}

	if metricsConfig.TrackBytesSentByIPGlobalPerTenant {
		prometheus.MustRegister(bytesSentPerTenantFromIPGauge)
	}
	if metricsConfig.TrackBytesReceivedByIPDetailed {
		prometheus.MustRegister(bytesReceivedByIPGauge)

	}
	if metricsConfig.TrackBytesReceivedByIPPerTenant {
		prometheus.MustRegister(bytesReceivedPerIPGauge)
	}

	if metricsConfig.TrackBytesReceivedByIPGlobalPerTenant {
		prometheus.MustRegister(bytesReceivedPerTenantFromIPGauge)
	}
}

func publishIPGauges(currentMetrics *Metrics, cfg OpsLogConfig) {
	metricsConfig := cfg.MetricsConfig

	// Publish detailed requests by IP from dedicated storage
	if metricsConfig.TrackRequestsByIPDetailed {
		currentMetrics.RequestsByIPDetailed.Range(func(key, count any) bool {
			parts := strings.Split(key.(string), "|")
			if len(parts) != 2 {
				log.Warn().Msgf("Invalid key format in RequestsByIPDetailed: %v", key)
				return true
			}

			user, ip := parts[0], parts[1]
			userStr, tenantStr := extractUserAndTenant(user)
			requestCount := float64(count.(*atomic.Uint64).Load())

			if requestCount > 0 {
				requestsByIPGauge.With(prometheus.Labels{
					"pod":    cfg.PodName,
					"user":   userStr,
					"tenant": tenantStr,
					"ip":     ip,
				}).Set(requestCount)
			}
			return true
		})
	}

	// Publish aggregated requests per IP per tenant from dedicated storage
	if metricsConfig.TrackRequestsByIPPerTenant {
		currentMetrics.RequestsPerIPPerTenant.Range(func(key, count any) bool {
			parts := strings.Split(key.(string), "|")
			if len(parts) != 2 {
				log.Warn().Msgf("Invalid key format in RequestsPerIPPerTenant: %v", key)
				return true
			}

			tenant, ip := parts[0], parts[1]
			requestCount := float64(count.(*atomic.Uint64).Load())

			if requestCount > 0 {
				requestsPerIPGauge.With(prometheus.Labels{
					"pod":    cfg.PodName,
					"tenant": tenant,
					"ip":     ip,
				}).Set(requestCount)
			}
			return true
		})
	}

	// Publish global tenant aggregation from dedicated storage
	if metricsConfig.TrackRequestsByIPGlobalPerTenant {
		currentMetrics.RequestsPerTenantFromIP.Range(func(key, count any) bool {
			tenant := key.(string)
			requestCount := float64(count.(*atomic.Uint64).Load())

			if requestCount > 0 {
				requestsPerTenantFromIPGauge.With(prometheus.Labels{
					"pod":    cfg.PodName,
					"tenant": tenant,
				}).Set(requestCount)
			}
			return true
		})
	}

	// Publish requests by IP, bucket, method, tenant from dedicated storage
	if metricsConfig.TrackRequestsByIPBucketMethodTenant {
		currentMetrics.RequestsByIPBucketMethodTenant.Range(func(key, count any) bool {
			parts := strings.Split(key.(string), "|")
			if len(parts) != 4 {
				log.Warn().Msgf("Invalid key format in RequestsByIPBucketMethodTenant: %v", key)
				return true
			}

			ip, bucket, method, tenant := parts[0], parts[1], parts[2], parts[3]
			requestCount := float64(count.(*atomic.Uint64).Load())

			if requestCount > 0 {
				requestsByIPBucketMethodTenantGauge.With(prometheus.Labels{
					"pod":    cfg.PodName,
					"ip":     ip,
					"bucket": bucket,
					"method": method,
					"tenant": tenant,
				}).Set(requestCount)
			}
			return true
		})
	}

	// Publish detailed bytes sent by IP from dedicated storage
	if metricsConfig.TrackBytesSentByIPDetailed {
		currentMetrics.BytesSentByIPDetailed.Range(func(key, bytes any) bool {
			parts := strings.Split(key.(string), "|")
			if len(parts) != 2 {
				log.Warn().Msgf("Invalid key format in BytesSentByIPDetailed: %v", key)
				return true
			}

			user, ip := parts[0], parts[1]
			userStr, tenantStr := extractUserAndTenant(user)
			totalBytes := float64(bytes.(*atomic.Uint64).Load())

			if totalBytes > 0 {
				bytesSentByIPGauge.With(prometheus.Labels{
					"pod":    cfg.PodName,
					"user":   userStr,
					"tenant": tenantStr,
					"ip":     ip,
				}).Set(totalBytes)
			}
			return true
		})
	}

	// Publish aggregated bytes sent per IP per tenant from dedicated storage
	if metricsConfig.TrackBytesSentByIPPerTenant {
		currentMetrics.BytesSentPerIPPerTenant.Range(func(key, bytes any) bool {
			parts := strings.Split(key.(string), "|")
			if len(parts) != 2 {
				log.Warn().Msgf("Invalid key format in BytesSentPerIPPerTenant: %v", key)
				return true
			}

			tenant, ip := parts[0], parts[1]
			totalBytes := float64(bytes.(*atomic.Uint64).Load())

			if totalBytes > 0 {
				bytesSentPerIPGauge.With(prometheus.Labels{
					"pod":    cfg.PodName,
					"tenant": tenant,
					"ip":     ip,
				}).Set(totalBytes)
			}
			return true
		})
	}

	// Publish global tenant bytes sent aggregation from dedicated storage
	if metricsConfig.TrackBytesSentByIPGlobalPerTenant {
		currentMetrics.BytesSentPerTenantFromIP.Range(func(key, bytes any) bool {
			tenant := key.(string)
			totalBytes := float64(bytes.(*atomic.Uint64).Load())

			if totalBytes > 0 {
				bytesSentPerTenantFromIPGauge.With(prometheus.Labels{
					"pod":    cfg.PodName,
					"tenant": tenant,
				}).Set(totalBytes)
			}
			return true
		})
	}

	// Publish detailed bytes received by IP from dedicated storage
	if metricsConfig.TrackBytesReceivedByIPDetailed {
		currentMetrics.BytesReceivedByIPDetailed.Range(func(key, bytes any) bool {
			parts := strings.Split(key.(string), "|")
			if len(parts) != 2 {
				log.Warn().Msgf("Invalid key format in BytesReceivedByIPDetailed: %v", key)
				return true
			}

			user, ip := parts[0], parts[1]
			userStr, tenantStr := extractUserAndTenant(user)
			totalBytes := float64(bytes.(*atomic.Uint64).Load())

			if totalBytes > 0 {
				bytesReceivedByIPGauge.With(prometheus.Labels{
					"pod":    cfg.PodName,
					"user":   userStr,
					"tenant": tenantStr,
					"ip":     ip,
				}).Set(totalBytes)
			}
			return true
		})
	}

	// Publish aggregated bytes received per IP per tenant from dedicated storage
	if metricsConfig.TrackBytesReceivedByIPPerTenant {
		currentMetrics.BytesReceivedPerIPPerTenant.Range(func(key, bytes any) bool {
			parts := strings.Split(key.(string), "|")
			if len(parts) != 2 {
				log.Warn().Msgf("Invalid key format in BytesReceivedPerIPPerTenant: %v", key)
				return true
			}

			tenant, ip := parts[0], parts[1]
			totalBytes := float64(bytes.(*atomic.Uint64).Load())

			if totalBytes > 0 {
				bytesReceivedPerIPGauge.With(prometheus.Labels{
					"pod":    cfg.PodName,
					"tenant": tenant,
					"ip":     ip,
				}).Set(totalBytes)
			}
			return true
		})
	}

	// Publish global tenant bytes received aggregation from dedicated storage
	if metricsConfig.TrackBytesReceivedByIPGlobalPerTenant {
		currentMetrics.BytesReceivedPerTenantFromIP.Range(func(key, bytes any) bool {
			tenant := key.(string)
			totalBytes := float64(bytes.(*atomic.Uint64).Load())

			if totalBytes > 0 {
				bytesReceivedPerTenantFromIPGauge.With(prometheus.Labels{
					"pod":    cfg.PodName,
					"tenant": tenant,
				}).Set(totalBytes)
			}
			return true
		})
	}
}
