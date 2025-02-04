// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and prysm contributors
//
// SPDX-License-Identifier: Apache-2.0

package radosgwusage

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/nats-io/nats.go"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog/log"
)

var (
	prysmTartgetUp = newGaugeVec("prysm_target_up", "Indicates if the exporter can reach the target (1 = up, 0 = down).", []string{})
	scrapeErrors   = newCounterVec("exporter_scrape_errors_total", "Total number of errors during scraping.", []string{})

	// User-level metrics
	userMetadata = newGaugeVec("radosgw_user_metadata", "User metadata", []string{"user", "display_name", "email", "storage_class", "rgw_cluster_id", "node", "instance_id"})

	userLabels               = []string{"user", "rgw_cluster_id", "node", "instance_id"}
	userBucketsTotal         = newGaugeVec("radosgw_user_buckets_total", "Total number of buckets for each user", userLabels)
	userObjectsTotal         = newGaugeVec("radosgw_user_objects_total", "Total number of objects for each user", userLabels)
	userDataSizeTotal        = newGaugeVec("radosgw_user_data_size_bytes", "Total size of data for each user in bytes", userLabels)
	userOpsTotal             = newGaugeVec("radosgw_user_ops_total", "Total operations performed by each user", userLabels)
	userReadOpsTotal         = newGaugeVec("radosgw_user_read_ops_total", "Total read operations per user", userLabels)
	userWriteOpsTotal        = newGaugeVec("radosgw_user_write_ops_total", "Total write operations per user", userLabels)
	userBytesSentTotal       = newGaugeVec("radosgw_user_bytes_sent_total", "Total bytes sent by each user (cumulative)", userLabels)
	userBytesReceivedTotal   = newGaugeVec("radosgw_user_bytes_received_total", "Total bytes received by each user (cumulative)", userLabels)
	userSuccessOpsTotal      = newGaugeVec("radosgw_user_success_ops_total", "Total successful operations per user", userLabels)
	userThroughputBytesTotal = newGaugeVec("radosgw_user_throughput_bytes_total", "Total throughput for each user in bytes (read and write combined)", userLabels)
	userErrorRateTotal       = newGaugeVec("radosgw_user_error_rate_total", "Total number of errors per user", userLabels)
	apiUsagePerUser          = newGaugeVec("radosgw_api_usage_per_user", "API usage per user and category", []string{"user", "rgw_cluster_id", "node", "instance_id", "api_category"})
	// User quota metrics
	userQuotaEnabled    = newGaugeVec("radosgw_usage_user_quota_enabled", "User quota enabled", userLabels)
	userQuotaMaxSize    = newGaugeVec("radosgw_usage_user_quota_size", "Maximum allowed size for user", userLabels)
	userQuotaMaxObjects = newGaugeVec("radosgw_usage_user_quota_size_objects", "Maximum allowed number of objects across all user buckets", userLabels)

	// Bucket-level metrics
	bucketLabels               = []string{"bucket", "owner", "zonegroup", "rgw_cluster_id", "node", "instance_id"}
	bucketReadOpsTotal         = newGaugeVec("radosgw_bucket_read_ops_total", "Total read operations in each bucket", bucketLabels)
	bucketWriteOpsTotal        = newGaugeVec("radosgw_bucket_write_ops_total", "Total write operations in each bucket", bucketLabels)
	bucketOpsTotal             = newGaugeVec("radosgw_bucket_ops_total", "Total operations performed in each bucket", bucketLabels)
	bucketBytesSentTotal       = newGaugeVec("radosgw_bucket_bytes_sent_total", "Total bytes sent from each bucket", bucketLabels)
	bucketBytesReceivedTotal   = newGaugeVec("radosgw_bucket_bytes_received_total", "Total bytes received by each bucket", bucketLabels)
	bucketThroughputBytesTotal = newGaugeVec("radosgw_bucket_throughput_bytes_total", "Total throughput for each bucket in bytes (read and write combined)", bucketLabels)
	bucketSize                 = newGaugeVec("radosgw_usage_bucket_size", "Size of bucket", bucketLabels)
	bucketObjectCount          = newGaugeVec("radosgw_usage_bucket_objects", "Number of objects in bucket", bucketLabels)
	bucketShards               = newGaugeVec("radosgw_usage_bucket_shards", "Number of shards in bucket", bucketLabels)
	bucketAPIUsageTotal        = newGaugeVec("radosgw_bucket_api_usage_total", "Total API usage per category for each bucket", []string{"bucket", "owner", "zonegroup", "rgw_cluster_id", "node", "instance_id", "category"})
	// Quota metrics
	bucketQuotaEnabled    = newGaugeVec("radosgw_usage_bucket_quota_enabled", "Quota enabled for bucket", bucketLabels)
	bucketQuotaMaxSize    = newGaugeVec("radosgw_usage_bucket_quota_size", "Maximum allowed bucket size", bucketLabels)
	bucketQuotaMaxObjects = newGaugeVec("radosgw_usage_bucket_quota_size_objects", "Maximum allowed bucket size in number of objects", bucketLabels)

	// Cluster-level metrics
	// clusterReadOpsTotal         = newCounterVec("radosgw_cluster_read_ops_total", "Total read operations performed in the cluster", []string{"rgw_cluster_id", "node", "instance_id"})
	// clusterWriteOpsTotal        = newCounterVec("radosgw_cluster_write_ops_total", "Total write operations performed in the cluster", []string{"rgw_cluster_id", "node", "instance_id"})
	// clusterOpsTotal             = newCounterVec("radosgw_cluster_ops_total", "Total operations performed in the cluster", []string{"rgw_cluster_id", "node", "instance_id"})
	// clusterBytesSentTotal       = newCounterVec("radosgw_cluster_bytes_sent_total", "Total bytes sent in the cluster", []string{"rgw_cluster_id", "node", "instance_id"})
	// clusterBytesReceivedTotal   = newCounterVec("radosgw_cluster_bytes_received_total", "Total bytes received in the cluster", []string{"rgw_cluster_id", "node", "instance_id"})
	// clusterThroughputBytesTotal = newCounterVec("radosgw_cluster_throughput_bytes_total", "Total throughput of the cluster in bytes (read and write combined)", []string{"rgw_cluster_id", "node", "instance_id"})

	clusterReadOpsTotal         = newGaugeVec("radosgw_cluster_read_ops_total", "Total read operations performed in the cluster", []string{"rgw_cluster_id", "node", "instance_id"})
	clusterWriteOpsTotal        = newGaugeVec("radosgw_cluster_write_ops_total", "Total write operations performed in the cluster", []string{"rgw_cluster_id", "node", "instance_id"})
	clusterOpsTotal             = newGaugeVec("radosgw_cluster_ops_total", "Total operations performed in the cluster", []string{"rgw_cluster_id", "node", "instance_id"})
	clusterBytesSentTotal       = newGaugeVec("radosgw_cluster_bytes_sent_total", "Total bytes sent in the cluster", []string{"rgw_cluster_id", "node", "instance_id"})
	clusterBytesReceivedTotal   = newGaugeVec("radosgw_cluster_bytes_received_total", "Total bytes received in the cluster", []string{"rgw_cluster_id", "node", "instance_id"})
	clusterThroughputBytesTotal = newGaugeVec("radosgw_cluster_throughput_bytes_total", "Total throughput of the cluster in bytes (read and write combined)", []string{"rgw_cluster_id", "node", "instance_id"})

	// clusterErrorRate          = newGaugeVec("radosgw_cluster_error_rate", "Error rate (percentage) for the entire cluster", []string{"rgw_cluster_id", "node", "instance_id"})
	// clusterCapacityUsageBytes = newGaugeVec("radosgw_cluster_capacity_usage_bytes", "Total capacity used across the entire cluster in bytes", []string{"rgw_cluster_id", "node", "instance_id"})
	// clusterSuccessOpsTotal    = newGaugeVec("radosgw_cluster_success_ops_total", "Total successful operations across the entire cluster", []string{"rgw_cluster_id", "node", "instance_id"})

	// bucketSuccessOpsTotal = newGaugeVec("radosgw_bucket_success_ops_total", "Total successful operations for each bucket", bucketLabels)
	// bucketErrorRate       = newGaugeVec("radosgw_bucket_error_rate", "Error rate for each bucket (percentage)", bucketLabels)
	// bucketCapacityUsage   = newGaugeVec("radosgw_bucket_capacity_usage_bytes", "Total capacity used by each bucket in bytes", bucketLabels)
	// bucketUsageBytes           = newGaugeVec("radosgw_usage_bucket_bytes", "Bucket used bytes", bucketLabels)
	// bucketUtilizedBytes        = newGaugeVec("radosgw_usage_bucket_utilized_bytes", "Bucket utilized bytes", bucketLabels)

)

func newCounterVec(name, help string, labels []string) *prometheus.CounterVec {
	return prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: name,
		Help: help,
	}, labels)
}

func newGaugeVec(name, help string, labels []string) *prometheus.GaugeVec {
	return prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: name,
		Help: help,
	}, labels)
}

func newHistogramVec(name, help string, labels []string) *prometheus.HistogramVec {
	return prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    name,
		Help:    help,
		Buckets: prometheus.DefBuckets,
	}, labels)
}

func init() {
	// Register all metrics with Prometheus's default registry
	prometheus.MustRegister(prysmTartgetUp, scrapeErrors)

	prometheus.MustRegister(userMetadata)
	prometheus.MustRegister(userBucketsTotal)
	prometheus.MustRegister(userObjectsTotal)
	prometheus.MustRegister(userDataSizeTotal)
	prometheus.MustRegister(userOpsTotal)
	prometheus.MustRegister(userReadOpsTotal)
	prometheus.MustRegister(userWriteOpsTotal)
	prometheus.MustRegister(userBytesSentTotal)
	prometheus.MustRegister(userBytesReceivedTotal)
	prometheus.MustRegister(userSuccessOpsTotal)
	prometheus.MustRegister(userThroughputBytesTotal)
	prometheus.MustRegister(userErrorRateTotal)
	prometheus.MustRegister(apiUsagePerUser)
	prometheus.MustRegister(userQuotaEnabled)
	prometheus.MustRegister(userQuotaMaxSize)
	prometheus.MustRegister(userQuotaMaxObjects)

	prometheus.MustRegister(bucketReadOpsTotal)
	prometheus.MustRegister(bucketWriteOpsTotal)
	prometheus.MustRegister(bucketOpsTotal)
	prometheus.MustRegister(bucketBytesSentTotal)
	prometheus.MustRegister(bucketBytesReceivedTotal)
	prometheus.MustRegister(bucketThroughputBytesTotal)
	prometheus.MustRegister(bucketSize)
	prometheus.MustRegister(bucketObjectCount)
	prometheus.MustRegister(bucketShards)
	prometheus.MustRegister(bucketAPIUsageTotal)
	prometheus.MustRegister(bucketQuotaEnabled)
	prometheus.MustRegister(bucketQuotaMaxSize)
	prometheus.MustRegister(bucketQuotaMaxObjects)

	prometheus.MustRegister(clusterReadOpsTotal)
	prometheus.MustRegister(clusterWriteOpsTotal)
	prometheus.MustRegister(clusterOpsTotal)
	prometheus.MustRegister(clusterBytesSentTotal)
	prometheus.MustRegister(clusterBytesReceivedTotal)
	prometheus.MustRegister(clusterThroughputBytesTotal)
}

func startPrometheusMetricsServer(port int) {
	http.Handle("/metrics", promhttp.Handler())
	http.ListenAndServe(":"+strconv.Itoa(port), nil)
}

func populateStatus(status *PrysmStatus) {
	// Safely get the current status snapshot
	up, errors := status.GetSnapshot()

	// Update Prometheus metrics
	prysmTartgetUp.With(prometheus.Labels{}).Set(up)
	scrapeErrors.With(prometheus.Labels{}).Add(float64(errors))
}

func populateMetricsFromKV(userMetrics, bucketMetrics, clusterMetrics nats.KeyValue, cfg RadosGWUsageConfig) {
	log.Info().Msg("Starting to populate metrics from KV")

	// Process user metrics
	populateUserMetricsFromKV(userMetrics, cfg)

	// Process bucket metrics
	populateBucketMetricsFromKV(bucketMetrics, cfg)

	// Process cluster metrics
	populateClusterMetricsFromKV(clusterMetrics, cfg)

	log.Info().Msg("Completed populating metrics from KV")
}

func populateUserMetricsFromKV(userMetrics nats.KeyValue, cfg RadosGWUsageConfig) {
	keys, err := userMetrics.Keys()
	if err != nil {
		log.Error().Err(err).Msg("Failed to fetch keys from user metrics KV")
		return
	}

	for _, key := range keys {
		entry, err := userMetrics.Get(key)
		if err != nil {
			log.Warn().Str("key", key).Err(err).Msg("Failed to fetch user metric")
			continue
		}

		var metrics UserLevelMetrics
		if err := json.Unmarshal(entry.Value(), &metrics); err != nil {
			log.Warn().Str("key", key).Err(err).Msg("Failed to unmarshal user metric")
			continue
		}

		userMetadata.With(prometheus.Labels{
			"user":           metrics.UserID,
			"display_name":   metrics.DisplayName,
			"email":          metrics.Email,
			"storage_class":  metrics.DefaultStorageClass,
			"rgw_cluster_id": cfg.ClusterID,
			"node":           cfg.NodeName,
			"instance_id":    cfg.InstanceID,
		}).Set(1)

		labels := prometheus.Labels{
			"user":           metrics.UserID,
			"rgw_cluster_id": cfg.ClusterID,
			"node":           cfg.NodeName,
			"instance_id":    cfg.InstanceID,
		}

		userBucketsTotal.With(labels).Set(float64(metrics.BucketsTotal))
		userObjectsTotal.With(labels).Set(float64(metrics.ObjectsTotal))
		userDataSizeTotal.With(labels).Set(float64(metrics.DataSizeTotal))
		userOpsTotal.With(labels).Set(float64(metrics.TotalOPs))
		userReadOpsTotal.With(labels).Set(float64(metrics.TotalReadOPs))
		userWriteOpsTotal.With(labels).Set(float64(metrics.TotalWriteOPs))
		userBytesSentTotal.With(labels).Set(float64(metrics.BytesSentTotal))
		userBytesReceivedTotal.With(labels).Set(float64(metrics.BytesReceivedTotal))
		userSuccessOpsTotal.With(labels).Set(float64(metrics.UserSuccessOpsTotal))
		userThroughputBytesTotal.With(labels).Set(float64(metrics.ThroughputBytesTotal))
		if metrics.TotalOPs > 0 {
			userErrorRateTotal.With(labels).Set(metrics.ErrorRatePerUser)
		}

		for category, ops := range metrics.APIUsage {
			labels := prometheus.Labels{
				"user":           metrics.UserID,
				"rgw_cluster_id": cfg.ClusterID,
				"node":           cfg.NodeName,
				"instance_id":    cfg.InstanceID,
				"api_category":   category,
			}
			apiUsagePerUser.With(labels).Set(float64(ops))
		}

		// User quota metrics
		userQuotaEnabled.With(labels).Set(boolToFloat64(&metrics.UserQuotaEnabled))
		if metrics.UserQuotaMaxSize != nil && *metrics.UserQuotaMaxSize > 0 {
			userQuotaMaxSize.With(labels).Set(float64(*metrics.UserQuotaMaxSize))
		}
		if metrics.UserQuotaMaxObjects != nil && *metrics.UserQuotaMaxObjects > 0 {
			userQuotaMaxObjects.With(labels).Set(float64(*metrics.UserQuotaMaxObjects))
		}
	}
}

func populateBucketMetricsFromKV(bucketMetrics nats.KeyValue, cfg RadosGWUsageConfig) {
	keys, err := bucketMetrics.Keys()
	if err != nil {
		log.Error().Err(err).Msg("Failed to fetch keys from bucket metrics KV")
		return
	}

	for _, key := range keys {
		entry, err := bucketMetrics.Get(key)
		if err != nil {
			log.Warn().Str("key", key).Err(err).Msg("Failed to fetch bucket metric")
			continue
		}

		var metrics UserBucketMetrics
		if err := json.Unmarshal(entry.Value(), &metrics); err != nil {
			log.Warn().Str("key", key).Err(err).Msg("Failed to unmarshal bucket metric")
			continue
		}

		labels := prometheus.Labels{
			"bucket":         metrics.BucketID,
			"owner":          metrics.Owner,
			"zonegroup":      metrics.Zonegroup,
			"rgw_cluster_id": cfg.ClusterID,
			"node":           cfg.NodeName,
			"instance_id":    cfg.InstanceID,
		}

		bucketReadOpsTotal.With(labels).Set(float64(metrics.TotalReadOPs))
		bucketWriteOpsTotal.With(labels).Set(float64(metrics.TotalWriteOPs))
		bucketOpsTotal.With(labels).Set(float64(metrics.TotalOPs))
		bucketBytesSentTotal.With(labels).Set(float64(metrics.BytesSentTotal))
		bucketBytesReceivedTotal.With(labels).Set(float64(metrics.BytesReceivedTotal))
		bucketThroughputBytesTotal.With(labels).Set(float64(metrics.Throughput))
		bucketSize.With(labels).Set(float64(metrics.BucketSize))
		bucketObjectCount.With(labels).Set(float64(metrics.ObjectCount))

		if metrics.NumShards != nil {
			bucketShards.With(labels).Set(float64(*metrics.NumShards))
		}

		for category, ops := range metrics.APIUsage {
			labels := prometheus.Labels{
				"bucket":         metrics.BucketID,
				"owner":          metrics.Owner,
				"zonegroup":      metrics.Zonegroup,
				"rgw_cluster_id": cfg.ClusterID,
				"node":           cfg.NodeName,
				"instance_id":    cfg.InstanceID,
				"category":       category,
			}
			bucketAPIUsageTotal.With(labels).Set(float64(ops))
		}

		// Set quota information
		bucketQuotaEnabled.With(labels).Set(boolToFloat64(&metrics.QuotaEnabled))
		if metrics.QuotaMaxSize != nil && *metrics.QuotaMaxSize > 0 {
			bucketQuotaMaxSize.With(labels).Set(float64(*metrics.QuotaMaxSize))
		}
		if metrics.QuotaMaxObjects != nil && *metrics.QuotaMaxObjects > 0 {
			bucketQuotaMaxObjects.With(labels).Set(float64(*metrics.QuotaMaxObjects))
		}
	}
}

func populateClusterMetricsFromKV(clusterMetrics nats.KeyValue, cfg RadosGWUsageConfig) {
	keys, err := clusterMetrics.Keys()
	if err != nil {
		log.Error().Err(err).Msg("Failed to fetch keys from cluster metrics KV")
		return
	}

	for _, key := range keys {
		entry, err := clusterMetrics.Get(key)
		if err != nil {
			log.Warn().Str("key", key).Err(err).Msg("Failed to fetch cluster metric")
			continue
		}

		var metrics ClusterLevelMetrics
		if err := json.Unmarshal(entry.Value(), &metrics); err != nil {
			log.Warn().Str("key", key).Err(err).Msg("Failed to unmarshal cluster metric")
			continue
		}

		labels := prometheus.Labels{
			"rgw_cluster_id": cfg.ClusterID,
			"node":           cfg.NodeName,
			"instance_id":    cfg.InstanceID,
		}

		clusterReadOpsTotal.With(labels).Set(float64(metrics.TotalReadOPs))
		clusterWriteOpsTotal.With(labels).Set(float64(metrics.TotalWriteOPs))
		clusterOpsTotal.With(labels).Set(float64(metrics.TotalOPs))
		clusterBytesSentTotal.With(labels).Set(float64(metrics.BytesSentTotal))
		clusterBytesReceivedTotal.With(labels).Set(float64(metrics.BytesReceivedTotal))
		clusterThroughputBytesTotal.With(labels).Set(float64(metrics.Throughput))

		// Error rate is averaged across all users/buckets
		// if entry.UserLevel.Totals.OpsTotal > 0 {
		// 	errorRate := float64(entry.UserLevel.Totals.OpsTotal-entry.UserLevel.Totals.SuccessOpsTotal) / float64(entry.UserLevel.Totals.OpsTotal) * 100
		// 	clusterMetrics.ErrorRate += errorRate
		// }
	}
}

// // Store the last known values for cluster metrics
// var lastClusterMetrics = make(map[string]map[string]float64)

// func updateCounterForCluster(counterVec *prometheus.CounterVec, labels prometheus.Labels, metricName string, currentValue float64) {
// 	clusterID := labels["rgw_cluster_id"]

// 	// Initialize map for cluster if not exists
// 	if _, ok := lastClusterMetrics[clusterID]; !ok {
// 		lastClusterMetrics[clusterID] = make(map[string]float64)
// 	}

// 	// Calculate delta
// 	delta := currentValue
// 	if lastValue, ok := lastClusterMetrics[clusterID][metricName]; ok {
// 		delta = currentValue - lastValue
// 		if delta < 0 {
// 			log.Warn().
// 				Str("rgw_cluster_id", clusterID).
// 				Str("metric", metricName).
// 				Msg("Counter value decreased; resetting to current value")
// 			delta = currentValue
// 		}
// 	}

// 	// Increment Prometheus counter
// 	counterVec.With(labels).Add(delta)

// 	// Store current value as the last value
// 	lastClusterMetrics[clusterID][metricName] = currentValue
// }

// func populateClusterMetricsFromKV(clusterMetrics nats.KeyValue, cfg RadosGWUsageConfig) {
// 	keys, err := clusterMetrics.Keys()
// 	if err != nil {
// 		log.Error().Err(err).Msg("Failed to fetch keys from cluster metrics KV")
// 		return
// 	}

// 	for _, key := range keys {
// 		entry, err := clusterMetrics.Get(key)
// 		if err != nil {
// 			log.Warn().Str("key", key).Err(err).Msg("Failed to fetch cluster metric")
// 			continue
// 		}

// 		var metrics ClusterLevelMetrics
// 		if err := json.Unmarshal(entry.Value(), &metrics); err != nil {
// 			log.Warn().Str("key", key).Err(err).Msg("Failed to unmarshal cluster metric")
// 			continue
// 		}

// 		labels := prometheus.Labels{
// 			"rgw_cluster_id": cfg.ClusterID,
// 			"node":           cfg.NodeName,
// 			"instance_id":    cfg.InstanceID,
// 		}

// 		// Update counters with deltas
// 		updateCounterForCluster(clusterReadOpsTotal, labels, "TotalReadOPs", float64(metrics.TotalReadOPs))
// 		updateCounterForCluster(clusterWriteOpsTotal, labels, "TotalWriteOPs", float64(metrics.TotalWriteOPs))
// 		updateCounterForCluster(clusterOpsTotal, labels, "TotalOPs", float64(metrics.TotalOPs))
// 		updateCounterForCluster(clusterBytesSentTotal, labels, "BytesSentTotal", float64(metrics.BytesSentTotal))
// 		updateCounterForCluster(clusterBytesReceivedTotal, labels, "BytesReceivedTotal", float64(metrics.BytesReceivedTotal))
// 		updateCounterForCluster(clusterThroughputBytesTotal, labels, "Throughput", float64(metrics.Throughput))
// 	}
// 	log.Info().Msg("Completed populating cluster metrics from KV")
// }

func boolToFloat64(b *bool) float64 {
	if b != nil && *b {
		return 1.0
	}
	return 0.0
}
