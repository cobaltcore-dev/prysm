// Copyright 2024 Clyso GmbH
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package radosgwusage

import (
	"net/http"
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	// General operation metrics
	opsTotal           = newCounterVec("radosgw_usage_ops_total", "Number of operations", []string{"bucket", "owner", "category", "rgw_cluster_id", "node", "instance_id"})
	successfulOpsTotal = newCounterVec("radosgw_usage_successful_ops_total", "Number of successful operations", []string{"bucket", "owner", "category", "rgw_cluster_id", "node", "instance_id"})

	// Byte metrics
	bytesSentTotal     = newCounterVec("radosgw_usage_sent_bytes_total", "Bytes sent by the RADOSGW", []string{"bucket", "owner", "category", "rgw_cluster_id", "node", "instance_id"})
	bytesReceivedTotal = newCounterVec("radosgw_usage_received_bytes_total", "Bytes received by the RADOSGW", []string{"bucket", "owner", "category", "rgw_cluster_id", "node", "instance_id"})

	// Bucket usage metrics
	bucketUsageBytes    = newGaugeVec("radosgw_usage_bucket_bytes", "Bucket used bytes", []string{"bucket", "owner", "zonegroup", "rgw_cluster_id", "node", "instance_id"})
	bucketUtilizedBytes = newGaugeVec("radosgw_usage_bucket_utilized_bytes", "Bucket utilized bytes", []string{"bucket", "owner", "zonegroup", "rgw_cluster_id", "node", "instance_id"})
	bucketUsageObjects  = newGaugeVec("radosgw_usage_bucket_objects", "Number of objects in bucket", []string{"bucket", "owner", "zonegroup", "rgw_cluster_id", "node", "instance_id"})

	// Quota metrics
	bucketQuotaEnabled      = newGaugeVec("radosgw_usage_bucket_quota_enabled", "Quota enabled for bucket", []string{"bucket", "owner", "zonegroup", "rgw_cluster_id", "node", "instance_id"})
	bucketQuotaMaxSize      = newGaugeVec("radosgw_usage_bucket_quota_size", "Maximum allowed bucket size", []string{"bucket", "owner", "zonegroup", "rgw_cluster_id", "node", "instance_id"})
	bucketQuotaMaxSizeBytes = newGaugeVec("radosgw_usage_bucket_quota_size_bytes", "Maximum allowed bucket size in bytes", []string{"bucket", "owner", "zonegroup", "rgw_cluster_id", "node", "instance_id"})
	bucketQuotaMaxObjects   = newGaugeVec("radosgw_usage_bucket_quota_size_objects", "Maximum allowed bucket size in number of objects", []string{"bucket", "owner", "zonegroup", "rgw_cluster_id", "node", "instance_id"})

	// Shards and user metadata
	bucketShards = newGaugeVec("radosgw_usage_bucket_shards", "Number of shards in bucket", []string{"bucket", "owner", "zonegroup", "rgw_cluster_id", "node", "instance_id"})
	userMetadata = newGaugeVec("radosgw_user_metadata", "User metadata", []string{"user", "display_name", "email", "storage_class", "rgw_cluster_id", "node", "instance_id"})

	// User quota metrics
	userQuotaEnabled      = newGaugeVec("radosgw_usage_user_quota_enabled", "User quota enabled", []string{"user", "rgw_cluster_id", "node", "instance_id"})
	userQuotaMaxSize      = newGaugeVec("radosgw_usage_user_quota_size", "Maximum allowed size for user", []string{"user", "rgw_cluster_id", "node", "instance_id"})
	userQuotaMaxSizeBytes = newGaugeVec("radosgw_usage_user_quota_size_bytes", "Maximum allowed size in bytes for user", []string{"user", "rgw_cluster_id", "node", "instance_id"})
	userQuotaMaxObjects   = newGaugeVec("radosgw_usage_user_quota_size_objects", "Maximum allowed number of objects across all user buckets", []string{"user", "rgw_cluster_id", "node", "instance_id"})

	// User bucket quota metrics
	userBucketQuotaEnabled      = newGaugeVec("radosgw_usage_user_bucket_quota_enabled", "User per-bucket-quota enabled", []string{"user", "rgw_cluster_id", "node", "instance_id"})
	userBucketQuotaMaxSize      = newGaugeVec("radosgw_usage_user_bucket_quota_size", "Maximum allowed size for each bucket of user", []string{"user", "rgw_cluster_id", "node", "instance_id"})
	userBucketQuotaMaxSizeBytes = newGaugeVec("radosgw_usage_user_bucket_quota_size_bytes", "Maximum allowed size in bytes for each bucket of user", []string{"user", "rgw_cluster_id", "node", "instance_id"})
	userBucketQuotaMaxObjects   = newGaugeVec("radosgw_usage_user_bucket_quota_size_objects", "Maximum allowed number of objects in each user bucket", []string{"user", "rgw_cluster_id", "node", "instance_id"})

	// Cluster-level metrics
	clusterOpsTotal             = newGaugeVec("radosgw_cluster_ops_total", "Total operations performed in the cluster", []string{"rgw_cluster_id", "node", "instance_id"})
	clusterBytesSentTotal       = newGaugeVec("radosgw_cluster_bytes_sent_total", "Total bytes sent in the cluster", []string{"rgw_cluster_id", "node", "instance_id"})
	clusterBytesReceivedTotal   = newGaugeVec("radosgw_cluster_bytes_received_total", "Total bytes received in the cluster", []string{"rgw_cluster_id", "node", "instance_id"})
	clusterCurrentOps           = newGaugeVec("radosgw_cluster_current_ops", "Current number of operations (reads/writes) in the cluster", []string{"rgw_cluster_id", "node", "instance_id"})
	clusterMaxOps               = newGaugeVec("radosgw_cluster_max_ops", "Maximum observed operations (reads/writes) in the cluster", []string{"rgw_cluster_id", "node", "instance_id"})
	clusterThroughputBytesTotal = newGaugeVec("radosgw_cluster_throughput_bytes_total", "Total throughput of the cluster in bytes (read and write combined)", []string{"rgw_cluster_id", "node", "instance_id"})
	clusterLatencySeconds       = newHistogramVec("radosgw_cluster_latency_seconds", "Latency/response times at the cluster level in seconds", []string{"rgw_cluster_id", "node", "instance_id"})

	// User-level metrics
	// Overview metrics
	userBucketsTotal  = newGaugeVec("radosgw_user_buckets_total", "Total number of buckets for each user", []string{"user", "rgw_cluster_id", "node", "instance_id"})
	userObjectsTotal  = newGaugeVec("radosgw_user_objects_total", "Total number of objects for each user", []string{"user", "rgw_cluster_id", "node", "instance_id"})
	userDataSizeTotal = newGaugeVec("radosgw_user_data_size_bytes", "Total size of data for each user in bytes", []string{"user", "rgw_cluster_id", "node", "instance_id"})
	// Total user objects and bytes
	userTotalObjects = newGaugeVec("radosgw_usage_user_total_objects", "Usage of objects by user", []string{"user", "rgw_cluster_id", "node", "instance_id"})
	userTotalBytes   = newGaugeVec("radosgw_usage_user_total_bytes", "Usage of bytes by user", []string{"user", "rgw_cluster_id", "node", "instance_id"})
	//
	userOpsTotal             = newGaugeVec("radosgw_user_ops_total", "Total operations performed by each user", []string{"user", "rgw_cluster_id", "node", "instance_id"})
	userBytesSentTotal       = newGaugeVec("radosgw_user_bytes_sent_total", "Total bytes sent by each user", []string{"user", "rgw_cluster_id", "node", "instance_id"})
	userBytesReceivedTotal   = newGaugeVec("radosgw_user_bytes_received_total", "Total bytes received by each user", []string{"user", "rgw_cluster_id", "node", "instance_id"})
	userCurrentOps           = newGaugeVec("radosgw_user_current_ops", "Current number of operations (reads/writes) for each user", []string{"user", "rgw_cluster_id", "node", "instance_id"})
	userMaxOps               = newGaugeVec("radosgw_user_max_ops", "Maximum observed operations (reads/writes) for each user", []string{"user", "rgw_cluster_id", "node", "instance_id"})
	userRequestsTotal        = newCounterVec("radosgw_user_requests_total", "Total number of requests made by each user", []string{"user", "rgw_cluster_id", "node", "instance_id"})
	userThroughputBytesTotal = newGaugeVec("radosgw_user_throughput_bytes_total", "Total throughput for each user in bytes (read and write combined)", []string{"user", "rgw_cluster_id", "node", "instance_id"})
	userLatencySeconds       = newHistogramVec("radosgw_user_latency_seconds", "Latency/response times for each user in seconds", []string{"user", "rgw_cluster_id", "node", "instance_id"})

	// Bucket-level metrics
	bucketOpsTotal             = newGaugeVec("radosgw_bucket_ops_total", "Total operations performed in each bucket", []string{"bucket", "owner", "rgw_cluster_id", "node", "instance_id"})
	bucketBytesSentTotal       = newGaugeVec("radosgw_bucket_bytes_sent_total", "Total bytes sent from each bucket", []string{"bucket", "owner", "rgw_cluster_id", "node", "instance_id"})
	bucketBytesReceivedTotal   = newGaugeVec("radosgw_bucket_bytes_received_total", "Total bytes received by each bucket", []string{"bucket", "owner", "rgw_cluster_id", "node", "instance_id"})
	bucketCurrentOps           = newGaugeVec("radosgw_bucket_current_ops", "Current number of operations (reads/writes) for each bucket", []string{"bucket", "owner", "rgw_cluster_id", "node", "instance_id"})
	bucketMaxOps               = newGaugeVec("radosgw_bucket_max_ops", "Maximum observed operations (reads/writes) for each bucket", []string{"bucket", "owner", "rgw_cluster_id", "node", "instance_id"})
	bucketRequestsTotal        = newCounterVec("radosgw_bucket_requests_total", "Total number of requests made to each bucket", []string{"bucket", "owner", "rgw_cluster_id", "node", "instance_id"})
	bucketThroughputBytesTotal = newGaugeVec("radosgw_bucket_throughput_bytes_total", "Total throughput for each bucket in bytes (read and write combined)", []string{"bucket", "owner", "rgw_cluster_id", "node", "instance_id"})
	bucketLatencySeconds       = newHistogramVec("radosgw_bucket_latency_seconds", "Latency/response times for each bucket in seconds", []string{"bucket", "owner", "rgw_cluster_id", "node", "instance_id"})

	// Miscellaneous metrics
	scrapeDurationSeconds = newGaugeVec("radosgw_usage_scrape_duration_seconds", "Amount of time each scrape takes", []string{})
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
	prometheus.MustRegister(opsTotal)
	prometheus.MustRegister(successfulOpsTotal)
	prometheus.MustRegister(bytesSentTotal)
	prometheus.MustRegister(bytesReceivedTotal)
	prometheus.MustRegister(bucketUsageBytes)
	prometheus.MustRegister(bucketUtilizedBytes)
	prometheus.MustRegister(bucketUsageObjects)
	prometheus.MustRegister(bucketQuotaEnabled)
	prometheus.MustRegister(bucketQuotaMaxSize)
	prometheus.MustRegister(bucketQuotaMaxSizeBytes)
	prometheus.MustRegister(bucketQuotaMaxObjects)
	prometheus.MustRegister(bucketShards)
	prometheus.MustRegister(userMetadata)
	prometheus.MustRegister(userQuotaEnabled)
	prometheus.MustRegister(userQuotaMaxSize)
	prometheus.MustRegister(userQuotaMaxSizeBytes)
	prometheus.MustRegister(userQuotaMaxObjects)
	prometheus.MustRegister(userBucketQuotaEnabled)
	prometheus.MustRegister(userBucketQuotaMaxSize)
	prometheus.MustRegister(userBucketQuotaMaxSizeBytes)
	prometheus.MustRegister(userBucketQuotaMaxObjects)
	prometheus.MustRegister(userTotalObjects)
	prometheus.MustRegister(userTotalBytes)
	prometheus.MustRegister(userBucketsTotal)
	prometheus.MustRegister(userObjectsTotal)
	prometheus.MustRegister(userDataSizeTotal)
	prometheus.MustRegister(clusterOpsTotal)
	prometheus.MustRegister(clusterBytesSentTotal)
	prometheus.MustRegister(clusterBytesReceivedTotal)
	prometheus.MustRegister(clusterCurrentOps)
	prometheus.MustRegister(clusterMaxOps)
	prometheus.MustRegister(clusterThroughputBytesTotal)
	prometheus.MustRegister(clusterLatencySeconds)
	prometheus.MustRegister(userOpsTotal)
	prometheus.MustRegister(userBytesSentTotal)
	prometheus.MustRegister(userBytesReceivedTotal)
	prometheus.MustRegister(userCurrentOps)
	prometheus.MustRegister(userMaxOps)
	prometheus.MustRegister(userRequestsTotal)
	prometheus.MustRegister(userThroughputBytesTotal)
	prometheus.MustRegister(userLatencySeconds)
	prometheus.MustRegister(bucketOpsTotal)
	prometheus.MustRegister(bucketBytesSentTotal)
	prometheus.MustRegister(bucketBytesReceivedTotal)
	prometheus.MustRegister(bucketCurrentOps)
	prometheus.MustRegister(bucketMaxOps)
	prometheus.MustRegister(bucketRequestsTotal)
	prometheus.MustRegister(bucketThroughputBytesTotal)
	prometheus.MustRegister(bucketLatencySeconds)
	prometheus.MustRegister(scrapeDurationSeconds)
}

func publishToPrometheus(entries []UsageEntry, scrapeDuration float64, cfg RadosGWUsageConfig) {
	var (
		totalClusterOps             uint64
		totalClusterBytesSent       uint64
		totalClusterBytesReceived   uint64
		totalClusterThroughputBytes uint64
		totalClusterLatencySeconds  float64
		totalClusterCurrentOps      uint64
		totalClusterMaxOps          uint64
	)

	for _, entry := range entries {
		aggregateUserMetrics(&entry)
		populateBucketMetrics(entry, &cfg)
		populateUserMetrics(entry, &cfg)
		aggregateClusterMetrics(
			entry,
			&totalClusterOps,
			&totalClusterBytesSent,
			&totalClusterBytesReceived,
			&totalClusterThroughputBytes,
			&totalClusterLatencySeconds,
			&totalClusterCurrentOps,
			&totalClusterMaxOps,
		)
	}
	// Populate cluster-level metrics after aggregating all entries
	populateClusterMetrics(
		cfg.ClusterID,
		cfg.NodeName,
		cfg.InstanceID,
		totalClusterOps,
		totalClusterBytesSent,
		totalClusterBytesReceived,
		totalClusterThroughputBytes,
		totalClusterLatencySeconds,
		totalClusterCurrentOps,
		totalClusterMaxOps,
	)

	scrapeDurationSeconds.With(prometheus.Labels{}).Set(scrapeDuration)
}

func aggregateUserMetrics(entry *UsageEntry) {
	entry.TotalBuckets = len(entry.Buckets)
	entry.TotalObjects = 0
	entry.TotalDataSize = 0
	entry.TotalOps = 0
	entry.TotalBytesSent = 0
	entry.TotalBytesReceived = 0

	for _, bucket := range entry.Buckets {
		entry.TotalObjects += *bucket.Usage.RgwMain.NumObjects
		entry.TotalDataSize += *bucket.Usage.RgwMain.SizeUtilized
		entry.TotalOps += bucket.TotalOps
		entry.TotalBytesSent += bucket.TotalBytesSent
		entry.TotalBytesReceived += bucket.TotalBytesReceived
	}
}

func populateBucketMetrics(entry UsageEntry, cfg *RadosGWUsageConfig) {
	for _, bucket := range entry.Buckets {
		labels := prometheus.Labels{
			"bucket":         bucket.Bucket,
			"owner":          bucket.Owner,
			"rgw_cluster_id": cfg.ClusterID,
			"node":           cfg.NodeName,
			"instance_id":    cfg.InstanceID,
		}
		bucketOpsTotal.With(labels).Set(float64(bucket.TotalOps))
		bucketBytesSentTotal.With(labels).Set(float64(bucket.TotalBytesSent))
		bucketBytesReceivedTotal.With(labels).Set(float64(bucket.TotalBytesReceived))
		bucketThroughputBytesTotal.With(labels).Add(float64(bucket.TotalThroughputBytes))
		bucketLatencySeconds.With(labels).Observe(bucket.TotalLatencySeconds)
		bucketCurrentOps.With(labels).Set(float64(bucket.CurrentOps))
		bucketMaxOps.With(labels).Set(float64(bucket.MaxOps))

		for _, category := range bucket.Categories {
			categoryLabels := prometheus.Labels{
				"bucket":         bucket.Bucket,
				"owner":          bucket.Owner,
				"category":       category.Category,
				"rgw_cluster_id": cfg.ClusterID,
				"node":           cfg.NodeName,
				"instance_id":    cfg.InstanceID,
			}
			opsTotal.With(categoryLabels).Add(float64(category.Ops))
			successfulOpsTotal.With(categoryLabels).Add(float64(category.SuccessfulOps))
			bytesSentTotal.With(categoryLabels).Add(float64(category.BytesSent))
			bytesReceivedTotal.With(categoryLabels).Add(float64(category.BytesReceived))
		}

		bucketLabels := prometheus.Labels{
			"bucket":         bucket.Bucket,
			"owner":          bucket.Owner,
			"zonegroup":      bucket.Zonegroup,
			"rgw_cluster_id": cfg.ClusterID,
			"node":           cfg.NodeName,
			"instance_id":    cfg.InstanceID,
		}
		bucketUsageBytes.With(bucketLabels).Set(float64(*bucket.Usage.RgwMain.Size))
		bucketUtilizedBytes.With(bucketLabels).Set(float64(*bucket.Usage.RgwMain.SizeUtilized))
		bucketUsageObjects.With(bucketLabels).Set(float64(*bucket.Usage.RgwMain.NumObjects))
		bucketQuotaEnabled.With(bucketLabels).Set(boolToFloat64(bucket.BucketQuota.Enabled))
		bucketQuotaMaxSize.With(bucketLabels).Set(float64(*bucket.BucketQuota.MaxSize))
		bucketQuotaMaxSizeBytes.With(bucketLabels).Set(float64(*bucket.BucketQuota.MaxSizeKb) * 1024)
		bucketQuotaMaxObjects.With(bucketLabels).Set(float64(*bucket.BucketQuota.MaxObjects))
		bucketShards.With(bucketLabels).Set(float64(bucket.NumShards))
	}
}

func populateUserMetrics(entry UsageEntry, cfg *RadosGWUsageConfig) {
	labels := prometheus.Labels{
		"user":           entry.User,
		"rgw_cluster_id": cfg.ClusterID,
		"node":           cfg.NodeName,
		"instance_id":    cfg.InstanceID,
	}
	userBucketsTotal.With(labels).Set(float64(entry.TotalBuckets))
	userObjectsTotal.With(labels).Set(float64(entry.TotalObjects))
	userDataSizeTotal.With(labels).Set(float64(entry.TotalDataSize))
	userOpsTotal.With(labels).Set(float64(entry.TotalOps))
	userBytesSentTotal.With(labels).Set(float64(entry.TotalBytesSent))
	userBytesReceivedTotal.With(labels).Set(float64(entry.TotalBytesReceived))
	userThroughputBytesTotal.With(labels).Add(float64(entry.TotalThroughputBytes))
	userLatencySeconds.With(labels).Observe(entry.TotalLatencySeconds)
	userCurrentOps.With(labels).Set(float64(entry.CurrentOps))
	userMaxOps.With(labels).Set(float64(entry.MaxOps))

	userMetadata.With(prometheus.Labels{
		"user":           entry.User,
		"display_name":   entry.DisplayName,
		"email":          entry.Email,
		"storage_class":  entry.DefaultStorageClass,
		"rgw_cluster_id": cfg.ClusterID,
		"node":           cfg.NodeName,
		"instance_id":    cfg.InstanceID,
	}).Set(1)
	userQuotaEnabled.With(labels).Set(boolToFloat64(entry.UserQuota.Enabled))

	if entry.UserQuota.MaxSize != nil {
		userQuotaMaxSize.With(labels).Set(float64(*entry.UserQuota.MaxSize))
	}
	if entry.UserQuota.MaxSizeKb != nil {
		userQuotaMaxSizeBytes.With(labels).Set(float64(*entry.UserQuota.MaxSizeKb) * 1024)
	}
	if entry.UserQuota.MaxObjects != nil {
		userQuotaMaxObjects.With(labels).Set(float64(*entry.UserQuota.MaxObjects))
	}
	if entry.Stats.NumObjects != nil {
		userTotalObjects.With(labels).Set(float64(*entry.Stats.NumObjects))
	}
	if entry.Stats.Size != nil {
		userTotalBytes.With(labels).Set(float64(*entry.Stats.Size))
	}
}

func aggregateClusterMetrics(entry UsageEntry, totalOps, totalBytesSent, totalBytesReceived, totalThroughputBytes *uint64, totalLatencySeconds *float64, totalCurrentOps, totalMaxOps *uint64) {
	*totalOps += entry.TotalOps
	*totalBytesSent += entry.TotalBytesSent
	*totalBytesReceived += entry.TotalBytesReceived
	*totalThroughputBytes += entry.TotalThroughputBytes
	*totalLatencySeconds += entry.TotalLatencySeconds
	*totalCurrentOps += entry.CurrentOps
	if entry.MaxOps > *totalMaxOps {
		*totalMaxOps = entry.MaxOps
	}
}

func populateClusterMetrics(clusterID, node, instanceID string, totalOps, totalBytesSent, totalBytesReceived, totalThroughputBytes uint64, totalLatencySeconds float64, totalCurrentOps, totalMaxOps uint64) {
	labels := prometheus.Labels{
		"rgw_cluster_id": clusterID,
		"node":           node,
		"instance_id":    instanceID,
	}
	clusterOpsTotal.With(labels).Set(float64(totalOps))
	clusterBytesSentTotal.With(labels).Set(float64(totalBytesSent))
	clusterBytesReceivedTotal.With(labels).Set(float64(totalBytesReceived))
	clusterThroughputBytesTotal.With(labels).Add(float64(totalThroughputBytes))
	clusterLatencySeconds.With(labels).Observe(totalLatencySeconds)
	clusterCurrentOps.With(labels).Set(float64(totalCurrentOps))
	clusterMaxOps.With(labels).Set(float64(totalMaxOps))
}

func boolToFloat64(b *bool) float64 {
	if b != nil && *b {
		return 1.0
	}
	return 0.0
}

func startPrometheusMetricsServer(port int) {
	http.Handle("/metrics", promhttp.Handler())
	http.ListenAndServe(":"+strconv.Itoa(port), nil)
}
