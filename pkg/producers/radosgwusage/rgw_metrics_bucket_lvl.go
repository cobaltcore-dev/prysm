// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and prysm contributors
//
// SPDX-License-Identifier: Apache-2.0

package radosgwusage

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ceph/go-ceph/rgw/admin"
	"github.com/cobaltcore-dev/prysm/pkg/producers/radosgwusage/rgwadmin"
	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog/log"
)

type UserBucketMetrics struct {
	BucketID      string
	Owner         string
	Zonegroup     string
	TotalReadOPs  uint64 // Cumulative read operations for a specific bucket. Critical for understanding bucket usage.
	TotalWriteOPs uint64 // Cumulative write operations. Helps track data being uploaded to a bucket.
	TotalOPs      uint64 // Sum of total reads and writes. Useful to gauge overall activity for a bucket.
	Throughput    uint64 // Derived from total read/write data transferred and time. Important for tracking bandwidth.
	ObjectCount   uint64 // Number of objects in a bucket. Important for understanding the storage object count.
	BucketSize    uint64 // Total size consumed by the bucket, including all objects. Important for capacity tracking.
	CreationTime  string // Knowing when a bucket was created can be useful for tracking lifecycle and access management.
	// ReplicationStatus // If replication is enabled, track the sync status for disaster recovery purposes.
	// LifecycleRules // Track lifecycle policies applied to the bucket (e.g., auto-deletion of old objects).
	BytesSentTotal     uint64            // Total Data Downloaded
	BytesReceivedTotal uint64            // Total Data Uploaded
	NumShards          *uint64           // Shards
	APIUsage           map[string]uint64 // Tracks API operations per category for the bucket
	QuotaEnabled       bool
	QuotaMaxSize       *int64
	QuotaMaxObjects    *int64
}

func updateBucketMetricsInKV(bucketData, userUsageData, bucketMetrics nats.KeyValue) error {
	log.Debug().Msg("Starting bucket-level metrics aggregation")

	// Fetch all keys from bucketData KV
	keys, err := bucketData.Keys()
	if err != nil {
		log.Error().Err(err).Msg("Failed to fetch keys from bucket data")
		return fmt.Errorf("failed to fetch keys from bucket data: %w", err)
	}

	for _, key := range keys {
		// Fetch bucket metadata
		entry, err := bucketData.Get(key)
		if err != nil {
			log.Warn().
				Str("key", key).
				Err(err).
				Msg("Failed to fetch bucket data from KV")
			continue
		}

		var bucket admin.Bucket
		if err := json.Unmarshal(entry.Value(), &bucket); err != nil {
			log.Warn().
				Str("key", key).
				Err(err).
				Msg("Failed to unmarshal bucket data")
			continue
		}

		log.Debug().
			Str("bucket_id", bucket.Bucket).
			Str("owner", bucket.Owner).
			Msg("Processing bucket metrics")

		// Initialize metrics
		metrics := UserBucketMetrics{
			BucketID:           bucket.Bucket,
			Owner:              bucket.Owner,
			CreationTime:       bucket.Mtime,
			Zonegroup:          "",
			TotalReadOPs:       0,
			TotalWriteOPs:      0,
			TotalOPs:           0,
			Throughput:         0,
			ObjectCount:        uint64(0),
			BucketSize:         uint64(0),
			BytesSentTotal:     0,
			BytesReceivedTotal: 0,
		}

		// Aggregate usage data for this bucket
		user, tenant := SplitUserTenant(bucket.Owner)
		bucketUsageKeyPrefix := BuildUserTenantBucketKey(user, tenant, bucket.Bucket)
		usageKeys, err := userUsageData.Keys()
		if err != nil {
			// log.Error().
			// 	Str("bucket_id", bucket.Bucket).
			// 	Err(err).
			// 	Msg("Failed to fetch usage keys from KV")
			continue
		}

		for _, usageKey := range usageKeys {
			if !strings.HasPrefix(usageKey, bucketUsageKeyPrefix) {
				continue
			}

			// Fetch usage data
			usageEntry, err := userUsageData.Get(usageKey)
			if err != nil {
				log.Warn().
					Str("key", usageKey).
					Err(err).
					Msg("Failed to fetch usage data")
				continue
			}

			var usage rgwadmin.Usage
			if err := json.Unmarshal(usageEntry.Value(), &usage); err != nil {
				log.Warn().
					Str("key", usageKey).
					Err(err).
					Msg("Failed to unmarshal usage data")
				continue
			}

			for _, entry := range usage.Entries {
				for _, bucketUsage := range entry.Buckets {
					if bucketUsage.Bucket != bucket.Bucket {
						continue
					}

					for _, category := range bucketUsage.Categories {
						// Aggregate metrics at the bucket level
						metrics.TotalOPs += category.Ops
						metrics.BytesSentTotal += category.BytesSent
						metrics.BytesReceivedTotal += category.BytesReceived

						if isReadCategory(category.Category) {
							metrics.TotalReadOPs += category.Ops
						} else if isWriteCategory(category.Category) {
							metrics.TotalWriteOPs += category.Ops
						}

						// Track API usage per bucket
						if metrics.APIUsage == nil {
							metrics.APIUsage = make(map[string]uint64)
						}
						metrics.APIUsage[category.Category] += category.Ops
					}
				}
			}
		}

		// Populate static metrics from bucket data
		metrics.Zonegroup = bucket.Zonegroup
		metrics.NumShards = bucket.NumShards

		if bucket.Usage.RgwMain.NumObjects != nil {
			metrics.ObjectCount = *bucket.Usage.RgwMain.NumObjects
		}
		if bucket.Usage.RgwMain.SizeActual != nil {
			metrics.BucketSize = *bucket.Usage.RgwMain.SizeActual
		}

		// Calculate throughput
		metrics.Throughput = metrics.BytesSentTotal + metrics.BytesReceivedTotal

		// Set quota information
		metrics.QuotaEnabled = false
		// Check and populate bucket quota if enabled
		if bucket.BucketQuota.Enabled != nil && *bucket.BucketQuota.Enabled {
			metrics.QuotaEnabled = true
			metrics.QuotaMaxSize = bucket.BucketQuota.MaxSize
			metrics.QuotaMaxObjects = bucket.BucketQuota.MaxObjects
		}

		// Prepare the metrics key
		user, tenant = SplitUserTenant(bucket.Owner)
		metricsKey := BuildUserTenantBucketKey(user, tenant, bucket.Bucket)

		// Serialize and store metrics
		metricsData, err := json.Marshal(metrics)
		if err != nil {
			log.Error().
				Str("bucket_id", bucket.Bucket).
				Err(err).
				Msg("Failed to serialize bucket metrics")
			continue
		}

		if _, err := bucketMetrics.Put(metricsKey, metricsData); err != nil {
			log.Error().
				Str("bucket_id", bucket.Bucket).
				Err(err).
				Msg("Failed to store bucket metrics in KV")
		} else {
			log.Debug().
				Str("bucket_id", bucket.Bucket).
				Str("key", metricsKey).
				Msg("Bucket metrics stored in KV successfully")
		}
	}

	log.Info().Msg("Completed bucket metrics aggregation and storage")
	return nil
}

// Count buckets directly from bucketData
func GetBucketCountForUser(userID string, bucketData nats.KeyValue) (uint64, error) {
	bucketCount := uint64(0)

	// Fetch all keys from bucketData KV
	keys, err := bucketData.Keys()
	if err != nil {
		return 0, fmt.Errorf("failed to fetch keys from bucket data: %w", err)
	}

	for _, key := range keys {
		entry, err := bucketData.Get(key)
		if err != nil {
			continue
		}

		var bucket admin.Bucket
		if err := json.Unmarshal(entry.Value(), &bucket); err != nil {
			continue
		}

		if bucket.Owner == userID {
			bucketCount++
		}
	}

	return bucketCount, nil
}
