// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and prysm contributors
//
// SPDX-License-Identifier: Apache-2.0
package radosgwusage

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
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

func syncUsers(userData nats.KeyValue, cfg RadosGWUsageConfig, status *PrysmStatus) error {
	log.Info().Msg("Starting user synchronization")

	// Create RadosGW admin client
	co, err := createRadosGWClient(cfg, status)
	if err != nil {
		log.Error().Err(err).Msg("Failed to create RadosGW admin client")
		return err
	}

	// Fetch all users with concurrency control
	users, err := fetchAllUsers(co)
	if err != nil {
		log.Error().Err(err).Msg("Failed to fetch users")
		return err
	}

	// Store users in KV store in batch (could also be done concurrently if needed)
	err = storeUsersInKV(users, userData)
	if err != nil {
		log.Error().Err(err).Msg("Failed to store users in KV")
		return err
	}
	log.Info().Msg("User synchronization completed")
	return nil
}

func fetchAllUsers(co *admin.API) ([]admin.User, error) {
	userIDs, err := co.GetUsers(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to get user list: %v", err)
	}

	userDataCh := make(chan admin.User, len(*userIDs))
	errCh := make(chan string, len(*userIDs))

	var wg sync.WaitGroup
	const maxConcurrency = 10
	sem := make(chan struct{}, maxConcurrency)

	for _, userName := range *userIDs {
		wg.Add(1)
		sem <- struct{}{}
		go func(userName string) {
			defer wg.Done()
			defer func() { <-sem }()
			fetchUserInfo(co, userName, userDataCh, errCh)
		}(userName)
	}

	wg.Wait()
	close(userDataCh)
	close(errCh)

	var userData []admin.User
	var usersProcessed, usersFailed int

	for data := range userDataCh {
		userData = append(userData, data)
		usersProcessed++
	}

	for range errCh {
		usersFailed++
	}

	log.Debug().
		Int("usersProcessed", usersProcessed).
		Int("usersFailed", usersFailed).
		Msg("Completed user data collection")

	return userData, nil
}

func fetchUserInfo(co *admin.API, userID string, userDataCh chan admin.User, errCh chan string) {
	const maxRetries = 3

	for attempt := 1; attempt <= maxRetries; attempt++ {
		userInfo, err := co.GetUser(context.Background(), admin.User{ID: userID, GenerateStat: ptr(true)})
		if err != nil {
			log.Error().
				Str("user", userID).
				Int("attempt", attempt).
				Err(err).
				Msg("Error fetching user info")

			if attempt < maxRetries {
				time.Sleep(2 * time.Second)
				continue
			}

			errCh <- userID
			return
		}

		userDataCh <- userInfo
		return
	}
}

func storeUsersInKV(users []admin.User, userData nats.KeyValue) error {
	if len(users) == 0 {
		log.Warn().Msg("No users to store in KV")
		return nil
	}

	// Batch store users
	for _, user := range users {
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
			continue // Skip storing this user but continue others
		}

		userKey := fmt.Sprintf("user_%s", kvUser.GetKVFriendlyUserIdentification())
		if _, err := userData.Put(userKey, userDataJSON); err != nil {
			log.Warn().
				Str("user", kvUser.GetUserIdentification()).
				Err(err).
				Msg("Failed to update KV for user")
		}
	}

	log.Info().Int("users_stored", len(users)).Msg("Successfully stored users in KV")
	return nil
}
