// Copyright 2025 Clyso GmbH
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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

// @see admin.User
type KVUser struct {
	ID                  string              `json:"id"`
	DisplayName         string              `json:"displayName"`
	Email               string              `json:"email"`
	Suspended           *int                `json:"suspended"`
	MaxBuckets          *int                `json:"maxBuckets"`
	Caps                []admin.UserCapSpec `json:"caps"`
	OpMask              string              `json:"op_mask"`
	DefaultPlacement    string              `json:"default_placement"`
	DefaultStorageClass string              `json:"default_storage_class"`
	PlacementTags       []interface{}       `json:"placement_tags"`
	BucketQuota         admin.QuotaSpec     `json:"bucket_quota"`
	UserQuota           admin.QuotaSpec     `json:"user_quota"`
	TempURLKeys         []interface{}       `json:"temp_url_keys"`
	Type                string              `json:"type"`
	Tenant              string              `json:"tenant"`
	Stats               UserStats           `json:"stats"`
}

func (user *KVUser) GetUserIdentification() string {
	if len(user.Tenant) > 0 {
		return fmt.Sprintf("%s$%s", user.ID, user.Tenant)
	}
	return user.ID
}

func (user *KVUser) GetKVFriendlyUserIdentification() string {
	if len(user.Tenant) > 0 {
		return fmt.Sprintf("%s_tenant_%s", user.ID, user.Tenant)
	}
	return user.ID
}

type UserStats struct {
	Size        *uint64 `json:"size"`
	SizeRounded *uint64 `json:"sizeRounded"`
	NumObjects  *uint64 `json:"numObjects"`
}

func refreshUsersInKV(syncControl nats.KeyValue, userData nats.KeyValue, co *admin.API, prefix string) error {
	syncUsersKey := "sync_users"
	syncUsersInProgressKey := "sync_users_in_progress"

	// Check if a global sync is already in progress
	_, err := syncControl.Get(syncUsersInProgressKey)
	if err == nil {
		log.Debug().Msg("Global user sync already in progress; skipping")
		return nil
	}

	// Set the global sync in-progress key
	if _, err := syncControl.Put(syncUsersInProgressKey, []byte("true")); err != nil {
		return fmt.Errorf("failed to set global sync in-progress key: %w", err)
	}
	defer func() {
		if err := syncControl.Delete(syncUsersInProgressKey); err != nil {
			log.Warn().Err(err).Msg("Failed to delete global sync in-progress key")
		}
	}()

	// Check if global user sync is required
	globalSync := false
	value, err := syncControl.Get(syncUsersKey)
	if err != nil {
		log.Warn().Msgf("Key %s not found; initializing with true", syncUsersKey)
		if _, putErr := syncControl.Put(syncUsersKey, []byte("true")); putErr != nil {
			return fmt.Errorf("failed to initialize %s key: %w", syncUsersKey, putErr)
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

	// Prepare list of granular users to sync
	granularUserIDs := []string{}
	for _, key := range keys {
		if strings.HasPrefix(key, "sync_user_") {
			userID := strings.TrimPrefix(key, "sync_user_")
			granularUserIDs = append(granularUserIDs, userID)
		}
	}

	// If no global or granular sync is required, return early
	if !globalSync && len(granularUserIDs) == 0 {
		log.Debug().Msg("No users to sync; skipping")
		return nil
	}

	log.Debug().Msgf("Global sync: %v, Granular sync: %v", globalSync, granularUserIDs)

	// Process users in a single pass
	usersFailed := 0
	if globalSync {
		log.Debug().Msg("Performing global user sync")
		allUsers, err := fetchAllUsers(co, syncControl, prefix)
		if err != nil {
			return fmt.Errorf("failed to fetch all users: %w", err)
		}

		for _, user := range allUsers {
			if err := storeUserInKV(user, userData); err != nil {
				usersFailed++
			}
		}

		// Reset the global sync flag
		if _, err := syncControl.Put(syncUsersKey, []byte("false")); err != nil {
			log.Warn().Err(err).Msg("Failed to reset global sync_users flag")
		}
	}

	// Process granular user sync in the same loop
	for _, userID := range granularUserIDs {
		userInfo, err := co.GetUser(context.Background(), admin.User{ID: userID})
		if err != nil {
			log.Warn().Str("user", userID).Err(err).Msg("Failed to fetch user for granular sync")
			usersFailed++
			continue
		}

		if err := storeUserInKV(userInfo, userData); err != nil {
			usersFailed++
			continue
		}

		// Reset the granular sync flag
		userKey := fmt.Sprintf("sync_user_%s", userID)
		if err := syncControl.Delete(userKey); err != nil {
			log.Warn().Str("user", userID).Err(err).Msg("Failed to reset granular sync flag")
		}
	}

	log.Debug().
		Int("usersFailed", usersFailed).
		Msg("Completed user sync in KV")

	return nil
}

func checkAndRefreshUsers(syncControl nats.KeyValue, userData nats.KeyValue, cfg RadosGWUsageConfig, status *PrysmStatus) {
	// Validate the configuration to ensure necessary fields are set
	if cfg.AdminURL == "" || cfg.AccessKey == "" || cfg.SecretKey == "" {
		log.Fatal().Msg("invalid configuration: AdminURL, AccessKey, and SecretKey must be provided")
	}

	// Timers for incremental and full syncs
	fullSyncTicker := time.NewTicker(2 * time.Minute)         // Trigger full sync every 10 minutes
	incrementalSyncTicker := time.NewTicker(30 * time.Second) // Incremental sync every 30 seconds
	defer fullSyncTicker.Stop()
	defer incrementalSyncTicker.Stop()

	for {
		select {
		case <-fullSyncTicker.C:
			if isFlagSet(syncControl, "metric_calc_in_progress") {
				log.Debug().Msg("Skipping full user sync; metric calculation in progress")
				continue
			}
			if isFlagSet(syncControl, "sync_usages_in_progress") {
				log.Debug().Msg("Skipping full user sync; sync usage in progress")
				continue
			}

			// Create a new RadosGW admin client
			co, err := createRadosGWClient(cfg, status)
			if err != nil {
				log.Error().Err(err).Msg("Failed to create RadosGW admin client")
				continue
			}

			// Trigger global user sync
			log.Debug().Msg("Triggering periodic full user sync")
			if _, err := syncControl.Put("sync_users", []byte("true")); err != nil {
				log.Error().Err(err).Msg("Failed to trigger global user sync")
			}
			err = refreshUsersInKV(syncControl, userData, co, cfg.SyncControlBucketPrefix)
			if err != nil {
				status.IncrementScrapeErrors()
				log.Error().Err(err).Msg("Error during incremental user sync")
			} else {
				status.UpdateTargetUp(true)
			}

		case <-incrementalSyncTicker.C:
			if isFlagSet(syncControl, "metric_calc_in_progress") {
				log.Debug().Msg("Skipping incremental user sync; metric calculation in progress")
				continue
			}
			if isFlagSet(syncControl, "sync_usages_in_progress") {
				log.Debug().Msg("Skipping full user sync; sync usage in progress")
				continue
			}

			// Create a new RadosGW admin client
			co, err := createRadosGWClient(cfg, status)
			if err != nil {
				log.Error().Err(err).Msg("Failed to create RadosGW admin client")
				continue
			}

			// Perform incremental sync
			log.Debug().Msg("Starting incremental user sync")
			err = refreshUsersInKV(syncControl, userData, co, cfg.SyncControlBucketPrefix)
			if err != nil {
				status.IncrementScrapeErrors()
				log.Error().Err(err).Msg("Error during incremental user sync")
			} else {
				status.UpdateTargetUp(true)
			}
		}
	}
}

func triggerUserSync(syncControl nats.KeyValue, userID string) error {
	userSyncKey := fmt.Sprintf("sync_user_%s", userID)

	// Set the sync flag to true
	if _, err := syncControl.Put(userSyncKey, []byte("true")); err != nil {
		return fmt.Errorf("failed to set sync flag for user %s: %w", userID, err)
	}

	log.Info().Str("user", userID).Msg("Triggered sync for user")
	return nil
}

// fetchAllUsers retrieves all user metadata from RADOSGW using go-ceph's admin client.
func fetchAllUsers(co *admin.API, syncControl nats.KeyValue, prefix string) ([]admin.User, error) {
	// Step 1: Fetch the list of user IDs
	userIDs, err := co.GetUsers(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to get user list: %v", err)
	}

	// Channels for collecting results and errors
	userDataCh := make(chan admin.User, len(*userIDs))
	errCh := make(chan string, len(*userIDs))

	// Step 2: Fetch user metadata concurrently
	for _, userName := range *userIDs {
		go func(userName string) {
			if fetchUserInfo(co, syncControl, userName, userDataCh, errCh) {
				log.Trace().
					Str("user", userName).
					Msg("successfully fetched user info")
			}
		}(userName)
	}

	// Step 3: Collect results from the channels
	var userData []admin.User
	var usersProcessed, usersFailed int

	for i := 0; i < len(*userIDs); i++ {
		select {
		case data := <-userDataCh:
			userData = append(userData, data)
			usersProcessed++
		case userID := <-errCh:
			log.Warn().
				Str("user", userID).
				Msg("error received during user data collection; marking for next sync")
			usersFailed++
		}
	}

	// Close channels
	close(userDataCh)
	close(errCh)

	// Step 4: Log summary and return results
	log.Debug().
		Int("usersProcessed", usersProcessed).
		Int("usersFailed", usersFailed).
		Msg("completed user data collection")

	return userData, nil
}

func fetchUserInfo(co *admin.API, syncControl nats.KeyValue, userID string, userDataCh chan admin.User, errCh chan string) bool {
	const maxRetries = 3
	for attempt := 1; attempt <= maxRetries; attempt++ {
		userInfo, err := co.GetUser(context.Background(), admin.User{ID: userID, GenerateStat: BoolPtr(true)})
		if err != nil {
			log.Error().
				Str("user", userID).
				Int("attempt", attempt).
				Err(err).
				Msg("error fetching user info")

			if outputToFile {
				logErrorToFile(err, userID)
			}

			// Retry if not on the last attempt
			if attempt < maxRetries {
				time.Sleep(2 * time.Second) // Backoff between retries
				continue
			}

			// Mark user for next sync if retries fail
			if _, syncErr := syncControl.Put(fmt.Sprintf("sync_user_%s", userID), []byte("true")); syncErr != nil {
				log.Error().Str("user", userID).Err(syncErr).Msg("Failed to set sync flag for user after retries")
			}
			errCh <- userID
			return false
		}

		// On success, send the user info to the channel
		userDataCh <- userInfo
		return true
	}

	return false
}

func storeUserInKV(user admin.User, userData nats.KeyValue) error {
	kvUser := KVUser{
		ID:                  user.ID,
		DisplayName:         user.DisplayName,
		Email:               user.Email,
		Suspended:           user.Suspended,
		MaxBuckets:          user.MaxBuckets,
		Caps:                user.Caps,
		OpMask:              user.OpMask,
		DefaultPlacement:    user.DefaultPlacement,
		DefaultStorageClass: user.DefaultStorageClass,
		PlacementTags:       user.PlacementTags,
		BucketQuota:         user.BucketQuota,
		UserQuota:           user.UserQuota,
		Type:                user.Type,
		Tenant:              user.Tenant,
		Stats: UserStats{
			Size:        user.Stat.Size,
			SizeRounded: user.Stat.SizeRounded,
			NumObjects:  user.Stat.NumObjects,
		},
	}

	userDataJSON, err := json.Marshal(kvUser)
	if err != nil {
		log.Error().
			Str("user", user.ID).
			Err(err).
			Msg("Error serializing user data")
		return err
	}

	userKey := fmt.Sprintf("user_%s", kvUser.GetKVFriendlyUserIdentification())
	if _, err := userData.Put(userKey, userDataJSON); err != nil {
		log.Warn().
			Str("user", kvUser.GetUserIdentification()).
			Err(err).
			Msg("Failed to update KV for user")
		return err
	}

	return nil
}
