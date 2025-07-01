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

	// Track detailed requests by IP and user - only if enabled
	if metricsConfig.TrackRequestsByIPDetailed {

		currentMetrics.RequestsByIP.Range(func(key, count any) bool {
			parts := strings.Split(key.(string), "|")
			if len(parts) != 2 {
				log.Warn().Msgf("Invalid key format in RequestsByIP: %v", key)
				return true
			}

			user, ip := parts[0], parts[1]
			userStr, tenantStr := extractUserAndTenant(user)
			requestCount := float64(count.(*atomic.Uint64).Load())

			requestsByIPGauge.With(prometheus.Labels{
				"pod":    cfg.PodName,
				"user":   userStr,
				"tenant": tenantStr,
				"ip":     ip,
			}).Set(requestCount)

			return true
		})
	}

	// Track aggregated requests per IP - only if enabled
	if metricsConfig.TrackRequestsByIPPerTenant {
		ipTenantMap := make(map[string]map[string]float64)
		currentMetrics.RequestsByIP.Range(func(key, count any) bool {
			parts := strings.Split(key.(string), "|")
			if len(parts) != 2 {
				return true
			}

			user, ip := parts[0], parts[1]
			_, tenantStr := extractUserAndTenant(user)
			requestCount := float64(count.(*atomic.Uint64).Load())

			if ipTenantMap[ip] == nil {
				ipTenantMap[ip] = make(map[string]float64)
			}
			ipTenantMap[ip][tenantStr] += requestCount

			return true
		})

		for ip, tenantCounts := range ipTenantMap {
			for tenant, count := range tenantCounts {
				requestsPerIPGauge.With(prometheus.Labels{
					"pod":    cfg.PodName,
					"tenant": tenant,
					"ip":     ip,
				}).Set(count)
			}
		}
	}

	// Track global tenant aggregation - only if enabled
	if metricsConfig.TrackRequestsByIPGlobalPerTenant {
		tenantTotalMap := make(map[string]float64)
		currentMetrics.RequestsByIP.Range(func(key, count any) bool {
			parts := strings.Split(key.(string), "|")
			if len(parts) != 2 {
				return true
			}

			user, _ := parts[0], parts[1]
			_, tenantStr := extractUserAndTenant(user)
			requestCount := float64(count.(*atomic.Uint64).Load())

			tenantTotalMap[tenantStr] += requestCount

			return true
		})

		for tenant, totalCount := range tenantTotalMap {
			requestsPerTenantFromIPGauge.With(prometheus.Labels{
				"pod":    cfg.PodName,
				"tenant": tenant,
			}).Set(totalCount)
		}
	}

	// Track requests by IP, bucket, method, tenant - independent
	if metricsConfig.TrackRequestsByIPBucketMethodTenant {
		currentMetrics.RequestsByIPBucketMethodTenant.Range(func(key, count any) bool {
			parts := strings.Split(key.(string), "|")
			if len(parts) != 4 {
				log.Warn().Msgf("Invalid key format in RequestsByIPBucketMethodTenant: %v", key)
				return true
			}

			ip, bucket, method, user := parts[0], parts[1], parts[2], parts[3]
			_, tenantStr := extractUserAndTenant(user)
			requestCount := float64(count.(*atomic.Uint64).Load())

			requestsByIPBucketMethodTenantGauge.With(prometheus.Labels{
				"pod":    cfg.PodName,
				"ip":     ip,
				"bucket": bucket,
				"method": method,
				"tenant": tenantStr,
			}).Set(requestCount)

			return true
		})
	}

	// Track bytes sent by IP - only if enabled
	if metricsConfig.TrackBytesSentByIPDetailed {

		currentMetrics.BytesSentByIP.Range(func(key, bytesSent any) bool {
			parts := strings.Split(key.(string), "|")
			if len(parts) != 2 {
				log.Warn().Msgf("Invalid key format in BytesSentByIP: %v", key)
				return true
			}

			user, ip := parts[0], parts[1]
			userStr, tenantStr := extractUserAndTenant(user)
			totalBytesSent := float64(bytesSent.(*atomic.Uint64).Load())

			bytesSentByIPGauge.With(prometheus.Labels{
				"pod":    cfg.PodName,
				"user":   userStr,
				"tenant": tenantStr,
				"ip":     ip,
			}).Set(totalBytesSent)

			return true
		})
	}

	// Track aggregated bytes sent per IP - only if enabled
	if metricsConfig.TrackBytesSentByIPPerTenant {
		ipTenantBytesMap := make(map[string]map[string]float64)
		currentMetrics.BytesSentByIP.Range(func(key, bytesSent any) bool {
			parts := strings.Split(key.(string), "|")
			if len(parts) != 2 {
				return true
			}

			user, ip := parts[0], parts[1]
			_, tenantStr := extractUserAndTenant(user)
			totalBytesSent := float64(bytesSent.(*atomic.Uint64).Load())

			if ipTenantBytesMap[ip] == nil {
				ipTenantBytesMap[ip] = make(map[string]float64)
			}
			ipTenantBytesMap[ip][tenantStr] += totalBytesSent

			return true
		})

		for ip, tenantBytes := range ipTenantBytesMap {
			for tenant, bytes := range tenantBytes {
				bytesSentPerIPGauge.With(prometheus.Labels{
					"pod":    cfg.PodName,
					"tenant": tenant,
					"ip":     ip,
				}).Set(bytes)
			}
		}
	}

	// Track global tenant aggregation for bytes sent - only if enabled
	if metricsConfig.TrackBytesSentByIPGlobalPerTenant {
		tenantTotalBytes := make(map[string]float64)
		currentMetrics.BytesSentByIP.Range(func(key, bytesSent any) bool {
			parts := strings.Split(key.(string), "|")
			if len(parts) != 2 {
				return true
			}

			user, _ := parts[0], parts[1]
			_, tenantStr := extractUserAndTenant(user)
			totalBytesSent := float64(bytesSent.(*atomic.Uint64).Load())

			tenantTotalBytes[tenantStr] += totalBytesSent

			return true
		})

		for tenant, totalBytes := range tenantTotalBytes {
			bytesSentPerTenantFromIPGauge.With(prometheus.Labels{
				"pod":    cfg.PodName,
				"tenant": tenant,
			}).Set(totalBytes)
		}
	}

	// Track bytes received by IP - only if enabled
	if metricsConfig.TrackBytesReceivedByIPDetailed {

		currentMetrics.BytesReceivedByIP.Range(func(key, bytesReceived any) bool {
			parts := strings.Split(key.(string), "|")
			if len(parts) != 2 {
				log.Warn().Msgf("Invalid key format in BytesReceivedByIP: %v", key)
				return true
			}

			user, ip := parts[0], parts[1]
			userStr, tenantStr := extractUserAndTenant(user)
			totalBytesReceived := float64(bytesReceived.(*atomic.Uint64).Load())

			bytesReceivedByIPGauge.With(prometheus.Labels{
				"pod":    cfg.PodName,
				"user":   userStr,
				"tenant": tenantStr,
				"ip":     ip,
			}).Set(totalBytesReceived)

			return true
		})
	}

	// Track aggregated bytes received per IP - only if enabled
	if metricsConfig.TrackBytesReceivedByIPPerTenant {
		ipTenantReceivedMap := make(map[string]map[string]float64)
		currentMetrics.BytesReceivedByIP.Range(func(key, bytesReceived any) bool {
			parts := strings.Split(key.(string), "|")
			if len(parts) != 2 {
				return true
			}

			user, ip := parts[0], parts[1]
			_, tenantStr := extractUserAndTenant(user)
			totalBytesReceived := float64(bytesReceived.(*atomic.Uint64).Load())

			if ipTenantReceivedMap[ip] == nil {
				ipTenantReceivedMap[ip] = make(map[string]float64)
			}
			ipTenantReceivedMap[ip][tenantStr] += totalBytesReceived

			return true
		})

		for ip, tenantBytes := range ipTenantReceivedMap {
			for tenant, bytes := range tenantBytes {
				bytesReceivedPerIPGauge.With(prometheus.Labels{
					"pod":    cfg.PodName,
					"tenant": tenant,
					"ip":     ip,
				}).Set(bytes)
			}
		}
	}

	// Track global tenant aggregation for bytes received - only if enabled
	if metricsConfig.TrackBytesReceivedByIPGlobalPerTenant {
		tenantTotalReceived := make(map[string]float64)
		currentMetrics.BytesReceivedByIP.Range(func(key, bytesReceived any) bool {
			parts := strings.Split(key.(string), "|")
			if len(parts) != 2 {
				return true
			}

			user, _ := parts[0], parts[1]
			_, tenantStr := extractUserAndTenant(user)
			totalBytesReceived := float64(bytesReceived.(*atomic.Uint64).Load())

			tenantTotalReceived[tenantStr] += totalBytesReceived

			return true
		})

		for tenant, totalBytes := range tenantTotalReceived {
			bytesReceivedPerTenantFromIPGauge.With(prometheus.Labels{
				"pod":    cfg.PodName,
				"tenant": tenant,
			}).Set(totalBytes)
		}
	}
}
