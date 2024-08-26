// Copyright (c) 2024 Clyso GmbH
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program. If not, see <https://www.gnu.org/licenses/>.

package radosgwusage

import "github.com/ceph/go-ceph/rgw/admin"

// UsageEntry represents a user's usage data and associated buckets.
type UsageEntry struct {
	User                 string          `json:"user"`                   // The ID of the user.
	DisplayName          string          `json:"display_name"`           // The display name of the user.
	Email                string          `json:"email"`                  // The email address of the user.
	DefaultStorageClass  string          `json:"default_storage_class"`  // The default storage class for the user.
	UserQuota            admin.QuotaSpec `json:"user_quota"`             // The quota specifications for the user.
	Stats                admin.UserStat  `json:"stats"`                  // Statistical information about the user's usage.
	Buckets              []BucketUsage   `json:"buckets"`                // A list of buckets associated with the user.
	Store                string          `json:"store"`                  // The storage backend used.
	TotalBuckets         int             `json:"total_buckets"`          // The total number of buckets for the user.
	TotalObjects         uint64          `json:"total_objects"`          // The total number of objects for the user.
	TotalDataSize        uint64          `json:"total_data_size"`        // The total size of data for the user (in bytes).
	TotalOps             uint64          `json:"total_ops"`              // The total number of operations performed by the user.
	TotalBytesSent       uint64          `json:"total_bytes_sent"`       // The total number of bytes sent by the user.
	TotalBytesReceived   uint64          `json:"total_bytes_received"`   // The total number of bytes received by the user.
	TotalThroughputBytes uint64          `json:"total_throughput_bytes"` // The total throughput in bytes (sent + received) for the user.
	TotalLatencySeconds  float64         `json:"total_latency_seconds"`  // The total latency in seconds for operations performed by the user.
	CurrentOps           uint64          `json:"current_ops"`            // The current number of operations being performed by the user.
	MaxOps               uint64          `json:"max_ops"`                // The maximum number of operations performed by the user at any given time.
}

// BucketUsage represents detailed information about a bucket, including usage and quotas.
type BucketUsage struct {
	Bucket               string          `json:"bucket"`                 // The name of the bucket.
	Owner                string          `json:"owner"`                  // The owner of the bucket.
	Zonegroup            string          `json:"zonegroup"`              // The zonegroup in which the bucket is located.
	Store                string          `json:"store"`                  // The storage backend used for the bucket.
	Usage                UsageStats      `json:"usage"`                  // The usage statistics of the bucket.
	BucketQuota          admin.QuotaSpec `json:"bucket_quota"`           // The quota specifications for the bucket.
	NumShards            uint64          `json:"num_shards"`             // The number of shards in the bucket.
	Categories           []CategoryUsage `json:"categories"`             // A list of operation categories within the bucket.
	TotalOps             uint64          `json:"total_ops"`              // The total number of operations performed in the bucket.
	TotalBytesSent       uint64          `json:"total_bytes_sent"`       // The total number of bytes sent from the bucket.
	TotalBytesReceived   uint64          `json:"total_bytes_received"`   // The total number of bytes received by the bucket.
	TotalThroughputBytes uint64          `json:"total_throughput_bytes"` // The total throughput in bytes (sent + received) for the bucket.
	TotalLatencySeconds  float64         `json:"total_latency_seconds"`  // The total latency in seconds for operations in the bucket.
	TotalRequests        uint64          `json:"total_requests"`         // The total number of requests performed in the bucket.
	CurrentOps           uint64          `json:"current_ops"`            // The current number of operations being performed in the bucket.
	MaxOps               uint64          `json:"max_ops"`                // The maximum number of operations performed in the bucket at any given time.
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
