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

	"github.com/cobaltcore-dev/prysm/pkg/producers/radosgwusage/rgwadmin"
	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog/log"
)

func syncUsers(userData nats.KeyValue, cfg RadosGWUsageConfig, status *PrysmStatus) error {
	log.Info().Msg("Starting user synchronization")

	// Create RadosGW admin client
	co, err := createRadosGWClient(cfg, status)
	if err != nil {
		log.Error().Err(err).Msg("Failed to create RadosGW admin client")
		return err
	}

	// Fetch and store all users with concurrency control
	err = fetchAllUsers(co, userData)
	if err != nil {
		log.Error().Err(err).Msg("Failed to fetch users")
		return err
	}

	log.Info().Msg("User synchronization completed")
	return nil
}

func fetchAllUsers(co *rgwadmin.API, userData nats.KeyValue) error {
	userIDs, err := co.GetUsers(context.Background())
	if err != nil {
		return fmt.Errorf("failed to get user list: %v", err)
	}

	userDataCh := make(chan rgwadmin.KVUser, len(userIDs))
	errCh := make(chan string, len(userIDs))

	var wg sync.WaitGroup
	const maxConcurrency = 10
	sem := make(chan struct{}, maxConcurrency)

	for _, userName := range userIDs {
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

	// var userData []rgwadmin.KVUser
	var usersProcessed, usersFailed int

	for data := range userDataCh {
		// userData = append(userData, data)
		storeUserInKV(data, userData)
		usersProcessed++
	}

	for range errCh {
		usersFailed++
	}

	log.Debug().
		Int("usersProcessed", usersProcessed).
		Int("usersFailed", usersFailed).
		Msg("Completed user data collection")

	return nil
}

func fetchUserInfo(co *rgwadmin.API, userID string, userDataCh chan rgwadmin.KVUser, errCh chan string) {
	maxRetries := 3

	for attempt := 1; attempt <= maxRetries; attempt++ {
		userInfo, err := co.GetKVUser(context.Background(), rgwadmin.User{ID: userID, GenerateStat: ptr(true)})
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

func storeUserInKV(user rgwadmin.KVUser, userData nats.KeyValue) error {
	userDataJSON, err := json.Marshal(user)
	if err != nil {
		log.Error().
			Str("user", user.ID).
			Err(err).
			Msg("Error serializing user data")
	}

	userKey := BuildUserTenantKey(user.ID, user.Tenant)
	if _, err := userData.Put(userKey, userDataJSON); err != nil {
		log.Warn().
			Str("user", userKey).
			Err(err).
			Msg("Failed to update KV for user")
	}

	log.Debug().Str("user", user.ID).Msg("Successfully stored user in KV")
	return nil
}
