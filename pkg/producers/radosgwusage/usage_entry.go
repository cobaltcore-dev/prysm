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
	"time"

	"github.com/ceph/go-ceph/rgw/admin"
)

// UsageEntry represents a user's usage data and associated buckets.
type UsageEntry struct {
	ClusterID string                 `json:"rgw_custer_id"` // The RGW cluster ID backend used for the bucket.
	Stats     admin.UserStat         `json:"stats"`         // Statistical information about the user's usage.
	UserLevel RadosGWUserMetrics     `json:"user_level"`    // Metrics related to the user level.
	Buckets   []RadosGWBucketMetrics `json:"bucket_levels"` // Metrics related to the bucket level.
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
	// Static user metadata
	Meta struct {
		ID                  string // User ID
		DisplayName         string // User display name
		Email               string // User email
		DefaultStorageClass string // Default storage class for the user
	}

	// Accumulated totals (Totals)
	Totals struct {
		BucketsTotal         int     // Total number of buckets for each user
		ObjectsTotal         uint64  // Total number of objects for each user
		DataSizeTotal        uint64  // Total size of data for each user (in bytes)
		OpsTotal             uint64  // Total operations (read + write) for each user
		ReadOpsTotal         uint64  // Total read operations for each user
		WriteOpsTotal        uint64  // Total write operations for each user
		BytesSentTotal       uint64  // Total bytes sent by each user
		BytesReceivedTotal   uint64  // Total bytes received by each user
		SuccessOpsTotal      uint64  // Total successful operations for each user
		ErrorRateTotal       float64 // Error rate for each user
		ThroughputBytesTotal float64 // Total throughput for each user in bytes
		TotalCapacity        uint64  // Total capacity usage for each user
	}

	// Current metrics calculated using deltas
	Current struct {
		OpsPerSec               float64            // Current operations per second (delta)
		ReadOpsPerSec           float64            // Current read operations per second (delta)
		WriteOpsPerSec          float64            // Current write operations per second (delta)
		DataBytesReceivedPerSec float64            // Current data received per second (delta)
		DataBytesSentPerSec     float64            // Current data sent per second (delta)
		ThroughputBytesPerSec   float64            // Current throughput in bytes per second (read and write combined)
		APIUsagePerSec          map[string]float64 // Current API usage by category per second (delta)
		TotalAPIUsagePerSec     float64            // Total API usage per second (across all categories)
	}

	// Quota information for the user
	Quota struct {
		Enabled    bool    // Is quota enabled?
		MaxSize    *uint64 // Maximum size allowed for the user (optional, use pointer)
		MaxObjects *uint64 // Maximum number of objects allowed for the user (optional, use pointer)
	}

	// API Usage per User, where the key is the API category (e.g., "get_obj", "put_obj") and the value is the count of operations for that category
	APIUsagePerUser map[string]uint64 // API usage breakdown per user by category (e.g., "get_obj": 100, "put_obj": 50)
}

// NewRadosGWUserMetrics creates and initializes a new instance of RadosGWUserMetrics
func NewRadosGWUserMetrics() *RadosGWUserMetrics {
	return &RadosGWUserMetrics{
		Meta: struct {
			ID                  string
			DisplayName         string
			Email               string
			DefaultStorageClass string
		}{
			ID:                  "",
			DisplayName:         "",
			Email:               "",
			DefaultStorageClass: "",
		},
		Quota: struct {
			Enabled    bool
			MaxSize    *uint64
			MaxObjects *uint64
		}{
			Enabled:    false,
			MaxSize:    nil,
			MaxObjects: nil,
		},
		Totals: struct {
			BucketsTotal         int
			ObjectsTotal         uint64
			DataSizeTotal        uint64
			OpsTotal             uint64
			ReadOpsTotal         uint64
			WriteOpsTotal        uint64
			BytesSentTotal       uint64
			BytesReceivedTotal   uint64
			SuccessOpsTotal      uint64
			ErrorRateTotal       float64
			ThroughputBytesTotal float64
			TotalCapacity        uint64
		}{
			BucketsTotal:         0,
			ObjectsTotal:         0,
			DataSizeTotal:        0,
			OpsTotal:             0,
			ReadOpsTotal:         0,
			WriteOpsTotal:        0,
			BytesSentTotal:       0,
			BytesReceivedTotal:   0,
			SuccessOpsTotal:      0,
			ErrorRateTotal:       0.0,
			ThroughputBytesTotal: 0.0,
			TotalCapacity:        0,
		},

		APIUsagePerUser: make(map[string]uint64),

		Current: struct {
			OpsPerSec               float64
			ReadOpsPerSec           float64
			WriteOpsPerSec          float64
			DataBytesReceivedPerSec float64
			DataBytesSentPerSec     float64
			ThroughputBytesPerSec   float64
			APIUsagePerSec          map[string]float64
			TotalAPIUsagePerSec     float64
		}{
			OpsPerSec:               0.0,
			ReadOpsPerSec:           0.0,
			WriteOpsPerSec:          0.0,
			DataBytesReceivedPerSec: 0.0,
			DataBytesSentPerSec:     0.0,
			ThroughputBytesPerSec:   0.0,
			APIUsagePerSec:          make(map[string]float64),
			TotalAPIUsagePerSec:     0.0,
		},
	}
}

type RadosGWBucketMetrics struct {
	Meta struct {
		Name      string     // Bucket name
		Owner     string     // Bucket owner
		Zonegroup string     // Zonegroup for the bucket
		Shards    *uint64    // Number of shards for the bucket
		CreatedAt *time.Time // Bucket creation time
	}

	Totals struct {
		DataSize      uint64  // Total size of data in the bucket (in bytes)
		UtilizedSize  uint64  // Total utilized size of data in the bucket (in bytes)
		Objects       uint64  // Total number of objects in the bucket
		ReadOps       uint64  // Total read operations
		WriteOps      uint64  // Total write operations
		BytesSent     uint64  // Total bytes sent from the bucket
		BytesReceived uint64  // Total bytes received by the bucket
		SuccessOps    uint64  // Total successful operations
		OpsTotal      uint64  // Total operations (read + write)
		ErrorRate     float64 // Error rate for operations (percentage)
		Capacity      uint64  // Total capacity used by the bucket
	}

	Current struct {
		OpsPerSec             float64            // Current total operations per second (read + write)
		ReadOpsPerSec         float64            // Current read operations per second (delta)
		WriteOpsPerSec        float64            // Current write operations per second (delta)
		BytesSentPerSec       float64            // Current bytes sent per second
		BytesReceivedPerSec   float64            // Current bytes received per second
		ThroughputBytesPerSec float64            // Current throughput in bytes per second (read + write)
		APIUsage              map[string]float64 // Current API usage rate (per category)
		TotalAPIUsagePerSec   float64            // Total API usage per second (across all categories)
	}

	Quota struct {
		Enabled    bool    // Is quota enabled?
		MaxSize    *uint64 // Maximum size allowed for the bucket (in bytes)
		MaxObjects *uint64 // Maximum number of objects allowed for the bucket
	}

	APIUsage map[string]uint64 // API usage per category (e.g., "get_obj": 100, "put_obj": 50)
}

// NewRadosGWBucketMetrics initializes and returns a new RadosGWBucketMetrics struct
func NewRadosGWBucketMetrics() RadosGWBucketMetrics {
	return RadosGWBucketMetrics{
		Meta: struct {
			Name      string
			Owner     string
			Zonegroup string
			Shards    *uint64
			CreatedAt *time.Time
		}{
			Name:      "",
			Owner:     "",
			Zonegroup: "",
			Shards:    nil,
			CreatedAt: nil,
		},
		Quota: struct {
			Enabled    bool
			MaxSize    *uint64
			MaxObjects *uint64
		}{
			Enabled:    false,
			MaxSize:    nil,
			MaxObjects: nil,
		},
		Totals: struct {
			DataSize      uint64
			UtilizedSize  uint64
			Objects       uint64
			ReadOps       uint64
			WriteOps      uint64
			BytesSent     uint64
			BytesReceived uint64
			SuccessOps    uint64
			OpsTotal      uint64
			ErrorRate     float64
			Capacity      uint64
		}{
			DataSize:      0,
			UtilizedSize:  0,
			Objects:       0,
			ReadOps:       0,
			WriteOps:      0,
			BytesSent:     0,
			BytesReceived: 0,
			SuccessOps:    0,
			OpsTotal:      0,
			ErrorRate:     0,
			Capacity:      0,
		},
		Current: struct {
			OpsPerSec             float64
			ReadOpsPerSec         float64
			WriteOpsPerSec        float64
			BytesSentPerSec       float64
			BytesReceivedPerSec   float64
			ThroughputBytesPerSec float64
			APIUsage              map[string]float64
			TotalAPIUsagePerSec   float64
		}{
			OpsPerSec:             0.0,
			ReadOpsPerSec:         0.0,
			WriteOpsPerSec:        0.0,
			BytesSentPerSec:       0.0,
			BytesReceivedPerSec:   0.0,
			ThroughputBytesPerSec: 0.0,
			APIUsage:              make(map[string]float64),
			TotalAPIUsagePerSec:   0.0,
		},
	}
}

type RadosGWClusterMetrics struct {
	OpsTotal              uint64  // Total operations (read + write)
	ReadOpsPerSec         float64 // Total read operations per second
	WriteOpsPerSec        float64 // Total write operations per second
	BytesSentPerSec       float64 // Total bytes sent per second
	BytesReceivedPerSec   float64 // Total bytes received per second
	ThroughputBytesPerSec float64 // Total throughput in bytes per second (read + write)
	ErrorRate             float64 // Total error rate across the cluster
	CurrentOpsPerSec      float64 // Current number of operations per second
	CapacityUsageBytes    uint64  // Total capacity usage across the cluster
}
