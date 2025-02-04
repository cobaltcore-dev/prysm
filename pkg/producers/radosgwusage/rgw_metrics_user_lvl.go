// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and prysm contributors
//
// SPDX-License-Identifier: Apache-2.0

package radosgwusage

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ceph/go-ceph/rgw/admin"
	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog/log"
)

type UserLevelMetrics struct {
	UserID               string
	DisplayName          string
	Email                string
	DefaultStorageClass  string
	Zonegroup            string
	BucketsTotal         uint64            // Tracks the total number of buckets for each user. Useful for capacity planning and monitoring. | Usage | = count of buckets
	ObjectsTotal         uint64            // Tracks the total number of objects for each user. Important for understanding storage usage. | User | = stats.num_objects
	DataSizeTotal        uint64            // Tracks the total size of data stored by each user. Key metric for tracking data consumption. | User | = stats.size_utilized
	TotalOPs             uint64            // Sum of total read and write operations per client. Important for tracking overall client activity. | Usage | = sum of all summary.categories.ops values
	TotalReadOPs         uint64            // Tracks the total read operations per user. Useful for understanding individual client read activity. | Usage | = sum of summary.categories.ops where category is one of the read categories
	TotalWriteOPs        uint64            // Tracks the total write operations per user. Useful for tracking client uploads. | Usage | = sum of summary.categories.ops where category is one of the write categories
	BytesSentTotal       uint64            // Tracks the total bytes sent by each user. Useful for monitoring bandwidth usage by client. | Usage | = sum of summary.categories.bytes_sent
	BytesReceivedTotal   uint64            // Tracks the total bytes received by each user. Complements bytes sent for monitoring total data transfer. | Usage | = sum of summary.categories.bytes_received
	ErrorRatePerUser     float64           // Tracking the number of failed requests for each client could help troubleshoot and improve client experience or identify problematic behaviors. | Usage | = (sum of summary.categories.ops - sum of summary.categories.successful_ops) / sum of summary.categories.ops
	UserSuccessOpsTotal  uint64            // Tracks the total successful operations performed by each user. Useful for reliability tracking. | Usage | = sum of summary.categories.successful_ops
	ThroughputBytesTotal uint64            // Derived from total data read and written per client over time. Key for bandwidth analysis per client. | Usage | = (sum of summary.categories.bytes_sent + sum of summary.categories.bytes_received)
	APIUsage             map[string]uint64 // Tracks API operations per category (e.g., "get_obj", "put_obj")
	UserQuotaEnabled     bool
	UserQuotaMaxSize     *int64
	UserQuotaMaxObjects  *int64
}

func updateUserMetricsInKV(userData, userUsageData, bucketData, userMetrics nats.KeyValue) error {
	log.Debug().Msg("Starting user-level metrics aggregation")

	// Fetch all keys from userData KV
	keys, err := userData.Keys()
	if err != nil {
		log.Error().Err(err).Msg("Failed to fetch keys from user data")
		return fmt.Errorf("failed to fetch keys from user data: %w", err)
	}

	// Prepare user metrics aggregation
	for _, key := range keys {
		if !strings.HasPrefix(key, "user_") {
			log.Debug().
				Str("key", key).
				Msg("Skipping non-user key")
			continue
		}

		// Fetch user metadata
		entry, err := userData.Get(key)
		if err != nil {
			log.Warn().
				Str("key", key).
				Err(err).
				Msg("Failed to fetch user data from KV")
			continue
		}

		var user KVUser
		if err := json.Unmarshal(entry.Value(), &user); err != nil {
			log.Warn().
				Str("key", key).
				Err(err).
				Msg("Failed to unmarshal user data")
			continue
		}

		log.Debug().
			Str("user_id", user.GetUserIdentification()).
			Str("display_name", user.DisplayName).
			Msg("Processing user metrics")

		// Initialize metrics
		metrics := UserLevelMetrics{
			UserID:               user.GetUserIdentification(),
			DisplayName:          user.DisplayName,
			Email:                user.Email,
			DefaultStorageClass:  user.DefaultStorageClass,
			BucketsTotal:         0,   // To be calculated
			ObjectsTotal:         0,   // To be aggregated from user stats
			DataSizeTotal:        0,   // To be aggregated from user stats
			TotalOPs:             0,   // To be aggregated from usage
			TotalReadOPs:         0,   // To be aggregated from usage
			TotalWriteOPs:        0,   // To be aggregated from usage
			BytesSentTotal:       0,   // To be aggregated from usage
			BytesReceivedTotal:   0,   // To be aggregated from usage
			ErrorRatePerUser:     0.0, // To be calculated from usage
			UserSuccessOpsTotal:  0,   // To be aggregated from usage
			ThroughputBytesTotal: 0,   // To be calculated from usage
		}

		// Process static user metadata
		if user.Stats.NumObjects != nil {
			metrics.ObjectsTotal = *user.Stats.NumObjects
		}
		if user.Stats.Size != nil {
			metrics.DataSizeTotal = *user.Stats.Size
		}

		// Calculate bucket count from bucketData
		bucketKeys, _ := bucketData.Keys()
		for _, bucketKey := range bucketKeys {
			bucketEntry, err := bucketData.Get(bucketKey)
			if err != nil {
				log.Warn().
					Str("bucket_key", bucketKey).
					Err(err).
					Msg("Failed to fetch bucket data")
				continue
			}

			var bucket admin.Bucket
			if err := json.Unmarshal(bucketEntry.Value(), &bucket); err != nil {
				log.Warn().
					Str("bucket_key", bucketKey).
					Err(err).
					Msg("Failed to unmarshal bucket data")
				continue
			}

			if bucket.Owner == user.GetUserIdentification() {
				metrics.BucketsTotal++
			}
		}

		// Aggregate usage data for this user
		userUsageKeyPrefix := fmt.Sprintf("usage_%s_", user.GetKVFriendlyUserIdentification())
		usageKeys, err := userUsageData.Keys()
		if err != nil {
			log.Error().
				Str("user_id", user.GetUserIdentification()).
				Err(err).
				Msg("Failed to fetch usage keys from KV")
			continue
		}

		for _, usageKey := range usageKeys {
			if !strings.HasPrefix(usageKey, userUsageKeyPrefix) {
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

			var usage KVUserUsage
			if err := json.Unmarshal(usageEntry.Value(), &usage); err != nil {
				log.Warn().
					Str("key", usageKey).
					Err(err).
					Msg("Failed to unmarshal usage data")
				continue
			}

			for _, usageEntry := range usage.Usage.Entries {
				for _, bucket := range usageEntry.Buckets {
					for _, category := range bucket.Categories {
						// Aggregate metrics at the user level
						metrics.TotalOPs += category.Ops
						metrics.UserSuccessOpsTotal += category.SuccessfulOps
						metrics.BytesSentTotal += category.BytesSent
						metrics.BytesReceivedTotal += category.BytesReceived

						if isReadCategory(category.Category) {
							metrics.TotalReadOPs += category.Ops
						} else if isWriteCategory(category.Category) {
							metrics.TotalWriteOPs += category.Ops
						}

						// Track API usage per user
						if metrics.APIUsage == nil {
							metrics.APIUsage = make(map[string]uint64)
						}
						metrics.APIUsage[category.Category] += category.Ops
					}
				}
			}
		}

		// Calculate derived metrics
		metrics.ThroughputBytesTotal = metrics.BytesSentTotal + metrics.BytesReceivedTotal
		if metrics.TotalOPs > 0 {
			errorOps := metrics.TotalOPs - metrics.UserSuccessOpsTotal
			metrics.ErrorRatePerUser = (float64(errorOps) / float64(metrics.TotalOPs)) * 100
		}
		// Set quota information
		metrics.UserQuotaEnabled = false
		// Check and populate user quota if enabled
		if user.UserQuota.Enabled != nil && *user.UserQuota.Enabled {
			metrics.UserQuotaEnabled = true
			metrics.UserQuotaMaxSize = user.UserQuota.MaxSize
			metrics.UserQuotaMaxObjects = user.UserQuota.MaxObjects
		}

		// Prepare the metrics key
		metricsKey := fmt.Sprintf("user_metrics_%s", user.ID)

		// Serialize and store metrics
		metricsData, err := json.Marshal(metrics)
		if err != nil {
			log.Error().
				Str("user_id", user.GetUserIdentification()).
				Err(err).
				Msg("Failed to serialize user metrics")
			continue
		}

		if _, err := userMetrics.Put(metricsKey, metricsData); err != nil {
			log.Error().
				Str("user_id", user.GetUserIdentification()).
				Err(err).
				Msg("Failed to store user metrics in KV")
		} else {
			log.Debug().
				Str("user_id", user.GetUserIdentification()).
				Str("key", metricsKey).
				Msg("User metrics stored in KV successfully")
		}
	}

	log.Info().Msg("Completed user metrics aggregation and storage")
	return nil
}

func isReadCategory(category string) bool {
	readCategories := []string{
		"get_obj", "list_bucket", "get_bucket_policy",
	}
	return contains(readCategories, category)
}

func isWriteCategory(category string) bool {
	writeCategories := []string{
		"put_obj", "delete_obj", "create_bucket",
	}
	return contains(writeCategories, category)
}
