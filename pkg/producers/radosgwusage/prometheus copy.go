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

// import (
// 	"net/http"
// 	"strconv"

// 	"github.com/prometheus/client_golang/prometheus"
// 	"github.com/prometheus/client_golang/prometheus/promhttp"
// )

// var (
// 	// User-level metrics
// 	userMetadata             = newGaugeVec("radosgw_user_metadata", "User metadata", []string{"user", "display_name", "email", "storage_class", "rgw_cluster_id", "node", "instance_id"})
// 	userBucketsTotal         = newGaugeVec("radosgw_user_buckets_total", "Total number of buckets for each user", []string{"user", "rgw_cluster_id", "node", "instance_id"})
// 	userObjectsTotal         = newGaugeVec("radosgw_user_objects_total", "Total number of objects for each user", []string{"user", "rgw_cluster_id", "node", "instance_id"})
// 	userDataSizeTotal        = newGaugeVec("radosgw_user_data_size_bytes", "Total size of data for each user in bytes", []string{"user", "rgw_cluster_id", "node", "instance_id"})
// 	userOpsTotal             = newGaugeVec("radosgw_user_ops_total", "Total operations performed by each user", []string{"user", "rgw_cluster_id", "node", "instance_id"})
// 	userReadOpsTotal         = newGaugeVec("radosgw_user_read_ops_total", "Total read operations per user", []string{"user", "rgw_cluster_id", "node", "instance_id"})
// 	userWriteOpsTotal        = newGaugeVec("radosgw_user_write_ops_total", "Total write operations per user", []string{"user", "rgw_cluster_id", "node", "instance_id"})
// 	userBytesSentTotal       = newGaugeVec("radosgw_user_bytes_sent_total", "Total bytes sent by each user (cumulative)", []string{"user", "rgw_cluster_id", "node", "instance_id"})
// 	userBytesReceivedTotal   = newGaugeVec("radosgw_user_bytes_received_total", "Total bytes received by each user (cumulative)", []string{"user", "rgw_cluster_id", "node", "instance_id"})
// 	userSuccessOpsTotal      = newGaugeVec("radosgw_user_success_ops_total", "Total successful operations per user", []string{"user", "rgw_cluster_id", "node", "instance_id"})
// 	userErrorRateTotal       = newGaugeVec("radosgw_user_error_rate_total", "Total number of errors per user", []string{"user", "rgw_cluster_id", "node", "instance_id"})
// 	userThroughputBytesTotal = newGaugeVec("radosgw_user_throughput_bytes_total", "Total throughput for each user in bytes (read and write combined)", []string{"user", "rgw_cluster_id", "node", "instance_id"})
// 	userCapacityUsage        = newGaugeVec("radosgw_user_capacity_usage_bytes", "Total capacity used by each user in bytes", []string{"user", "rgw_cluster_id", "node", "instance_id"})

// 	userOpsPerSec             = newGaugeVec("radosgw_user_ops_per_sec", "Current number of operations (reads/writes) per second for each user (rate)", []string{"user", "rgw_cluster_id", "node", "instance_id"})
// 	userReadOpsPerSec         = newGaugeVec("radosgw_user_read_ops_per_sec", "Current read operations per second for each user", []string{"user", "rgw_cluster_id", "node", "instance_id"})
// 	userWriteOpsPerSec        = newGaugeVec("radosgw_user_write_ops_per_sec", "Current write operations per second for each user", []string{"user", "rgw_cluster_id", "node", "instance_id"})
// 	userBytesReceivedPerSec   = newGaugeVec("radosgw_user_bytes_received_per_sec", "Bytes received by each user per second (rate)", []string{"user", "rgw_cluster_id", "node", "instance_id"})
// 	userBytesSentPerSec       = newGaugeVec("radosgw_user_bytes_sent_per_sec", "Bytes sent by each user per second (rate)", []string{"user", "rgw_cluster_id", "node", "instance_id"})
// 	userThroughputBytesPerSec = newGaugeVec("radosgw_user_throughput_bytes_per_sec", "Current throughput in bytes per second for each user (read and write combined)", []string{"user", "rgw_cluster_id", "node", "instance_id"})

// 	// User quota metrics
// 	userQuotaEnabled    = newGaugeVec("radosgw_usage_user_quota_enabled", "User quota enabled", []string{"user", "rgw_cluster_id", "node", "instance_id"})
// 	userQuotaMaxSize    = newGaugeVec("radosgw_usage_user_quota_size", "Maximum allowed size for user", []string{"user", "rgw_cluster_id", "node", "instance_id"})
// 	userQuotaMaxObjects = newGaugeVec("radosgw_usage_user_quota_size_objects", "Maximum allowed number of objects across all user buckets", []string{"user", "rgw_cluster_id", "node", "instance_id"})

// 	apiUsagePerUser            = newGaugeVec("radosgw_api_usage_per_user", "API usage per user and per category", []string{"user", "api_category", "rgw_cluster_id", "node", "instance_id"})
// 	apiUsagePerUserPerSec      = newGaugeVec("radosgw_api_usage_per_user_per_sec", "API usage per second per user and category", []string{"user", "api_category", "rgw_cluster_id", "node", "instance_id"})
// 	apiUsagePerUserTotalPerSec = newGaugeVec("radosgw_api_usage_per_user_total_per_sec", "Total API usage per second for each user", []string{"user", "rgw_cluster_id", "node", "instance_id"})

// 	// Bucket-level metrics
// 	bucketLabels               = []string{"bucket", "owner", "zonegroup", "rgw_cluster_id", "node", "instance_id"}
// 	bucketUsageBytes           = newGaugeVec("radosgw_usage_bucket_bytes", "Bucket used bytes", bucketLabels)
// 	bucketUtilizedBytes        = newGaugeVec("radosgw_usage_bucket_utilized_bytes", "Bucket utilized bytes", bucketLabels)
// 	bucketUsageObjects         = newGaugeVec("radosgw_usage_bucket_objects", "Number of objects in bucket", bucketLabels)
// 	bucketReadOpsTotal         = newGaugeVec("radosgw_bucket_read_ops_total", "Total read operations in each bucket", bucketLabels)
// 	bucketWriteOpsTotal        = newGaugeVec("radosgw_bucket_write_ops_total", "Total write operations in each bucket", bucketLabels)
// 	bucketBytesSentTotal       = newGaugeVec("radosgw_bucket_bytes_sent_total", "Total bytes sent from each bucket", bucketLabels)
// 	bucketBytesReceivedTotal   = newGaugeVec("radosgw_bucket_bytes_received_total", "Total bytes received by each bucket", bucketLabels)
// 	bucketSuccessOpsTotal      = newGaugeVec("radosgw_bucket_success_ops_total", "Total successful operations for each bucket", bucketLabels)
// 	bucketOpsTotal             = newGaugeVec("radosgw_bucket_ops_total", "Total operations performed in each bucket", bucketLabels)
// 	bucketErrorRate            = newGaugeVec("radosgw_bucket_error_rate", "Error rate for each bucket (percentage)", bucketLabels)
// 	bucketCapacityUsage        = newGaugeVec("radosgw_bucket_capacity_usage_bytes", "Total capacity used by each bucket in bytes", bucketLabels)
// 	bucketThroughputBytesTotal = newGaugeVec("radosgw_bucket_throughput_bytes_total", "Total throughput for each bucket in bytes (read and write combined)", bucketLabels)
// 	bucketAPIUsageTotal        = newGaugeVec("radosgw_bucket_api_usage_total", "Total number of API operations by category for each bucket", []string{"bucket", "owner", "zonegroup", "rgw_cluster_id", "node", "instance_id", "category"})

// 	bucketOpsPerSec             = newGaugeVec("radosgw_bucket_ops_per_sec", "Current operations per second for each bucket (rate)", bucketLabels)
// 	bucketReadOpsPerSec         = newGaugeVec("radosgw_bucket_read_ops_per_sec", "Current read operations per second for each bucket (rate)", bucketLabels)
// 	bucketWriteOpsPerSec        = newGaugeVec("radosgw_bucket_write_ops_per_sec", "Current write operations per second for each bucket (rate)", bucketLabels)
// 	bucketBytesSentPerSec       = newGaugeVec("radosgw_bucket_bytes_sent_per_sec", "Current bytes sent per second from each bucket", bucketLabels)
// 	bucketBytesReceivedPerSec   = newGaugeVec("radosgw_bucket_bytes_received_per_sec", "Current bytes received per second by each bucket", bucketLabels)
// 	bucketThroughputBytesPerSec = newGaugeVec("radosgw_bucket_throughput_bytes_per_sec", "Current throughput in bytes per second for each bucket (read and write combined)", bucketLabels)
// 	bucketAPIUsagePerSec        = newGaugeVec("radosgw_bucket_api_usage_per_sec", "Current API usage rate (ops per second) for each bucket and category.", []string{"bucket", "owner", "zonegroup", "rgw_cluster_id", "node", "instance_id", "category"})
// 	bucketAPIUsageTotalPerSec   = newGaugeVec("radosgw_bucket_api_usage_total_per_sec", "Total API usage per second for each bucket", []string{"bucket", "owner", "zonegroup", "rgw_cluster_id", "node", "instance_id"})

// 	// Quota metrics
// 	bucketQuotaEnabled    = newGaugeVec("radosgw_usage_bucket_quota_enabled", "Quota enabled for bucket", bucketLabels)
// 	bucketQuotaMaxSize    = newGaugeVec("radosgw_usage_bucket_quota_size", "Maximum allowed bucket size", bucketLabels)
// 	bucketQuotaMaxObjects = newGaugeVec("radosgw_usage_bucket_quota_size_objects", "Maximum allowed bucket size in number of objects", bucketLabels)

// 	// Shards and user metadata
// 	bucketShards = newGaugeVec("radosgw_usage_bucket_shards", "Number of shards in bucket", []string{"bucket", "owner", "zonegroup", "rgw_cluster_id", "node", "instance_id"})

// 	// Cluster-level metrics
// 	clusterOpsTotal             = newGaugeVec("radosgw_cluster_ops_total", "Total operations performed in the cluster", []string{"rgw_cluster_id", "node", "instance_id"})
// 	clusterBytesSentTotal       = newGaugeVec("radosgw_cluster_bytes_sent_total", "Total bytes sent in the cluster", []string{"rgw_cluster_id", "node", "instance_id"})
// 	clusterBytesReceivedTotal   = newGaugeVec("radosgw_cluster_bytes_received_total", "Total bytes received in the cluster", []string{"rgw_cluster_id", "node", "instance_id"})
// 	clusterThroughputBytesTotal = newGaugeVec("radosgw_cluster_throughput_bytes_total", "Total throughput of the cluster in bytes (read and write combined)", []string{"rgw_cluster_id", "node", "instance_id"})

// 	clusterOpsPerSec             = newGaugeVec("radosgw_cluster_ops_per_sec", "Current number of operations per second for the cluster", []string{"rgw_cluster_id", "node", "instance_id"})
// 	clusterReadsPerSec           = newGaugeVec("radosgw_cluster_reads_per_sec", "Total read operations per second for the entire cluster", []string{"rgw_cluster_id", "node", "instance_id"})
// 	clusterWritesPerSec          = newGaugeVec("radosgw_cluster_writes_per_sec", "Total write operations per second for the entire cluster", []string{"rgw_cluster_id", "node", "instance_id"})
// 	clusterBytesSentPerSec       = newGaugeVec("radosgw_cluster_bytes_sent_per_sec", "Total bytes sent per second for the entire cluster", []string{"rgw_cluster_id", "node", "instance_id"})
// 	clusterBytesReceivedPerSec   = newGaugeVec("radosgw_cluster_bytes_received_per_sec", "Total bytes received per second for the entire cluster", []string{"rgw_cluster_id", "node", "instance_id"})
// 	clusterThroughputBytesPerSec = newGaugeVec("radosgw_cluster_throughput_bytes_per_sec", "Total throughput (read and write) in bytes per second for the entire cluster", []string{"rgw_cluster_id", "node", "instance_id"})
// 	clusterErrorRate             = newGaugeVec("radosgw_cluster_error_rate", "Error rate (percentage) for the entire cluster", []string{"rgw_cluster_id", "node", "instance_id"})
// 	clusterCapacityUsageBytes    = newGaugeVec("radosgw_cluster_capacity_usage_bytes", "Total capacity used across the entire cluster in bytes", []string{"rgw_cluster_id", "node", "instance_id"})
// 	clusterSuccessOpsTotal       = newGaugeVec("radosgw_cluster_success_ops_total", "Total successful operations across the entire cluster", []string{"rgw_cluster_id", "node", "instance_id"})

// 	// Miscellaneous metrics
// 	scrapeDurationSeconds = newGaugeVec("radosgw_usage_scrape_duration_seconds", "Amount of time each scrape takes", []string{})
// )

// func newCounterVec(name, help string, labels []string) *prometheus.CounterVec {
// 	return prometheus.NewCounterVec(prometheus.CounterOpts{
// 		Name: name,
// 		Help: help,
// 	}, labels)
// }

// func newGaugeVec(name, help string, labels []string) *prometheus.GaugeVec {
// 	return prometheus.NewGaugeVec(prometheus.GaugeOpts{
// 		Name: name,
// 		Help: help,
// 	}, labels)
// }

// func newHistogramVec(name, help string, labels []string) *prometheus.HistogramVec {
// 	return prometheus.NewHistogramVec(prometheus.HistogramOpts{
// 		Name:    name,
// 		Help:    help,
// 		Buckets: prometheus.DefBuckets,
// 	}, labels)
// }

// func init() {
// 	// Register all metrics with Prometheus's default registry
// 	prometheus.MustRegister(userMetadata)
// 	prometheus.MustRegister(userBucketsTotal)
// 	prometheus.MustRegister(userObjectsTotal)
// 	prometheus.MustRegister(userDataSizeTotal)
// 	prometheus.MustRegister(userOpsTotal)
// 	prometheus.MustRegister(userReadOpsTotal)
// 	prometheus.MustRegister(userWriteOpsTotal)
// 	prometheus.MustRegister(userBytesSentTotal)
// 	prometheus.MustRegister(userBytesReceivedTotal)
// 	prometheus.MustRegister(userSuccessOpsTotal)
// 	prometheus.MustRegister(userErrorRateTotal)
// 	prometheus.MustRegister(userThroughputBytesTotal)
// 	prometheus.MustRegister(userCapacityUsage)

// 	prometheus.MustRegister(userOpsPerSec)
// 	prometheus.MustRegister(userReadOpsPerSec)
// 	prometheus.MustRegister(userWriteOpsPerSec)
// 	prometheus.MustRegister(userBytesReceivedPerSec)
// 	prometheus.MustRegister(userBytesSentPerSec)
// 	prometheus.MustRegister(userThroughputBytesPerSec)

// 	prometheus.MustRegister(userQuotaEnabled)
// 	prometheus.MustRegister(userQuotaMaxSize)
// 	prometheus.MustRegister(userQuotaMaxObjects)

// 	prometheus.MustRegister(apiUsagePerUser)
// 	prometheus.MustRegister(apiUsagePerUserPerSec)
// 	prometheus.MustRegister(apiUsagePerUserTotalPerSec)

// 	////
// 	prometheus.MustRegister(bucketUsageBytes)
// 	prometheus.MustRegister(bucketUtilizedBytes)
// 	prometheus.MustRegister(bucketUsageObjects)
// 	prometheus.MustRegister(bucketReadOpsTotal)
// 	prometheus.MustRegister(bucketWriteOpsTotal)
// 	prometheus.MustRegister(bucketBytesSentTotal)
// 	prometheus.MustRegister(bucketBytesReceivedTotal)
// 	prometheus.MustRegister(bucketSuccessOpsTotal)
// 	prometheus.MustRegister(bucketOpsTotal)
// 	prometheus.MustRegister(bucketErrorRate)
// 	prometheus.MustRegister(bucketCapacityUsage)
// 	prometheus.MustRegister(bucketThroughputBytesTotal)
// 	prometheus.MustRegister(bucketAPIUsageTotal)

// 	prometheus.MustRegister(bucketOpsPerSec)
// 	prometheus.MustRegister(bucketReadOpsPerSec)
// 	prometheus.MustRegister(bucketWriteOpsPerSec)
// 	prometheus.MustRegister(bucketBytesSentPerSec)
// 	prometheus.MustRegister(bucketBytesReceivedPerSec)
// 	prometheus.MustRegister(bucketThroughputBytesPerSec)
// 	prometheus.MustRegister(bucketAPIUsagePerSec)
// 	prometheus.MustRegister(bucketAPIUsageTotalPerSec)

// 	prometheus.MustRegister(bucketQuotaEnabled)
// 	prometheus.MustRegister(bucketQuotaMaxSize)
// 	prometheus.MustRegister(bucketQuotaMaxObjects)
// 	prometheus.MustRegister(bucketShards)

// 	prometheus.MustRegister(clusterOpsTotal)
// 	prometheus.MustRegister(clusterBytesSentTotal)
// 	prometheus.MustRegister(clusterBytesReceivedTotal)
// 	prometheus.MustRegister(clusterThroughputBytesTotal)

// 	prometheus.MustRegister(clusterOpsPerSec)
// 	prometheus.MustRegister(clusterReadsPerSec)
// 	prometheus.MustRegister(clusterWritesPerSec)
// 	prometheus.MustRegister(clusterBytesSentPerSec)
// 	prometheus.MustRegister(clusterBytesReceivedPerSec)
// 	prometheus.MustRegister(clusterThroughputBytesPerSec)
// 	prometheus.MustRegister(clusterErrorRate)
// 	prometheus.MustRegister(clusterCapacityUsageBytes)
// 	prometheus.MustRegister(clusterSuccessOpsTotal)

// 	prometheus.MustRegister(scrapeDurationSeconds)
// }

// func publishToPrometheus(entries []UsageEntry, scrapeDuration float64, cfg RadosGWUsageConfig) {
// 	clusterMetrics := RadosGWClusterMetrics{}

// 	for _, entry := range entries {
// 		populateBucketMetrics(entry, &cfg)
// 		populateUserMetrics(entry, &cfg)
// 		aggregateClusterMetrics(entry, &clusterMetrics)
// 	}
// 	// Populate cluster-level metrics after aggregating all entries
// 	populateClusterMetrics(cfg.ClusterID, cfg.NodeName, cfg.InstanceID, clusterMetrics)

// 	scrapeDurationSeconds.With(prometheus.Labels{}).Set(scrapeDuration)
// }

// func populateBucketMetrics(entry UsageEntry, cfg *RadosGWUsageConfig) {
// 	for _, bucket := range entry.Buckets {
// 		bucketLabels := prometheus.Labels{
// 			"bucket":         bucket.Meta.Name,
// 			"owner":          bucket.Meta.Owner,
// 			"zonegroup":      bucket.Meta.Zonegroup,
// 			"rgw_cluster_id": cfg.ClusterID,
// 			"node":           cfg.NodeName,
// 			"instance_id":    cfg.InstanceID,
// 		}
// 		bucketUsageBytes.With(bucketLabels).Set(float64(bucket.Totals.DataSize))
// 		bucketUtilizedBytes.With(bucketLabels).Set(float64(bucket.Totals.UtilizedSize))
// 		bucketUsageObjects.With(bucketLabels).Set(float64(bucket.Totals.Objects))
// 		bucketReadOpsTotal.With(bucketLabels).Set(float64(bucket.Totals.ReadOps))
// 		bucketWriteOpsTotal.With(bucketLabels).Set(float64(bucket.Totals.WriteOps))
// 		bucketBytesSentTotal.With(bucketLabels).Set(float64(bucket.Totals.BytesSent))
// 		bucketBytesReceivedTotal.With(bucketLabels).Set(float64(bucket.Totals.BytesReceived))
// 		bucketSuccessOpsTotal.With(bucketLabels).Set(float64(bucket.Totals.SuccessOps))
// 		bucketOpsTotal.With(bucketLabels).Set(float64(bucket.Totals.OpsTotal))
// 		bucketErrorRate.With(bucketLabels).Set(float64(bucket.Totals.ErrorRate))
// 		bucketCapacityUsage.With(bucketLabels).Set(float64(bucket.Totals.Capacity))
// 		bucketThroughputBytesTotal.With(bucketLabels).Set(float64(bucket.Totals.BytesSent + bucket.Totals.BytesReceived))

// 		bucketOpsPerSec.With(bucketLabels).Set(bucket.Current.OpsPerSec)
// 		bucketReadOpsPerSec.With(bucketLabels).Set(bucket.Current.ReadOpsPerSec)
// 		bucketWriteOpsPerSec.With(bucketLabels).Set(bucket.Current.WriteOpsPerSec)
// 		bucketBytesSentPerSec.With(bucketLabels).Set(bucket.Current.BytesSentPerSec)
// 		bucketBytesReceivedPerSec.With(bucketLabels).Set(bucket.Current.BytesReceivedPerSec)
// 		bucketThroughputBytesPerSec.With(bucketLabels).Set(bucket.Current.ThroughputBytesPerSec)

// 		// Set quota information
// 		bucketQuotaEnabled.With(bucketLabels).Set(boolToFloat64(&bucket.Quota.Enabled))
// 		if bucket.Quota.MaxSize != nil {
// 			bucketQuotaMaxSize.With(bucketLabels).Set(float64(*bucket.Quota.MaxSize))
// 		}
// 		if bucket.Quota.MaxObjects != nil {
// 			bucketQuotaMaxObjects.With(bucketLabels).Set(float64(*bucket.Quota.MaxObjects))
// 		}

// 		// Set shards
// 		if bucket.Meta.Shards != nil {
// 			bucketShards.With(bucketLabels).Set(float64(*bucket.Meta.Shards))
// 		}

// 		// Set API usage per bucket (instead of categories)
// 		for category, ops := range bucket.APIUsage {
// 			apiLabels := prometheus.Labels{
// 				"bucket":         bucket.Meta.Name,
// 				"owner":          bucket.Meta.Owner,
// 				"zonegroup":      bucket.Meta.Zonegroup,
// 				"rgw_cluster_id": cfg.ClusterID,
// 				"node":           cfg.NodeName,
// 				"instance_id":    cfg.InstanceID,
// 				"category":       category,
// 			}
// 			bucketAPIUsageTotal.With(apiLabels).Add(float64(ops)) // Total API ops for the category
// 		}

// 		// Set API usage per second for each category
// 		for category, rate := range bucket.Current.APIUsage {
// 			apiUsageLabels := prometheus.Labels{
// 				"bucket":         bucket.Meta.Name,
// 				"owner":          bucket.Meta.Owner,
// 				"zonegroup":      bucket.Meta.Zonegroup,
// 				"rgw_cluster_id": cfg.ClusterID,
// 				"node":           cfg.NodeName,
// 				"instance_id":    cfg.InstanceID,
// 				"category":       category,
// 			}
// 			bucketAPIUsagePerSec.With(apiUsageLabels).Set(rate)
// 		}

// 		// Total API usage per second for the bucket
// 		totalAPILabels := prometheus.Labels{
// 			"bucket":         bucket.Meta.Name,
// 			"owner":          bucket.Meta.Owner,
// 			"zonegroup":      bucket.Meta.Zonegroup,
// 			"rgw_cluster_id": cfg.ClusterID,
// 			"node":           cfg.NodeName,
// 			"instance_id":    cfg.InstanceID,
// 		}
// 		bucketAPIUsageTotalPerSec.With(totalAPILabels).Set(bucket.Current.TotalAPIUsagePerSec)
// 	}
// }

// func populateUserMetrics(entry UsageEntry, cfg *RadosGWUsageConfig) {
// 	userMetadata.With(prometheus.Labels{
// 		"user":           entry.UserLevel.Meta.ID,
// 		"display_name":   entry.UserLevel.Meta.DisplayName,
// 		"email":          entry.UserLevel.Meta.Email,
// 		"storage_class":  entry.UserLevel.Meta.DefaultStorageClass,
// 		"rgw_cluster_id": cfg.ClusterID,
// 		"node":           cfg.NodeName,
// 		"instance_id":    cfg.InstanceID,
// 	}).Set(1)

// 	labels := prometheus.Labels{
// 		"user":           entry.UserLevel.Meta.ID,
// 		"rgw_cluster_id": cfg.ClusterID,
// 		"node":           cfg.NodeName,
// 		"instance_id":    cfg.InstanceID,
// 	}
// 	userBucketsTotal.With(labels).Set(float64(entry.UserLevel.Totals.BucketsTotal))
// 	userObjectsTotal.With(labels).Set(float64((entry.UserLevel.Totals.ObjectsTotal)))
// 	userDataSizeTotal.With(labels).Set(float64(entry.UserLevel.Totals.DataSizeTotal))
// 	userOpsTotal.With(labels).Set(float64(entry.UserLevel.Totals.OpsTotal))
// 	userReadOpsTotal.With(labels).Set(float64(entry.UserLevel.Totals.ReadOpsTotal))
// 	userWriteOpsTotal.With(labels).Set(float64(entry.UserLevel.Totals.WriteOpsTotal))
// 	userBytesSentTotal.With(labels).Set(float64(entry.UserLevel.Totals.BytesSentTotal))
// 	userBytesReceivedTotal.With(labels).Set(float64(entry.UserLevel.Totals.BytesReceivedTotal))
// 	userSuccessOpsTotal.With(labels).Set(float64(entry.UserLevel.Totals.SuccessOpsTotal))
// 	userErrorRateTotal.With(labels).Set(float64(entry.UserLevel.Totals.ErrorRateTotal))
// 	userThroughputBytesTotal.With(labels).Add(float64(entry.UserLevel.Totals.ThroughputBytesTotal))
// 	userCapacityUsage.With(labels).Set(float64(entry.UserLevel.Totals.TotalCapacity))

// 	userOpsPerSec.With(labels).Set(float64(entry.UserLevel.Current.OpsPerSec))
// 	userReadOpsPerSec.With(labels).Set(float64(entry.UserLevel.Current.ReadOpsPerSec))
// 	userWriteOpsPerSec.With(labels).Set(float64(entry.UserLevel.Current.WriteOpsPerSec))
// 	userBytesReceivedPerSec.With(labels).Set(float64(entry.UserLevel.Current.DataBytesReceivedPerSec))
// 	userBytesSentPerSec.With(labels).Set(float64(entry.UserLevel.Current.DataBytesSentPerSec))
// 	userThroughputBytesPerSec.With(labels).Add(float64(entry.UserLevel.Current.ThroughputBytesPerSec))

// 	// User quota metrics
// 	userQuotaEnabled.With(labels).Set(boolToFloat64(&entry.UserLevel.Quota.Enabled))
// 	if entry.UserLevel.Quota.MaxSize != nil {
// 		userQuotaMaxSize.With(labels).Set(float64(*entry.UserLevel.Quota.MaxSize))
// 	}
// 	if entry.UserLevel.Quota.MaxObjects != nil {
// 		userQuotaMaxObjects.With(labels).Set(float64(*entry.UserLevel.Quota.MaxObjects))
// 	}

// 	for apiCategory, ops := range entry.UserLevel.APIUsagePerUser {
// 		// Export each category's API usage to Prometheus
// 		apiUsagePerUser.With(prometheus.Labels{
// 			"user":           entry.UserLevel.Meta.ID,
// 			"api_category":   apiCategory,
// 			"rgw_cluster_id": cfg.ClusterID,
// 			"node":           cfg.NodeName,
// 			"instance_id":    cfg.InstanceID,
// 		}).Add(float64(ops))
// 	}

// 	for apiCategory, rate := range entry.UserLevel.Current.APIUsagePerSec {
// 		apiUsagePerUserPerSec.With(prometheus.Labels{
// 			"user":           entry.UserLevel.Meta.ID,
// 			"api_category":   apiCategory,
// 			"rgw_cluster_id": cfg.ClusterID,
// 			"node":           cfg.NodeName,
// 			"instance_id":    cfg.InstanceID,
// 		}).Set(rate)
// 	}

// 	apiUsagePerUserTotalPerSec.With(labels).Set(entry.UserLevel.Current.TotalAPIUsagePerSec)
// }

// // Aggregate cluster metrics from both user and bucket levels
// func aggregateClusterMetrics(entry UsageEntry, clusterMetrics *RadosGWClusterMetrics) {

// 	clusterMetrics.OpsTotal += entry.UserLevel.Totals.OpsTotal
// 	clusterMetrics.BytesSent += float64(entry.UserLevel.Totals.BytesSentTotal)
// 	clusterMetrics.BytesReceived += float64(entry.UserLevel.Totals.BytesReceivedTotal)
// 	clusterMetrics.ThroughputBytes += float64(entry.UserLevel.Totals.ThroughputBytesTotal)
// 	clusterMetrics.ReadOpsPerSec += float64(entry.UserLevel.Current.ReadOpsPerSec)
// 	clusterMetrics.WriteOpsPerSec += float64(entry.UserLevel.Current.WriteOpsPerSec)
// 	clusterMetrics.BytesSentPerSec += float64(entry.UserLevel.Current.DataBytesSentPerSec)
// 	clusterMetrics.BytesReceivedPerSec += float64(entry.UserLevel.Current.DataBytesReceivedPerSec)
// 	clusterMetrics.ThroughputBytesPerSec += entry.UserLevel.Current.ThroughputBytesPerSec
// 	clusterMetrics.CapacityUsageBytes += entry.UserLevel.Totals.TotalCapacity

// 	// Error rate is averaged across all users/buckets
// 	if entry.UserLevel.Totals.OpsTotal > 0 {
// 		errorRate := float64(entry.UserLevel.Totals.OpsTotal-entry.UserLevel.Totals.SuccessOpsTotal) / float64(entry.UserLevel.Totals.OpsTotal) * 100
// 		clusterMetrics.ErrorRate += errorRate
// 	}

// 	clusterMetrics.CurrentOpsPerSec += entry.UserLevel.Current.OpsPerSec
// }

// // Populate cluster-level metrics into Prometheus
// func populateClusterMetrics(clusterID, node, instanceID string, clusterMetrics RadosGWClusterMetrics) {
// 	labels := prometheus.Labels{
// 		"rgw_cluster_id": clusterID,
// 		"node":           node,
// 		"instance_id":    instanceID,
// 	}

// 	clusterOpsTotal.With(labels).Set(float64(clusterMetrics.OpsTotal))
// 	clusterBytesSentTotal.With(labels).Set(float64(clusterMetrics.BytesSent))
// 	clusterBytesReceivedTotal.With(labels).Set(float64(clusterMetrics.BytesReceived))
// 	clusterReadsPerSec.With(labels).Set(clusterMetrics.ReadOpsPerSec)
// 	clusterWritesPerSec.With(labels).Set(clusterMetrics.WriteOpsPerSec)
// 	clusterBytesSentPerSec.With(labels).Set(clusterMetrics.BytesSentPerSec)
// 	clusterBytesReceivedPerSec.With(labels).Set(clusterMetrics.BytesReceivedPerSec)
// 	clusterThroughputBytesPerSec.With(labels).Set(clusterMetrics.ThroughputBytesPerSec)
// 	clusterErrorRate.With(labels).Set(clusterMetrics.ErrorRate)
// 	clusterOpsPerSec.With(labels).Set(clusterMetrics.CurrentOpsPerSec)
// 	clusterCapacityUsageBytes.With(labels).Set(float64(clusterMetrics.CapacityUsageBytes))
// }

// func boolToFloat64(b *bool) float64 {
// 	if b != nil && *b {
// 		return 1.0
// 	}
// 	return 0.0
// }

// func startPrometheusMetricsServer(port int) {
// 	http.Handle("/metrics", promhttp.Handler())
// 	http.ListenAndServe(":"+strconv.Itoa(port), nil)
// }
