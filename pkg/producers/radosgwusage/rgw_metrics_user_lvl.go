// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and prysm contributors
//
// SPDX-License-Identifier: Apache-2.0

package radosgwusage

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/cobaltcore-dev/prysm/pkg/producers/radosgwusage/rgwadmin"
	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog/log"
)

type UserLevelMetrics struct {
	User                string
	Tenant              string
	DisplayName         string
	Email               string
	DefaultStorageClass string
	Zonegroup           string
	BucketsTotal        uint64 // Tracks the total number of buckets for each user. Useful for capacity planning and monitoring. | Usage | = count of buckets
	ObjectsTotal        uint64 // Tracks the total number of objects for each user. Important for understanding storage usage. | User | = stats.num_objects
	DataSizeTotal       uint64 // Tracks the total size of data stored by each user. Key metric for tracking data consumption. | User | = stats.size_utilized
	UserQuotaEnabled    bool
	UserQuotaMaxSize    *int64
	UserQuotaMaxObjects *int64
}

func (m *UserLevelMetrics) GetUserIdentification() string {
	if len(m.Tenant) > 0 {
		return fmt.Sprintf("%s$%s", m.User, m.Tenant)
	}
	return m.User
}

func updateUserMetricsInKV(userData, userUsageData, bucketData, userMetrics nats.KeyValue) error {
	log.Debug().Msg("Starting user-level metrics aggregation")
	_ = userUsageData

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
				processUserMetrics(key, userData, userMetrics, bucketKeyMap)
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

func processUserMetrics(key string, userData, userMetrics nats.KeyValue, bucketKeyMap map[string]uint64) {
	entry, err := userData.Get(key)
	if err != nil {
		if errors.Is(err, nats.ErrKeyNotFound) {
			log.Debug().Str("key", key).Err(err).Msg("User data missing in KV")
			return
		}
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
	userID, tenant := NormalizeUserTenant(user.ID, user.Tenant)
	metrics := UserLevelMetrics{
		User:                userID,
		Tenant:              tenant,
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
	userKey := BuildUserTenantKey(userID, tenant)
	metrics.BucketsTotal = bucketKeyMap[userKey]

	// Calculate derived metrics.

	// Set quota information.
	if user.UserQuota.Enabled != nil && *user.UserQuota.Enabled {
		metrics.UserQuotaEnabled = true
		metrics.UserQuotaMaxSize = user.UserQuota.MaxSize
		metrics.UserQuotaMaxObjects = user.UserQuota.MaxObjects
	}

	// Prepare the metrics key.
	metricsKey := userKey

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
