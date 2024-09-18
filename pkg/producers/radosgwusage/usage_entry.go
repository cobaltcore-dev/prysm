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

import "github.com/ceph/go-ceph/rgw/admin"

// UsageEntry represents a user's usage data and associated buckets.
type UsageEntry struct {
	ClusterID           string          `json:"rgw_custer_id"`         // The RGW cluster ID backend used for the bucket.
	User                string          `json:"user"`                  // The ID of the user.
	DisplayName         string          `json:"display_name"`          // The display name of the user.
	Email               string          `json:"email"`                 // The email address of the user.
	DefaultStorageClass string          `json:"default_storage_class"` // The default storage class for the user.
	UserQuota           admin.QuotaSpec `json:"user_quota"`            // The quota specifications for the user.
	BucketQuota         admin.QuotaSpec `json:"bucket_quota"`          // The quota specifications for the user.
	Stats               admin.UserStat  `json:"stats"`                 // Statistical information about the user's usage.
	Buckets             []BucketUsage   `json:"buckets"`               // A list of buckets associated with the user.
	// TotalThroughputBytes uint64          `json:"total_throughput_bytes"` // The total throughput in bytes (sent + received) for the user.
	TotalLatencySeconds float64 `json:"total_latency_seconds"` // The total latency in seconds for operations performed by the user.
	// CurrentOps           uint64          `json:"current_ops"`            // The current number of operations being performed by the user.
	MaxOps uint64 `json:"max_ops"` // The maximum number of operations performed by the user at any given time.
	// Redesign
	UserLevel RadosGWUserMetrics `json:"user_level"` // Metrics related to the user level.
}

// BucketUsage represents detailed information about a bucket, including usage and quotas.
type BucketUsage struct {
	Bucket               string           `json:"bucket"`                 // The name of the bucket.
	Owner                string           `json:"owner"`                  // The owner of the bucket.
	Zonegroup            string           `json:"zonegroup"`              // The zonegroup in which the bucket is located.
	Usage                UsageStats       `json:"usage"`                  // The usage statistics of the bucket.
	BucketQuota          admin.QuotaSpec  `json:"bucket_quota"`           // The quota specifications for the bucket.
	NumShards            uint64           `json:"num_shards"`             // The number of shards in the bucket.
	Categories           []CategoryUsage  `json:"categories"`             // A list of operation categories within the bucket.
	APIUsagePerBucket    map[string]int64 `json:"api_usage_per_bucket"`   // A map of API usage per bucket.
	TotalOps             uint64           `json:"total_ops"`              // The total number of operations performed in the bucket.
	TotalBytesSent       uint64           `json:"total_bytes_sent"`       // The total number of bytes sent from the bucket.
	TotalBytesReceived   uint64           `json:"total_bytes_received"`   // The total number of bytes received by the bucket.
	TotalThroughputBytes uint64           `json:"total_throughput_bytes"` // The total throughput in bytes (sent + received) for the bucket.
	TotalLatencySeconds  float64          `json:"total_latency_seconds"`  // The total latency in seconds for operations in the bucket.
	TotalRequests        uint64           `json:"total_requests"`         // The total number of requests performed in the bucket.
	CurrentOps           uint64           `json:"current_ops"`            // The current number of operations being performed in the bucket.
	MaxOps               uint64           `json:"max_ops"`                // The maximum number of operations performed in the bucket at any given time.
	TotalReadOps         uint64           `json:"read_ops"`               // Total number of read operations (e.g., GET, LIST) for this bucket
	TotalWriteOps        uint64           `json:"write_ops"`              // Total number of write operations (e.g., PUT, DELETE) for this bucket
	TotalSuccessOps      uint64           `json:"success_ops"`            // Total number of successful operations for this bucket (sum of successful operations across all categories)
	ErrorRate            float64          `json:"error_rate"`             // Error rate for this bucket as a percentage (calculated as (total ops - successful ops) / total ops * 100)
}

// UsageStats represents the usage statistics of a bucket.
type UsageStats struct {
	RgwMain struct {
		Size           *uint64 `json:"size"`             // The total size of objects in the bucket (in bytes).
		SizeActual     *uint64 `json:"size_actual"`      // The actual size of the bucket (in bytes).
		SizeUtilized   *uint64 `json:"size_utilized"`    // The utilized size of the bucket (in bytes).
		SizeKb         *uint64 `json:"size_kb"`          // The size of the bucket in kilobytes.
		SizeKbActual   *uint64 `json:"size_kb_actual"`   // The actual size of the bucket in kilobytes.
		SizeKbUtilized *uint64 `json:"size_kb_utilized"` // The utilized size of the bucket in kilobytes.
		NumObjects     *uint64 `json:"num_objects"`      // The number of objects in the bucket.
	} `json:"rgw.main"`
	RgwMultimeta struct {
		Size           *uint64 `json:"size"`             // The size of multimeta objects in the bucket (in bytes).
		SizeActual     *uint64 `json:"size_actual"`      // The actual size of multimeta objects in the bucket (in bytes).
		SizeUtilized   *uint64 `json:"size_utilized"`    // The utilized size of multimeta objects in the bucket (in bytes).
		SizeKb         *uint64 `json:"size_kb"`          // The size of multimeta objects in the bucket in kilobytes.
		SizeKbActual   *uint64 `json:"size_kb_actual"`   // The actual size of multimeta objects in the bucket in kilobytes.
		SizeKbUtilized *uint64 `json:"size_kb_utilized"` // The utilized size of multimeta objects in the bucket in kilobytes.
		NumObjects     *uint64 `json:"num_objects"`      // The number of multimeta objects in the bucket.
	} `json:"rgw.multimeta"`
}

// CategoryUsage represents a category of operations in usage statistics.
type CategoryUsage struct {
	Category      string `json:"category"`       // The category of operations (e.g., PUT, GET, DELETE).
	BytesSent     uint64 `json:"bytes_sent"`     // The total number of bytes sent for this category.
	BytesReceived uint64 `json:"bytes_received"` // The total number of bytes received for this category.
	Ops           uint64 `json:"ops"`            // The total number of operations performed in this category.
	SuccessfulOps uint64 `json:"successful_ops"` // The total number of successful operations in this category.
}

// UsageMetrics represents aggregated usage metrics for operations.
type UsageMetrics struct {
	Ops           uint64 // The total number of operations.
	SuccessfulOps uint64 // The total number of successful operations.
	BytesSent     uint64 // The total number of bytes sent.
	BytesReceived uint64 // The total number of bytes received.
}

///// Redesign

// RadosGWUserMetrics holds the user-level RADOSGW metrics
type RadosGWUserMetrics struct {
	UserBucketsTotal         int     // Total number of buckets for each user
	UserObjectsTotal         uint64  // Total number of objects for each user
	UserDataSizeTotal        uint64  // Total size of data for each user (in bytes)
	UserOpsTotal             uint64  // Total number of operations for each user
	UserReadOpsTotal         uint64  // Total read operations for each user
	UserWriteOpsTotal        uint64  // Total write operations for each user
	UserBytesSentTotal       uint64  // Total bytes sent by each user
	UserBytesReceivedTotal   uint64  // Total bytes received by each user
	MaxOps                   int64   // Maximum observed operations (reads/writes) for each user
	UserSuccessOpsTotal      uint64  // Total successful operations for each user
	ErrorRateTotal           float64 // Error rate for each user (percentage)
	UserThroughputBytesTotal uint64  // Total throughput in bytes for each user (read and write combined)
	UserTotalCapacity        uint64  // Capacity Usage

	UserOpsCurrent             uint64  // Current number of operations (reads/writes) during the current interval
	UserBytesSentCurrent       uint64  // Bytes sent during the current interval
	UserBytesReceivedCurrent   uint64  // Bytes received during the current interval
	UserThroughputBytesCurrent float64 // Current throughput in bytes (over the current interval)

	// Additional metrics (placeholders for further implementation)
	// API Usage per User, where the key is the API category (e.g., "get_obj", "put_obj") and the value is the count of operations for that category
	APIUsagePerUser        map[string]int64 // API usage breakdown per user by category (e.g., "get_obj": 100, "put_obj": 50)
	Top10CapacityConsumers []string         // Top 10 capacity-consuming users (list of user IDs or names)
}

// NewRadosGWUserMetrics creates and initializes a new instance of RadosGWUserMetrics
func NewRadosGWUserMetrics() *RadosGWUserMetrics {
	return &RadosGWUserMetrics{
		UserBucketsTotal:           0,
		UserObjectsTotal:           0,
		UserDataSizeTotal:          0,
		UserOpsTotal:               0,
		UserReadOpsTotal:           0,
		UserWriteOpsTotal:          0,
		UserBytesSentTotal:         0,
		UserBytesReceivedTotal:     0,
		MaxOps:                     0,
		ErrorRateTotal:             0.0,
		UserSuccessOpsTotal:        0,
		UserThroughputBytesTotal:   0,
		UserOpsCurrent:             0,
		UserBytesSentCurrent:       0,
		UserBytesReceivedCurrent:   0,
		UserThroughputBytesCurrent: 0,
		APIUsagePerUser:            make(map[string]int64),
		Top10CapacityConsumers:     []string{},
	}
}
