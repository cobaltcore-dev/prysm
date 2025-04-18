// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and prysm contributors
//
// SPDX-License-Identifier: Apache-2.0

package opslog

import (
	"fmt"
	"net/http"
	"strings"
	"sync/atomic"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog/log"
)

var (
	previousMetrics *Metrics = nil

	// Total requests grouped by user and bucket
	totalRequestsCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "radosgw_total_requests",
			Help: "Total number of requests processed",
		},
		[]string{"pod", "user", "tenant", "bucket", "method", "http_status"},
	)

	// Requests grouped by HTTP method (GET, PUT, POST, DELETE, etc.)
	requestsByMethodCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "radosgw_requests_by_method",
			Help: "Number of requests grouped by HTTP method (GET, PUT, DELETE, etc.)",
		},
		[]string{"pod", "user", "tenant", "bucket", "method"},
	)

	// Requests grouped by operation
	requestsByOperationCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "radosgw_requests_by_operation",
			Help: "Number of requests grouped by operation",
		},
		[]string{"pod", "user", "tenant", "bucket", "operation", "method"},
	)

	// Requests grouped by HTTP status code (200, 404, 500, etc.)
	requestsByStatusCodeCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "radosgw_requests_by_status",
			Help: "Number of requests grouped by HTTP status code",
		},
		[]string{"pod", "user", "tenant", "bucket", "status"},
	)

	// Bytes sent grouped by user and bucket
	bytesSentCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "radosgw_bytes_sent",
			Help: "Total bytes sent",
		},
		[]string{"pod", "user", "tenant", "bucket"},
	)

	// Bytes received grouped by user and bucket
	bytesReceivedCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "radosgw_bytes_received",
			Help: "Total bytes received",
		},
		[]string{"pod", "user", "tenant", "bucket"},
	)

	// Count of request errors grouped by user and bucket
	errorsCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "radosgw_errors_total",
			Help: "Total number of errors",
		},
		[]string{"pod", "user", "tenant", "bucket", "http_status"},
	)

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

	// HTTP errors by user and IP
	httpErrorsByUserCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "radosgw_http_errors_by_user",
			Help: "Total HTTP errors by user and bucket",
		},
		[]string{"pod", "user", "tenant", "bucket", "http_status"},
	)

	httpErrorsByIPCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "radosgw_http_errors_by_ip",
			Help: "Total HTTP errors by IP and bucket",
		},
		[]string{"pod", "bucket", "ip", "http_status"},
	)

	// Histogram for request duration
	requestsDurationHistogram = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "radosgw_requests_duration",
			Help:    "Histogram for request latencies",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"user", "tenant", "bucket", "method"},
		// []string{"pod", "user", "tenant", "bucket", "method"},
	)
)

func initPrometheusSettings(metricsConfig *MetricsConfig) {
	// Register core Prometheus metrics
	prometheus.MustRegister(totalRequestsCounter)

	// Conditional registrations based on config
	if metricsConfig.TrackRequestsByMethod {
		prometheus.MustRegister(requestsByMethodCounter)
	}

	if metricsConfig.TrackRequestsByOperation {
		prometheus.MustRegister(requestsByOperationCounter)
	}

	if metricsConfig.TrackRequestsByStatus {
		prometheus.MustRegister(requestsByStatusCodeCounter)
	}

	if metricsConfig.TrackBytesSentByUser || metricsConfig.TrackBytesSentByBucket {
		prometheus.MustRegister(bytesSentCounter)
	}

	if metricsConfig.TrackBytesReceivedByUser || metricsConfig.TrackBytesReceivedByBucket {
		prometheus.MustRegister(bytesReceivedCounter)
	}

	if metricsConfig.TrackErrorsByUser || metricsConfig.TrackErrorsByIP {
		prometheus.MustRegister(errorsCounter)
	}

	// Conditional IP tracking
	if metricsConfig.TrackRequestsByIP {
		prometheus.MustRegister(requestsByIPGauge)
	}
	if metricsConfig.TrackRequestsByIPBucketMethodTenant {
		prometheus.MustRegister(requestsByIPBucketMethodTenantGauge)
	}
	if metricsConfig.TrackBytesSentByIP {
		prometheus.MustRegister(bytesSentByIPGauge)
	}
	if metricsConfig.TrackBytesReceivedByIP {
		prometheus.MustRegister(bytesReceivedByIPGauge)
	}

	// Conditional error tracking
	if metricsConfig.TrackErrorsByUser {
		prometheus.MustRegister(httpErrorsByUserCounter)
	}

	if metricsConfig.TrackErrorsByIP {
		prometheus.MustRegister(httpErrorsByIPCounter)
	}

	if metricsConfig.TrackLatencyByMethod ||
		metricsConfig.TrackLatencyByBucket ||
		metricsConfig.TrackLatencyByTenant ||
		metricsConfig.TrackLatencyByUser ||
		metricsConfig.TrackLatencyByBucketAndMethod {
		prometheus.MustRegister(requestsDurationHistogram)
	}
}

// PublishToPrometheus updates Prometheus metrics from aggregated data
func PublishToPrometheus(totalMetrics *Metrics, cfg OpsLogConfig) {
	metricsConfig := cfg.MetricsConfig

	// On first call, just store the snapshot and return
	if previousMetrics == nil {
		previousMetrics = totalMetrics.Clone()
		return
	}

	// Snapshot current total
	currentMetrics := totalMetrics.Clone()

	// Compute diff (delta) against last snapshot
	diffMetrics := SubtractMetrics(currentMetrics, previousMetrics)

	// Update snapshot for next interval
	previousMetrics = currentMetrics

	if metricsConfig.TrackRequestsByUser {
		// Total requests grouped by user, bucket, method, and status
		diffMetrics.RequestsByUser.Range(func(key, requestCount any) bool {
			parts := strings.Split(key.(string), "|")
			if len(parts) != 4 { // Expecting user | bucket | method | http_status
				log.Warn().Msgf("Invalid key format in RequestsByUser: %v", key)
				return true
			}

			user, bucket, method, httpStatus := parts[0], parts[1], parts[2], parts[3]
			userStr, tenantStr := extractUserAndTenant(user)
			rqCount := float64(requestCount.(*atomic.Uint64).Load())
			if rqCount <= 0 {
				return true
			}

			add := func(labels prometheus.Labels) {
				totalRequestsCounter.With(labels).Add(rqCount)
			}

			// Full request count per user, grouped by method & status
			add(prometheus.Labels{
				"pod":         cfg.PodName,
				"user":        userStr,
				"tenant":      tenantStr,
				"bucket":      bucket,
				"method":      method,
				"http_status": httpStatus,
			})

			// Aggregated per bucket (all methods, but status-specific)
			add(prometheus.Labels{
				"pod":         cfg.PodName,
				"user":        userStr,
				"tenant":      tenantStr,
				"bucket":      bucket,
				"method":      "all",
				"http_status": httpStatus,
			})

			// Fully aggregated per bucket (all methods, all statuses)
			add(prometheus.Labels{
				"pod":         cfg.PodName,
				"user":        userStr,
				"tenant":      tenantStr,
				"bucket":      bucket,
				"method":      "all",
				"http_status": "all",
			})

			// Fully aggregated per user (all buckets, methods, and statuses)
			add(prometheus.Labels{
				"pod":         cfg.PodName,
				"user":        userStr,
				"tenant":      tenantStr,
				"bucket":      "all",
				"method":      "all",
				"http_status": "all",
			})

			return true
		})
	}

	if metricsConfig.TrackRequestsByTenant {
		diffMetrics.RequestsByUser.Range(func(key, requestCount any) bool {
			parts := strings.Split(key.(string), "|")
			if len(parts) != 4 {
				log.Warn().Msgf("Invalid key format in RequestsByUser: %v", key)
				return true
			}
			user := parts[0]
			_, tenantStr := extractUserAndTenant(user)
			rqCount := float64(requestCount.(*atomic.Uint64).Load())
			if rqCount <= 0 {
				return true
			}

			// Track per-tenant request count
			totalRequestsCounter.With(prometheus.Labels{
				"pod":         cfg.PodName,
				"user":        "all",
				"tenant":      tenantStr,
				"bucket":      "all",
				"method":      "all",
				"http_status": "all",
			}).Add(rqCount)

			return true
		})
	}

	if metricsConfig.TrackRequestsByBucket {
		diffMetrics.RequestsByBucket.Range(func(key, requestCount any) bool {
			parts := strings.Split(key.(string), "|")
			if len(parts) != 3 {
				log.Warn().Msgf("Invalid key format in RequestsByBucket: %v", key)
				return true
			}

			bucket, method, httpStatus := parts[0], parts[1], parts[2]
			rqCount := float64(requestCount.(*atomic.Uint64).Load())
			if rqCount <= 0 {
				return true
			}

			add := func(labels prometheus.Labels) {
				totalRequestsCounter.With(labels).Add(rqCount)
			}

			// Full request count per bucket, grouped by method & status
			add(prometheus.Labels{
				"pod":         cfg.PodName,
				"user":        "all",
				"tenant":      "all",
				"bucket":      bucket,
				"method":      method,
				"http_status": httpStatus,
			})

			// Aggregate version per bucket (all methods, but status-specific)
			add(prometheus.Labels{
				"pod":         cfg.PodName,
				"user":        "all",
				"tenant":      "all",
				"bucket":      bucket,
				"method":      "all",
				"http_status": httpStatus,
			})

			// Fully aggregated request count (all methods, all statuses)
			add(prometheus.Labels{
				"pod":         cfg.PodName,
				"user":        "all",
				"tenant":      "all",
				"bucket":      bucket,
				"method":      "all",
				"http_status": "all",
			})

			return true
		})
	}

	if metricsConfig.TrackRequestsByMethod {
		// Requests per HTTP Method (GET, PUT, DELETE) grouped by User & Bucket
		diffMetrics.RequestsByMethod.Range(func(key, count any) bool {
			// Key format: "user|bucket|method"
			parts := strings.Split(key.(string), "|")
			user, bucket, method := parts[0], parts[1], parts[2]
			userStr, tenantStr := extractUserAndTenant(user)
			requestCount := float64(count.(*atomic.Uint64).Load())

			requestsByMethodCounter.With(prometheus.Labels{
				"pod":    cfg.PodName,
				"user":   userStr,
				"tenant": tenantStr,
				"bucket": bucket,
				"method": method,
			}).Add(requestCount)

			return true
		})
	}

	if metricsConfig.TrackRequestsByOperation {
		// Requests per Operation (Grouped by User & Bucket)
		diffMetrics.RequestsByOperation.Range(func(key, count any) bool {
			// Key format: "user|bucket|operation|method"
			parts := strings.Split(key.(string), "|")
			user, bucket, operation, method := parts[0], parts[1], parts[2], parts[3]
			userStr, tenantStr := extractUserAndTenant(user)
			requestCount := float64(count.(*atomic.Uint64).Load())

			requestsByOperationCounter.With(prometheus.Labels{
				"pod":       cfg.PodName,
				"user":      userStr,
				"tenant":    tenantStr,
				"bucket":    bucket,
				"operation": operation,
				"method":    method,
			}).Add(requestCount)

			return true
		})
	}

	if metricsConfig.TrackRequestsByStatus {
		// Requests per Status Code (Grouped by User & Bucket)
		diffMetrics.RequestsByStatusCode.Range(func(status, count any) bool {
			statusStr := status.(string)
			requestCount := float64(count.(*atomic.Uint64).Load())

			requestsByStatusCodeCounter.With(prometheus.Labels{
				"pod":    cfg.PodName,
				"user":   "all",
				"tenant": "all",
				"bucket": "all",
				"status": statusStr,
			}).Add(requestCount)

			return true
		})
	}

	if metricsConfig.TrackBytesSentByBucket {
		diffMetrics.BytesSentByBucket.Range(func(bucket, bytes any) bool {
			bucketStr := bucket.(string)
			totalBytes := float64(bytes.(*atomic.Uint64).Load())

			bytesSentCounter.With(prometheus.Labels{
				"pod":    cfg.PodName,
				"user":   "all",
				"tenant": "all",
				"bucket": bucketStr,
			}).Add(totalBytes)
			return true
		})
	}

	if metricsConfig.TrackBytesSentByUser {
		diffMetrics.BytesSentByUser.Range(func(user, bytes any) bool {
			userStr, tenantStr := extractUserAndTenant(user.(string))
			totalBytes := float64(bytes.(*atomic.Uint64).Load())

			bytesSentCounter.With(prometheus.Labels{
				"pod":    cfg.PodName,
				"user":   userStr,
				"tenant": tenantStr,
				"bucket": "all",
			}).Add(totalBytes)
			return true
		})
	}

	if metricsConfig.TrackBytesReceivedByUser {
		diffMetrics.BytesReceivedByUser.Range(func(user, bytes any) bool {
			userStr, tenantStr := extractUserAndTenant(user.(string))
			totalBytes := float64(bytes.(*atomic.Uint64).Load())

			bytesReceivedCounter.With(prometheus.Labels{
				"pod":    cfg.PodName,
				"user":   userStr,
				"tenant": tenantStr,
				"bucket": "all",
			}).Add(totalBytes)
			return true
		})
	}

	if metricsConfig.TrackBytesReceivedByBucket {
		diffMetrics.BytesReceivedByBucket.Range(func(bucket, bytes any) bool {
			bucketStr := bucket.(string)
			totalBytes := float64(bytes.(*atomic.Uint64).Load())

			bytesReceivedCounter.With(prometheus.Labels{
				"pod":    cfg.PodName,
				"user":   "all",
				"tenant": "all",
				"bucket": bucketStr,
			}).Add(totalBytes)
			return true
		})
	}

	if metricsConfig.TrackErrorsByUser {
		// Iterate over users and publish their specific error counts
		diffMetrics.ErrorsByUser.Range(func(user, count any) bool {
			userStr, tenantStr := extractUserAndTenant(user.(string))
			if atomicPtr, ok := count.(*atomic.Uint64); ok {
				errorCount := atomicPtr.Load()
				errorsCounter.With(prometheus.Labels{
					"pod":         cfg.PodName,
					"user":        userStr,
					"tenant":      tenantStr,
					"bucket":      "all",
					"http_status": "all",
				}).Add(float64(errorCount))
			} else {
				fmt.Printf("Warning: Failed to cast error count for user %s\n", userStr)
			}
			return true
		})
	}

	if metricsConfig.TrackRequestsByIP {
		// Publish requests per IP & User
		currentMetrics.RequestsByIP.Range(func(key, count any) bool {
			keyStr := key.(string)
			parts := strings.Split(keyStr, "|")
			user, ip := parts[0], parts[1]
			userStr, tenantStr := extractUserAndTenant(user)

			if atomicPtr, ok := count.(*atomic.Uint64); ok {
				requestCount := atomicPtr.Load()
				requestsByIPGauge.With(prometheus.Labels{
					"pod":    cfg.PodName,
					"user":   userStr,
					"tenant": tenantStr,
					"ip":     ip,
				}).Set(float64(requestCount))
			}
			return true
		})
	}

	if metricsConfig.TrackRequestsByIPBucketMethodTenant {
		currentMetrics.RequestsByIPBucketMethodTenant.Range(func(key, count any) bool {
			parts := strings.Split(key.(string), "|")
			if len(parts) != 4 {
				log.Warn().Msgf("Invalid key format in RequestsByIPBucketMethodTenant: %v", key)
				return true
			}

			ip, bucket, method, user := parts[0], parts[1], parts[2], parts[3]
			_, tenantStr := extractUserAndTenant(user)

			if atomicPtr, ok := count.(*atomic.Uint64); ok {
				requestsByIPBucketMethodTenantGauge.With(prometheus.Labels{
					"pod":    cfg.PodName,
					"ip":     ip,
					"bucket": bucket,
					"method": method,
					"tenant": tenantStr,
				}).Set(float64(atomicPtr.Load()))
			}
			return true
		})
	}

	if metricsConfig.TrackBytesSentByIP {
		// Publish bytes sent per IP & User
		currentMetrics.BytesSentByIP.Range(func(key, bytesSent any) bool {
			keyStr := key.(string)
			parts := strings.Split(keyStr, "|")
			user, ip := parts[0], parts[1]
			userStr, tenantStr := extractUserAndTenant(user)

			if atomicPtr, ok := bytesSent.(*atomic.Uint64); ok {
				totalBytesSent := atomicPtr.Load()
				bytesSentByIPGauge.With(prometheus.Labels{
					"pod":    cfg.PodName,
					"user":   userStr,
					"tenant": tenantStr,
					"ip":     ip,
				}).Set(float64(totalBytesSent))
			}
			return true
		})
	}

	if metricsConfig.TrackBytesReceivedByIP {
		// Publish bytes received per IP & User
		currentMetrics.BytesReceivedByIP.Range(func(key, bytesReceived any) bool {
			keyStr := key.(string)
			parts := strings.Split(keyStr, "|")
			user, ip := parts[0], parts[1]
			userStr, tenantStr := extractUserAndTenant(user)

			if atomicPtr, ok := bytesReceived.(*atomic.Uint64); ok {
				totalBytesReceived := atomicPtr.Load()
				bytesReceivedByIPGauge.With(prometheus.Labels{
					"pod":    cfg.PodName,
					"user":   userStr,
					"tenant": tenantStr,
					"ip":     ip,
				}).Set(float64(totalBytesReceived))
			}
			return true
		})
	}

	if metricsConfig.TrackErrorsByUser {
		// Publish HTTP errors per User & Bucket
		diffMetrics.ErrorsByUserAndBucket.Range(func(key, count any) bool {
			keyStr := key.(string)
			parts := strings.Split(keyStr, "|")
			user, bucket, status := parts[0], parts[1], parts[2]
			userStr, tenantStr := extractUserAndTenant(user)

			// Exclude HTTP status codes in the 2xx range
			if strings.HasPrefix(status, "2") {
				return true
			}

			if atomicPtr, ok := count.(*atomic.Uint64); ok {
				errorCount := atomicPtr.Load()
				httpErrorsByUserCounter.With(prometheus.Labels{
					"pod":         cfg.PodName,
					"user":        userStr,
					"tenant":      tenantStr,
					"bucket":      bucket,
					"http_status": status,
				}).Add(float64(errorCount))
			}
			return true
		})
	}

	if metricsConfig.TrackErrorsByBucket {
		diffMetrics.ErrorsByUserAndBucket.Range(func(key, count any) bool {
			parts := strings.Split(key.(string), "|")
			if len(parts) != 3 {
				log.Warn().Msgf("Invalid key format in ErrorsByUserAndBucket: %v", key)
				return true
			}
			_, bucket, status := parts[0], parts[1], parts[2]

			// Exclude HTTP status codes in the 2xx range
			if strings.HasPrefix(status, "2") {
				return true
			}

			errorCount := float64(count.(*atomic.Uint64).Load())

			errorsCounter.With(prometheus.Labels{
				"pod":         cfg.PodName,
				"user":        "all",
				"tenant":      "all",
				"bucket":      bucket,
				"http_status": status,
			}).Add(errorCount)

			return true
		})
	}

	if metricsConfig.TrackErrorsByStatus {
		diffMetrics.RequestsByStatusCode.Range(func(status, count any) bool {
			statusStr := status.(string)

			// Exclude HTTP status codes in the 2xx range
			if strings.HasPrefix(statusStr, "2") {
				return true
			}

			requestCount := float64(count.(*atomic.Uint64).Load())
			errorsCounter.With(prometheus.Labels{
				"pod":         cfg.PodName,
				"user":        "all",
				"tenant":      "all",
				"bucket":      "all",
				"http_status": status.(string),
			}).Add(requestCount)

			return true
		})
	}

	if metricsConfig.TrackErrorsByIP {
		// Publish HTTP errors per IP & Bucket
		diffMetrics.ErrorsByIPAndBucket.Range(func(key, count any) bool {
			keyStr := key.(string)
			parts := strings.Split(keyStr, "|")
			ip, bucket, status := parts[0], parts[1], parts[2]

			// Exclude HTTP status codes in the 2xx range
			if strings.HasPrefix(status, "2") {
				return true
			}

			if atomicPtr, ok := count.(*atomic.Uint64); ok {
				errorCount := atomicPtr.Load()
				httpErrorsByIPCounter.With(prometheus.Labels{
					"pod":         cfg.PodName,
					"ip":          ip,
					"bucket":      bucket,
					"http_status": status,
				}).Add(float64(errorCount))
			}
			return true
		})
	}

	// Update request duration histogram (latency metrics)
	if metricsConfig.TrackLatencyByMethod || metricsConfig.TrackLatencyByUser ||
		metricsConfig.TrackLatencyByBucket || metricsConfig.TrackLatencyByTenant ||
		metricsConfig.TrackLatencyByBucketAndMethod {

		diffMetrics.LatencyByMethod.Range(func(key, totalLatency any) bool {
			parts := strings.Split(key.(string), "|")
			if len(parts) != 3 {
				log.Warn().Msgf("Invalid key format in LatencyByMethod: %v", key)
				return true
			}
			user, bucket, method := parts[0], parts[1], parts[2]
			userStr, tenantStr := extractUserAndTenant(user)

			// Fetch request count for this method
			countVal, exists := diffMetrics.RequestsByMethod.Load(key)
			if !exists {
				countVal, exists = currentMetrics.RequestsByMethod.Load(key)
				if !exists {
					log.Warn().Msgf("Missing request count for latency key: %v", key)
					return true
				}
			}
			count := float64(countVal.(*atomic.Uint64).Load())
			if count == 0 {
				log.Warn().Msgf("Zero request count for latency key: %v", key)
				return true
			}

			// Compute avg latency (convert ms → sec)
			avgLatencySec := float64(totalLatency.(*atomic.Uint64).Load()) / count / 1000.0

			if metricsConfig.TrackLatencyByBucketAndMethod {
				// Fine-grained latency tracking per bucket & method
				requestsDurationHistogram.With(prometheus.Labels{
					// "pod":    cfg.PodName,
					"user":   userStr,
					"tenant": tenantStr,
					"bucket": bucket,
					"method": method,
				}).Observe(avgLatencySec)
			}

			if metricsConfig.TrackLatencyByMethod {
				// Aggregated latency for all users (reduces cardinality)
				requestsDurationHistogram.With(prometheus.Labels{
					// "pod":    cfg.PodName,
					"user":   "all",
					"tenant": "all",
					"bucket": bucket,
					"method": method,
				}).Observe(avgLatencySec)
			}

			if metricsConfig.TrackLatencyByBucket {
				requestsDurationHistogram.With(prometheus.Labels{
					"user":   "all",
					"tenant": "all",
					"bucket": bucket,
					"method": "all",
				}).Observe(avgLatencySec)
			}

			if metricsConfig.TrackLatencyByTenant {
				requestsDurationHistogram.With(prometheus.Labels{
					"user":   "all",
					"tenant": tenantStr,
					"bucket": "all",
					"method": "all",
				}).Observe(avgLatencySec)
			}

			if metricsConfig.TrackLatencyByUser {
				requestsDurationHistogram.With(prometheus.Labels{
					"user":   userStr,
					"tenant": tenantStr,
					"bucket": "all",
					"method": "all",
				}).Observe(avgLatencySec)
			}

			// // Aggregate latency for all methods
			// requestsDurationHistogram.With(prometheus.Labels{
			// 	"pod":    cfg.PodName,
			// 	"user":   userStr,
			// 	"tenant": tenantStr,
			// 	"bucket": bucket,
			// 	"method": "all",
			// }).Observe(avgLatencySec)

			return true
		})
	}

	log.Info().Msg("Updated Prometheus metrics for users and buckets")
}

func StartPrometheusServer(port int, metricsConfig *MetricsConfig) {
	// Initialize Prometheus settings based on the configuration
	initPrometheusSettings(metricsConfig)

	// Start the Prometheus HTTP server
	go func() {
		http.Handle("/metrics", promhttp.Handler())
		log.Info().Msgf("starting prometheus metrics server on :%d", port)
		err := http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
		if err != nil {
			log.Fatal().Err(err).Msg("error starting prometheus metrics server")
		}
	}()
}
