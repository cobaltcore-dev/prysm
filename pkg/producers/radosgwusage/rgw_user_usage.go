// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and prysm contributors
//
// SPDX-License-Identifier: Apache-2.0

package radosgwusage

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/ceph/go-ceph/rgw/admin"
	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog/log"
)

const (
	rootBucketPlaceholder        = "root"
	nonBucketSpecificPlaceholder = "-"
)

type KVUserUsage struct {
	ID          string        `json:"id"`
	LastUpdated time.Time     `json:"lastUpdated"`
	Usage       UserUsageSpec `json:"usage"`
}

// @see admin.Usage
type UserUsageSpec struct {
	Entries []UserUsageEntry   `json:"entries"`
	Summary []UserUsageSummary `json:"summary"`
}

type UserUsageEntry struct {
	User    string            `json:"user"`
	Buckets []UserUsageBucket `json:"buckets"`
}

type UserUsageBucket struct {
	Bucket     string                    `json:"bucket"`
	Time       string                    `json:"time"`
	Epoch      uint64                    `json:"epoch"`
	Owner      string                    `json:"owner"`
	Categories []UserUsageBucketCategory `json:"categories"`
}

type UserUsageBucketCategory struct {
	Category      string `json:"category"`
	BytesSent     uint64 `json:"bytes_sent"`
	BytesReceived uint64 `json:"bytes_received"`
	Ops           uint64 `json:"ops"`
	SuccessfulOps uint64 `json:"successful_ops"`
}

type UserUsageSummary struct {
	User       string                    `json:"user"`
	Categories []UserUsageBucketCategory `json:"categories"`
	Total      UserUsageSummaryTotal     `json:"total"`
}

type UserUsageSummaryTotal struct {
	BytesSent     uint64 `json:"bytes_sent"`
	BytesReceived uint64 `json:"bytes_received"`
	Ops           uint64 `json:"ops"`
	SuccessfulOps uint64 `json:"successful_ops"`
}

func refreshUserUsagesInKV(syncControl nats.KeyValue, userUsageData nats.KeyValue, co *admin.API, prefix string) error {

	// Define keys for tracking sync
	globalSyncKey := "sync_usages"
	globalSyncInProgressKey := "sync_usages_in_progress"

	// hack, perform allways globals sync
	if _, putErr := syncControl.Put(globalSyncKey, []byte("true")); putErr != nil {
		return fmt.Errorf("failed to initialize %s key: %w", globalSyncKey, putErr)
	}

	// Check if a global sync is already in progress
	_, err := syncControl.Get(globalSyncInProgressKey)
	if err == nil {
		log.Debug().Msg("Global user usage sync already in progress; skipping")
		return nil
	}

	// Set the global sync in-progress flag
	if _, err := syncControl.Put(globalSyncInProgressKey, []byte("true")); err != nil {
		return fmt.Errorf("failed to set global sync in-progress flag: %w", err)
	}
	defer func() {
		if err := syncControl.Delete(globalSyncInProgressKey); err != nil {
			log.Warn().Err(err).Msg("Failed to delete global sync in-progress flag")
		}
	}()

	// Check if global usage sync is required
	globalSync := false
	value, err := syncControl.Get(globalSyncKey)
	if err != nil {
		log.Warn().Msgf("Key %s not found; initializing with true", globalSyncKey)
		if _, putErr := syncControl.Put(globalSyncKey, []byte("true")); putErr != nil {
			return fmt.Errorf("failed to initialize %s key: %w", globalSyncKey, putErr)
		}
		globalSync = true
	} else {
		globalSync = value != nil && string(value.Value()) == "true"
	}

	// Fetch granular user sync flags
	keys, err := syncControl.Keys()
	if err != nil {
		log.Warn().Msg("sync_control bucket is empty; initializing")
		keys = []string{}
	}

	granularUserIDs := []string{}
	for _, key := range keys {
		if strings.HasPrefix(key, "sync_usage_user_") {
			userID := strings.TrimPrefix(key, "sync_usage_user_")
			granularUserIDs = append(granularUserIDs, userID)
		}
	}

	usersFailed := 0

	// Perform global or incremental sync for all users
	if globalSync || len(granularUserIDs) == 0 {
		log.Debug().Msg("Performing global or incremental user usage sync")

		// Fetch all user usage data
		usage, err := fetchUserUsage(co, "", syncControl)
		if err != nil {
			return fmt.Errorf("failed to fetch global user usage: %w", err)
		}

		//Always update KV entries for all users
		if err := storeUserUsageInKV(usage, userUsageData); err != nil {
			usersFailed++
		}

		// Reset the global sync flag
		if globalSync {
			if _, err := syncControl.Put(globalSyncKey, []byte("false")); err != nil {
				log.Warn().Err(err).Msg("Failed to reset global sync_usages flag")
			}
		}
	}

	// Perform granular sync for specific users
	for _, userID := range granularUserIDs {
		usage, err := fetchUserUsage(co, userID, syncControl)
		if err != nil {
			log.Warn().
				Str("user", userID).
				Err(err).
				Msg("Failed to fetch user usage for granular sync")
			usersFailed++
			continue
		}

		// Always update KV entries for granular sync users
		if err := storeUserUsageInKV(usage, userUsageData); err != nil {
			usersFailed++
			continue
		}

		// Reset the granular sync flag
		userKey := fmt.Sprintf("%s_sync_usage_user_%s", prefix, userID)
		if err := syncControl.Delete(userKey); err != nil {
			log.Warn().
				Str("user", userID).
				Err(err).
				Msg("Failed to reset granular sync flag")
		}
	}

	log.Debug().
		Int("usersFailed", usersFailed).
		Msg("Completed user usage sync in KV")

	return nil
}

func checkAndRefreshUserUsages(syncControl, userUsageData, userData nats.KeyValue, cfg RadosGWUsageConfig, status *PrysmStatus) {
	// Validate the configuration to ensure necessary fields are set
	if cfg.AdminURL == "" || cfg.AccessKey == "" || cfg.SecretKey == "" {
		log.Fatal().Msg("invalid configuration: AdminURL, AccessKey, and SecretKey must be provided")
	}

	// Timers for incremental and full syncs
	fullSyncTicker := time.NewTicker(1 * time.Minute) // Trigger full sync every 10 minutes
	// incrementalSyncTicker := time.NewTicker(30 * time.Second) // Incremental sync every 30 seconds
	defer fullSyncTicker.Stop()
	// defer incrementalSyncTicker.Stop()

	for {
		select {
		case <-fullSyncTicker.C:
			if isFlagSet(syncControl, "metric_calc_in_progress") {
				log.Debug().Msg("Skipping full user sync; metric calculation in progress")
				continue
			}
			if isFlagSet(syncControl, "sync_users_in_progress") {
				log.Debug().Msg("Skipping full user sync; sync users in progress")
				continue
			}

			// Create a new RadosGW admin client
			co, err := createRadosGWClient(cfg, status)
			if err != nil {
				log.Error().Err(err).Msg("Failed to create RadosGW admin client")
				continue
			}

			// Trigger global user sync
			log.Debug().Msg("Triggering periodic full user usage sync")
			if _, err := syncControl.Put("sync_usages", []byte("true")); err != nil {
				log.Error().Err(err).Msg("Failed to trigger global user usage sync")
			}
			// err := refreshUserUsagesInKV(syncControl, userUsageData, co, cfg.SyncControlBucketPrefix)
			err = refreshUserUsagesInKV2(syncControl, userUsageData, userData, co, cfg.SyncControlBucketPrefix)
			if err != nil {
				status.IncrementScrapeErrors()
				log.Error().Err(err).Msg("error refreshing users usage in KV")
			} else {
				status.UpdateTargetUp(true)
			}

			// case <-incrementalSyncTicker.C:
			// 	if isFlagSet(syncControl, "metric_calc_in_progress") {
			// 		log.Debug().Msg("Skipping incremental user sync; metric calculation in progress")
			// 		continue
			// 	}
			// 	if isFlagSet(syncControl, "sync_users_in_progress") {
			// 		log.Debug().Msg("Skipping full user sync; sync users in progress")
			// 		continue
			// 	}

			// 	// Perform incremental sync
			// 	log.Debug().Msg("Starting incremental user usage sync")
			// 	err := refreshUserUsagesInKV(syncControl, userUsageData, co, cfg.SyncControlBucketPrefix)
			// 	if err != nil {
			// 		log.Error().Err(err).Msg("error refreshing users usage in KV")
			// 	}
		}
	}
}

func triggerUserUsageSync(syncControl nats.KeyValue, userID string) error {
	userUsageSyncKey := fmt.Sprintf("sync_usage_user_%s", userID)

	// Set the sync flag to true
	if _, err := syncControl.Put(userUsageSyncKey, []byte("true")); err != nil {
		return fmt.Errorf("failed to set sync usage flag for user %s: %w", userID, err)
	}

	log.Info().Str("user", userID).Msg("Triggered usage sync for user")
	return nil
}

func fetchUserUsage(co *admin.API, userID string, syncControl nats.KeyValue) (admin.Usage, error) {
	var aggregatedUsage admin.Usage
	var startTime string

	if userID == "" {
		// Global sync: Fetch the global sync timestamp
		lastGlobalSyncKey := "last_sync_usage_global"
		value, err := syncControl.Get(lastGlobalSyncKey)
		if err == nil {
			startTime = string(value.Value())
		} else {
			log.Warn().
				Msg("No global last sync time found; fetching full usage data")
		}
	} else {
		// Granular sync: Fetch the individual user sync timestamp
		lastUserSyncKey := fmt.Sprintf("last_sync_usage_user_%s", userID)
		value, err := syncControl.Get(lastUserSyncKey)
		if err == nil {
			startTime = string(value.Value())
		} else {
			log.Warn().
				Str("user", userID).
				Msg("No last sync time found for user; fetching full usage data")
		}
	}

	// Prepare the usage request parameters
	usageRequest := admin.Usage{
		UserID: userID,
		Start:  startTime, // Incremental fetch if startTime is set
	}
	showEntries := true
	usageRequest.ShowEntries = &showEntries

	// Fetch the usage data from the API
	globalUsage, err := co.GetUsage(context.Background(), usageRequest)
	if err != nil {
		return admin.Usage{}, fmt.Errorf("failed to fetch usage data for user %s: %w", userID, err)
	}

	// Aggregate global entries
	aggregatedUsage.Entries = append(aggregatedUsage.Entries, globalUsage.Entries...)
	aggregatedUsage.Summary = append(aggregatedUsage.Summary, globalUsage.Summary...)

	// For global sync, fetch detailed usage for each user
	if userID == "" {
		for idx, entry := range globalUsage.Entries {
			detailedUsageRequest := admin.Usage{
				UserID:      entry.User,
				ShowEntries: &showEntries,
			}

			detailedUsage, err := co.GetUsage(context.Background(), detailedUsageRequest)
			if err != nil {
				log.Warn().
					Str("user", entry.User).
					Err(err).
					Msg("Failed to fetch detailed usage for user")
				continue
			}

			// Replace global entry with detailed entry
			if len(detailedUsage.Entries) > 0 {
				aggregatedUsage.Entries[idx] = detailedUsage.Entries[0]
			}
		}
	}

	updateSyncTimestamps(aggregatedUsage, userID, syncControl)

	return aggregatedUsage, nil
}

func updateSyncTimestamps(usage admin.Usage, userID string, syncControl nats.KeyValue) {
	lastSyncTime := time.Now().Format("2006-01-02 15:04:05")

	if userID == "" {
		// Update global sync timestamp
		if _, err := syncControl.Put("last_sync_usage_global", []byte(lastSyncTime)); err != nil {
			log.Error().
				Err(err).
				Msg("Failed to update global last sync time in KV")
		}

		// Update individual user sync timestamps
		for _, entry := range usage.Entries {
			userKey := fmt.Sprintf("last_sync_usage_user_%s", entry.User)
			if _, err := syncControl.Put(userKey, []byte(lastSyncTime)); err != nil {
				log.Warn().
					Str("user", entry.User).
					Err(err).
					Msg("Failed to update last sync time for user")
			}
		}
	} else {
		// Update individual user sync timestamp
		lastUserSyncKey := fmt.Sprintf("last_sync_usage_user_%s", userID)
		if _, err := syncControl.Put(lastUserSyncKey, []byte(lastSyncTime)); err != nil {
			log.Error().
				Str("user", userID).
				Err(err).
				Msg("Failed to update last sync time for user")
		}
	}
}

func fetchAllUserUsage(co *admin.API, syncControl nats.KeyValue) ([]admin.Usage, error) {
	usage, err := fetchUserUsage(co, "", syncControl)
	if err != nil {
		return nil, err
	}

	log.Info().Int("entries", len(usage.Entries)).Msg("Fetched global user usage data")
	return []admin.Usage{usage}, nil
}

func performFullUserUsageSyncFromKV(userData, syncControl, userUsageData nats.KeyValue, co *admin.API) error {
	log.Info().Msg("Starting full user usage sync using KV user data")

	// Fetch all user keys from the KV store
	keys, err := userData.Keys()
	if err != nil {
		return fmt.Errorf("failed to fetch keys from userData: %w", err)
	}

	usersFailed := 0
	showEntries := true

	// Iterate over user keys to fetch and store detailed usage
	for _, key := range keys {
		if !strings.HasPrefix(key, "user_") {
			log.Debug().Str("key", key).Msg("Skipping non-user key")
			continue
		}

		// Fetch user details from KV
		entry, err := userData.Get(key)
		if err != nil {
			log.Warn().Str("key", key).Err(err).Msg("Failed to fetch user data")
			usersFailed++
			continue
		}

		var user KVUser
		if err := json.Unmarshal(entry.Value(), &user); err != nil {
			log.Warn().Str("key", key).Err(err).Msg("Failed to unmarshal user data")
			usersFailed++
			continue
		}

		// Prepare and fetch detailed usage for the user
		detailedUsageRequest := admin.Usage{
			UserID:      user.GetUserIdentification(),
			ShowEntries: &showEntries,
		}

		detailedUsage, err := co.GetUsage(context.Background(), detailedUsageRequest)
		if err != nil {
			log.Warn().
				Str("user", user.GetUserIdentification()).
				Err(err).
				Msg("Failed to fetch detailed usage for user")
			usersFailed++
			continue
		}

		// Store the detailed usage in KV
		if err := storeUserUsageInKV(detailedUsage, userUsageData); err != nil {
			log.Warn().
				Str("user", user.GetUserIdentification()).
				Err(err).
				Msg("Failed to store user usage in KV")
			usersFailed++
			continue
		}
	}

	log.Info().
		Int("totalUsers", len(keys)).
		Int("usersFailed", usersFailed).
		Msg("Completed full user usage sync using KV")

	return nil
}

func fetchSingleUserUsage(co *admin.API, userID string, syncControl nats.KeyValue) (admin.Usage, error) {
	usage, err := fetchUserUsage(co, userID, syncControl)
	if err != nil {
		return admin.Usage{}, err
	}

	log.Info().
		Str("user", userID).
		Int("entries", len(usage.Entries)).
		Msg("Fetched user usage data")
	return usage, nil
}

func refreshUserUsagesInKV2(syncControl, userUsageData, userData nats.KeyValue, co *admin.API, prefix string) error {
	globalSyncKey := "sync_usages"
	globalSyncInProgressKey := "sync_usages_in_progress"

	// Check if a global sync is already in progress
	if isFlagSet(syncControl, globalSyncInProgressKey) {
		log.Debug().Msg("Global user usage sync already in progress; skipping")
		return nil
	}

	// Set the global sync in-progress flag
	if _, err := syncControl.Put(globalSyncInProgressKey, []byte("true")); err != nil {
		return fmt.Errorf("failed to set global sync in-progress flag: %w", err)
	}
	defer func() {
		if err := syncControl.Delete(globalSyncInProgressKey); err != nil {
			log.Warn().Err(err).Msg("Failed to delete global sync in-progress flag")
		}
	}()

	// Check if a global sync is required
	globalSync := isFlagSet(syncControl, globalSyncKey)
	if globalSync {
		log.Debug().Msg("Performing full user usage sync using KV")
		if err := performFullUserUsageSyncFromKV(userData, syncControl, userUsageData, co); err != nil {
			log.Error().Err(err).Msg("Error during full user usage sync using KV")
		}

		// Reset the global sync flag
		if _, err := syncControl.Put(globalSyncKey, []byte("false")); err != nil {
			log.Warn().Err(err).Msg("Failed to reset global sync flag")
		}
	}
	return nil
}

func storeUserUsageInKV(userUsage admin.Usage, userUsageData nats.KeyValue) error {
	bucketsFailed := 0
	skippedBuckets := 0

	// Iterate through each user entry and store usage for each bucket
	for _, entry := range userUsage.Entries {
		for _, bucket := range entry.Buckets {
			bucketName := bucket.Bucket
			if bucketName == "" {
				bucketName = rootBucketPlaceholder // Use a placeholder for root or unnamed buckets
			}
			if bucketName == nonBucketSpecificPlaceholder {
				skippedBuckets++
				log.Debug().
					Str("user", entry.User).
					Msg("Skipping non-bucket-specific usage ('-')")
				continue
			}

			// Create the KV structure for the bucket usage
			kvBucketUsage := KVUserUsage{
				ID:          fmt.Sprintf("%s_%s", entry.User, bucketName),
				LastUpdated: time.Now(),
				Usage: UserUsageSpec{
					Entries: []UserUsageEntry{
						{
							User: entry.User,
							Buckets: []UserUsageBucket{
								{
									Bucket:     bucket.Bucket,
									Time:       bucket.Time,
									Epoch:      bucket.Epoch,
									Owner:      bucket.Owner,
									Categories: convertCategories(bucket.Categories),
								},
							},
						},
					},
				},
			}

			// Serialize the bucket usage data
			bucketDataJSON, err := json.Marshal(kvBucketUsage)
			if err != nil {
				log.Error().
					Str("user", entry.User).
					Str("bucket", bucket.Bucket).
					Err(err).
					Msg("Error serializing bucket usage data")
				bucketsFailed++
				continue
			}

			bucketKey := fmt.Sprintf("usage_%s_%s", entry.User, bucketName)
			bucketKey = strings.ReplaceAll(bucketKey, "$", "_tenant_")

			// Write the bucket usage data to the KV store
			if _, err := userUsageData.Put(bucketKey, bucketDataJSON); err != nil {
				log.Warn().
					Str("user", entry.User).
					Str("bucket", bucket.Bucket).
					Err(err).
					Msg("Failed to update KV for bucket usage")
				bucketsFailed++
				continue
			}
		}
	}

	log.Debug().
		Int("bucketsFailed", bucketsFailed).
		Int("skippedBuckets", skippedBuckets).
		Msg("Completed storing bucket usage in KV")

	return nil
}

func convertBuckets(buckets []struct {
	Bucket     string `json:"bucket"`
	Time       string `json:"time"`
	Epoch      uint64 `json:"epoch"`
	Owner      string `json:"owner"`
	Categories []struct {
		Category      string `json:"category"`
		BytesSent     uint64 `json:"bytes_sent"`
		BytesReceived uint64 `json:"bytes_received"`
		Ops           uint64 `json:"ops"`
		SuccessfulOps uint64 `json:"successful_ops"`
	} `json:"categories"`
}) []UserUsageBucket {
	var result []UserUsageBucket
	for _, bucket := range buckets {
		result = append(result, UserUsageBucket{
			Bucket:     bucket.Bucket,
			Time:       bucket.Time,
			Epoch:      bucket.Epoch,
			Owner:      bucket.Owner,
			Categories: convertCategories(bucket.Categories),
		})
	}
	return result
}

func convertCategories(categories []struct {
	Category      string `json:"category"`
	BytesSent     uint64 `json:"bytes_sent"`
	BytesReceived uint64 `json:"bytes_received"`
	Ops           uint64 `json:"ops"`
	SuccessfulOps uint64 `json:"successful_ops"`
}) []UserUsageBucketCategory {
	var result []UserUsageBucketCategory
	for _, category := range categories {
		result = append(result, UserUsageBucketCategory{
			Category:      category.Category,
			BytesSent:     category.BytesSent,
			BytesReceived: category.BytesReceived,
			Ops:           category.Ops,
			SuccessfulOps: category.SuccessfulOps,
		})
	}
	return result
}
