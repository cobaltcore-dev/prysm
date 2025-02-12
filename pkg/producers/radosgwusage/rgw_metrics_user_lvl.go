// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and prysm contributors
//
// SPDX-License-Identifier: Apache-2.0

package radosgwusage

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/cobaltcore-dev/prysm/pkg/producers/radosgwusage/rgwadmin"
	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog/log"
)

type UserLevelMetrics struct {
	User                 string
	Tenant               string
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

func (m *UserLevelMetrics) GetUserIdentification() string {
	if len(m.Tenant) > 0 {
		return fmt.Sprintf("%s$%s", m.User, m.Tenant)
	}
	return m.User
}

func updateUserMetricsInKV(userData, userUsageData, bucketData, userMetrics nats.KeyValue) error {
	log.Debug().Msg("Starting user-level metrics aggregation")

	bucketKeyMap := make(map[string]uint64)
	bucketKeys, err := bucketData.Keys()
	if err != nil {
		log.Error().Err(err).Msg("Failed to fetch keys from bucket data")
		return fmt.Errorf("failed to fetch keys from bucket data: %w", err)
	}
	for _, key := range bucketKeys {
		prefix := key[:strings.LastIndex(key, ".")]
		bucketKeyMap[prefix]++ // Count this bucket for its owner.
	}

	usageKeyMap := make(map[string][]string)
	usageKeys, err := userUsageData.Keys()
	if err != nil {
		log.Error().Err(err).Msg("Failed to fetch usage keys from KV")
		return fmt.Errorf("failed to fetch keys from usage data: %w", err)
	}
	for _, key := range usageKeys {
		prefix := key[:strings.LastIndex(key, ".")]
		usageKeyMap[prefix] = append(usageKeyMap[prefix], key)
	}

	userKeys, err := userData.Keys()
	if err != nil {
		log.Error().Err(err).Msg("Failed to fetch keys from user data")
		return fmt.Errorf("failed to fetch keys from user data: %w", err)
	}

	// Create a worker pool to process users concurrently.
	const numWorkers = 10
	userCh := make(chan string, len(userKeys))
	var wg sync.WaitGroup

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for key := range userCh {
				processUserMetrics(key, userData, userUsageData, userMetrics, bucketKeyMap, usageKeyMap)
			}
		}()
	}

	// Feed the channel.
	for _, key := range userKeys {
		userCh <- key
	}
	close(userCh)
	wg.Wait()

	log.Info().Msg("Completed user metrics aggregation and storage")
	return nil
}

func processUserMetrics(key string, userData, userUsageData, userMetrics nats.KeyValue, bucketKeyMap map[string]uint64, usageKeyMap map[string][]string) {
	entry, err := userData.Get(key)
	if err != nil {
		log.Warn().Str("key", key).Err(err).Msg("Failed to fetch user data from KV")
		return
	}

	var user rgwadmin.KVUser
	if err := json.Unmarshal(entry.Value(), &user); err != nil {
		log.Warn().Str("key", key).Err(err).Msg("Failed to unmarshal user data")
		return
	}

	log.Debug().
		Str("user_id", user.GetUserIdentification()).
		Str("display_name", user.DisplayName).
		Msg("Processing user metrics")

	// Initialize metrics.
	userID := user.ID
	if strings.Index(userID, "$") > 0 { // if tenant is part of owner with devider $
		userID = userID[:strings.Index(userID, "$")]
	}
	metrics := UserLevelMetrics{
		User:                userID,
		Tenant:              user.Tenant,
		DisplayName:         user.DisplayName,
		Email:               user.Email,
		DefaultStorageClass: user.DefaultStorageClass,
		// Initialize numeric fields to zero.
	}

	// Process static user metadata.
	if user.Stats.NumObjects != nil {
		metrics.ObjectsTotal = *user.Stats.NumObjects
	}
	if user.Stats.Size != nil {
		metrics.DataSizeTotal = *user.Stats.Size
	}

	// Use the pre-indexed bucket count.
	metrics.BucketsTotal = bucketKeyMap[key]

	// Aggregate usage data.
	userUsageKeyPrefix := BuildUserTenantKey(user.ID, user.Tenant)

	for _, usageKey := range usageKeyMap[userUsageKeyPrefix] {
		usageEntry, err := userUsageData.Get(usageKey)
		if err != nil {
			log.Warn().Str("key", usageKey).Err(err).Msg("Failed to fetch usage data")
			continue
		}
		var usage rgwadmin.UsageEntryBucket
		if err := json.Unmarshal(usageEntry.Value(), &usage); err != nil {
			log.Warn().Str("key", usageKey).Err(err).Msg("Failed to unmarshal usage data")
			continue
		}

		for _, cat := range usage.Categories {
			metrics.TotalOPs += cat.Ops
			metrics.UserSuccessOpsTotal += cat.SuccessfulOps
			metrics.BytesSentTotal += cat.BytesSent
			metrics.BytesReceivedTotal += cat.BytesReceived
			if isReadCategory(cat.Category) {
				metrics.TotalReadOPs += cat.Ops
			} else if isWriteCategory(cat.Category) {
				metrics.TotalWriteOPs += cat.Ops
			}
			if metrics.APIUsage == nil {
				metrics.APIUsage = make(map[string]uint64)
			}
			metrics.APIUsage[cat.Category] += cat.Ops
		}
	}

	// Calculate derived metrics.
	metrics.ThroughputBytesTotal = metrics.BytesSentTotal + metrics.BytesReceivedTotal
	if metrics.TotalOPs > 0 {
		errorOps := metrics.TotalOPs - metrics.UserSuccessOpsTotal
		metrics.ErrorRatePerUser = (float64(errorOps) / float64(metrics.TotalOPs)) * 100
	}

	// Set quota information.
	if user.UserQuota.Enabled != nil && *user.UserQuota.Enabled {
		metrics.UserQuotaEnabled = true
		metrics.UserQuotaMaxSize = user.UserQuota.MaxSize
		metrics.UserQuotaMaxObjects = user.UserQuota.MaxObjects
	}

	// Prepare the metrics key.
	metricsKey := BuildUserTenantKey(user.ID, user.Tenant)

	// Serialize and store metrics.
	metricsData, err := json.Marshal(metrics)
	if err != nil {
		log.Error().Err(err).Str("user_id", user.GetUserIdentification()).Msg("Failed to serialize user metrics")
		return
	}
	if _, err := userMetrics.Put(metricsKey, metricsData); err != nil {
		log.Error().Err(err).Str("user_id", user.GetUserIdentification()).Msg("Failed to store user metrics in KV")
	} else {
		log.Debug().Str("user_id", user.GetUserIdentification()).Str("key", metricsKey).Msg("User metrics stored in KV successfully")
	}
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
