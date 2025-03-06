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
	// Total requests grouped by user and bucket
	totalRequestsCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "radosgw_total_requests",
			Help: "Total number of requests processed",
		},
		[]string{"pod", "user", "tenant", "bucket"},
	)

	// Requests grouped by HTTP method (GET, PUT, POST, DELETE, etc.)
	requestsByMethodCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "radosgw_requests_by_method",
			Help: "Number of requests grouped by HTTP method (GET, PUT, DELETE, etc.)",
		},
		[]string{"pod", "user", "tenant", "bucket", "method"},
	)

	// Requests grouped by operation (GET, PUT, DELETE, etc.)
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
		[]string{"pod", "user", "tenant", "bucket"},
	)

	// Minimum request latency (seconds) per user and bucket
	latencyMinGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "radosgw_latency_min_seconds",
			Help: "Minimum request latency in seconds",
		},
		[]string{"pod", "user", "tenant", "bucket"},
	)

	// Maximum request latency (seconds) per user and bucket
	latencyMaxGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "radosgw_latency_max_seconds",
			Help: "Maximum request latency in seconds",
		},
		[]string{"pod", "user", "tenant", "bucket"},
	)

	// Average request latency (seconds) per user and bucket
	latencyAvgGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "radosgw_latency_avg_seconds",
			Help: "Average request latency in seconds",
		},
		[]string{"pod", "user", "tenant", "bucket"},
	)

	requestsByIPGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "radosgw_requests_by_ip",
			Help: "Total number of requests per IP and user",
		},
		[]string{"pod", "user", "tenant", "ip"},
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
)

func init() {
	// Register the Prometheus metrics
	prometheus.MustRegister(totalRequestsCounter)
	prometheus.MustRegister(requestsByMethodCounter)
	prometheus.MustRegister(requestsByOperationCounter)
	prometheus.MustRegister(requestsByStatusCodeCounter)
	prometheus.MustRegister(bytesSentCounter)
	prometheus.MustRegister(bytesReceivedCounter)
	prometheus.MustRegister(errorsCounter)
	prometheus.MustRegister(latencyMinGauge)
	prometheus.MustRegister(latencyMaxGauge)
	prometheus.MustRegister(latencyAvgGauge)
	prometheus.MustRegister(requestsByIPGauge)
	prometheus.MustRegister(bytesSentByIPGauge)
	prometheus.MustRegister(bytesReceivedByIPGauge)
	prometheus.MustRegister(httpErrorsByUserCounter)
	prometheus.MustRegister(httpErrorsByIPCounter)
}

// PublishToPrometheus updates Prometheus metrics from aggregated data
func PublishToPrometheus(metrics *Metrics, cfg OpsLogConfig) {
	// Total requests grouped by user and bucket
	metrics.RequestsByUser.Range(func(user, requestCount interface{}) bool {
		userStr, tenantStr := extractUserAndTenant(user.(string))
		rqCount := float64(requestCount.(*atomic.Uint64).Load())

		totalRequestsCounter.With(prometheus.Labels{
			"pod":    cfg.PodName,
			"user":   userStr,
			"tenant": tenantStr,
			"bucket": "all",
		}).Add(rqCount)
		return true
	})

	metrics.RequestsByBucket.Range(func(bucket, requestCount interface{}) bool {
		bucketStr := bucket.(string)
		rqCount := float64(requestCount.(*atomic.Uint64).Load())

		totalRequestsCounter.With(prometheus.Labels{
			"pod":    cfg.PodName,
			"user":   "all",
			"tenant": "all",
			"bucket": bucketStr,
		}).Add(rqCount)
		return true
	})

	// Requests per HTTP Method (GET, PUT, DELETE) grouped by User & Bucket
	metrics.RequestsByMethod.Range(func(key, count interface{}) bool {
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

	// Requests per Operation (Grouped by User & Bucket)
	metrics.RequestsByOperation.Range(func(key, count interface{}) bool {
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

	// Requests per Status Code (Grouped by User & Bucket)
	metrics.RequestsByStatusCode.Range(func(status, count interface{}) bool {
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

	// Bytes Sent & Received per User & Bucket
	metrics.BytesSentByUser.Range(func(user, bytes interface{}) bool {
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

	metrics.BytesSentByBucket.Range(func(bucket, bytes interface{}) bool {
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

	metrics.BytesReceivedByUser.Range(func(user, bytes interface{}) bool {
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

	metrics.BytesReceivedByBucket.Range(func(bucket, bytes interface{}) bool {
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

	// Iterate over users and publish their specific error counts
	metrics.ErrorsByUser.Range(func(user, count interface{}) bool {
		userStr, tenantStr := extractUserAndTenant(user.(string))
		if atomicPtr, ok := count.(*atomic.Uint64); ok {
			errorCount := atomicPtr.Load()
			errorsCounter.With(prometheus.Labels{
				"pod":    cfg.PodName,
				"user":   userStr,
				"tenant": tenantStr,
				"bucket": "all",
			}).Add(float64(errorCount))
		} else {
			fmt.Printf("Warning: Failed to cast error count for user %s\n", userStr)
		}
		return true
	})

	// Publish min latency
	metrics.LatencyMinByUser.Range(func(user, latency interface{}) bool {
		userStr, tenantStr := extractUserAndTenant(user.(string))
		if atomicPtr, ok := latency.(*atomic.Uint64); ok {
			latencyMin := float64(atomicPtr.Load()) / 1000.0
			latencyMinGauge.With(prometheus.Labels{
				"pod":    cfg.PodName,
				"user":   userStr,
				"tenant": tenantStr,
				"bucket": "all",
			}).Set(latencyMin)
		}
		return true
	})

	// Publish max latency
	metrics.LatencyMaxByUser.Range(func(user, latency interface{}) bool {
		userStr, tenantStr := extractUserAndTenant(user.(string))
		if atomicPtr, ok := latency.(*atomic.Uint64); ok {
			latencyMax := float64(atomicPtr.Load()) / 1000.0
			latencyMaxGauge.With(prometheus.Labels{
				"pod":    cfg.PodName,
				"user":   userStr,
				"tenant": tenantStr,
				"bucket": "all",
			}).Set(latencyMax)
		}
		return true
	})

	// Publish average latency
	latencyCount := metrics.LatencyCount.Load()
	if latencyCount > 0 {
		avgLatency := float64(metrics.LatencySum.Load()) / float64(latencyCount) / 1000.0 // Convert ms to sec
		latencyAvgGauge.With(prometheus.Labels{
			"pod":    cfg.PodName,
			"user":   "all",
			"tenant": "all",
			"bucket": "all",
		}).Set(avgLatency)
	}

	// Publish requests per IP & User
	metrics.RequestsByIP.Range(func(key, count interface{}) bool {
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

	// Publish bytes sent per IP & User
	metrics.BytesSentByIP.Range(func(key, bytesSent interface{}) bool {
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

	// Publish bytes received per IP & User
	metrics.BytesReceivedByIP.Range(func(key, bytesReceived interface{}) bool {
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

	// Publish HTTP errors per User & Bucket
	metrics.ErrorsByUserAndBucket.Range(func(key, count interface{}) bool {
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

	// Publish HTTP errors per IP & Bucket
	metrics.ErrorsByIPAndBucket.Range(func(key, count interface{}) bool {
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

	log.Info().Msg("Updated Prometheus metrics for users and buckets")
}

func StartPrometheusServer(port int) {
	go func() {
		http.Handle("/metrics", promhttp.Handler())
		log.Info().Msgf("starting prometheus metrics server on :%d", port)
		err := http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
		if err != nil {
			log.Fatal().Err(err).Msg("error starting prometheus metrics server")
		}
	}()
}
