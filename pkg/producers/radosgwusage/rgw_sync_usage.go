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

const (
	rootBucketPlaceholder        = "root"
	nonBucketSpecificPlaceholder = "-"
)

// type KVUserUsage struct {
// 	ID          string        `json:"id"`
// 	LastUpdated time.Time     `json:"lastUpdated"`
// 	Usage       UserUsageSpec `json:"usage"`
// }

func syncUsage(userUsageData nats.KeyValue, cfg RadosGWUsageConfig, status *PrysmStatus) error {
	log.Info().Msg("Starting usage sync process")

	// Create a new RadosGW admin client.
	co, err := createRadosGWClient(cfg, status)
	if err != nil {
		log.Error().Err(err).Msg("Failed to create RadosGW admin client")
		return err
	}

	// Fetch and store global usage (for all users).
	err = fetchUserUsageGlobal(co, userUsageData)
	if err != nil {
		log.Error().Err(err).Msg("Failed to fetch global user usage")
		return err
	}

	log.Info().Msg("Usage synchronization completed")
	return nil
}

func fetchUserUsageGlobal(co *rgwadmin.API, userUsageData nats.KeyValue) error {
	// Fetch the initial global usage data.
	// globalUsage, err := co.GetUsage(context.Background(), rgwadmin.Usage{
	// 	ShowEntries: ptr(true),
	// 	ShowSummary: ptr(false),
	// })
	// if err != nil {
	// 	return fmt.Errorf("failed to fetch global usage: %w", err)
	// }

	// if len(globalUsage.Entries) == 0 {
	// 	return nil
	// }
	userIDs, err := co.GetUsers(context.Background())
	if err != nil {
		return fmt.Errorf("failed to get user list: %v", err)
	}

	usageDataCh := make(chan rgwadmin.Usage, len(userIDs))
	errCh := make(chan string, len(userIDs))
	// usageDataCh := make(chan rgwadmin.Usage, len(globalUsage.Entries))
	// errCh := make(chan string, len(globalUsage.Entries))

	var wg sync.WaitGroup
	const maxConcurrency = 10
	sem := make(chan struct{}, maxConcurrency)

	// for _, entry := range globalUsage.Entries {
	for _, entry := range userIDs {
		wg.Add(1)
		sem <- struct{}{} // Acquire a semaphore token
		go func(userID string) {
			defer wg.Done()
			defer func() { <-sem }() // Release token when done
			fetchUsageDetails(co, userID, usageDataCh, errCh)

		}(entry)
		// }(entry.User)
	}

	wg.Wait()
	close(usageDataCh)
	close(errCh)

	// var userData []rgwadmin.KVUser
	var usageProcessed, usageFailed int

	for data := range usageDataCh {
		// userData = append(userData, data)
		storeUserUsageInKV(data, userUsageData)
		usageProcessed++
	}

	for range errCh {
		usageFailed++
	}

	log.Debug().
		Int("usageProcessed", usageProcessed).
		Int("usageFailed", usageFailed).
		Msg("Completed usage data collection")

	return nil
}

func fetchUsageDetails(co *rgwadmin.API, userID string, usageDataCh chan rgwadmin.Usage, errCh chan string) {
	maxRetries := 3

	for attempt := 1; attempt <= maxRetries; attempt++ {
		usageData, err := co.GetUsage(context.Background(), rgwadmin.Usage{
			UserID:      userID,
			ShowEntries: ptr(true),
		})
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

		usageDataCh <- usageData
		return
	}
}

func storeUserUsageInKV(userUsage rgwadmin.Usage, userUsageData nats.KeyValue) error {
	bucketsFailed := 0
	skippedBuckets := 0

	// Process each usage entry (for each user)
	for _, entry := range userUsage.Entries {
		// Process each bucket for that user
		for _, bucket := range entry.Buckets {
			bucketName := bucket.Bucket
			if bucketName == "" {
				bucketName = rootBucketPlaceholder // e.g., "root"
			}
			if bucketName == nonBucketSpecificPlaceholder {
				skippedBuckets++
				log.Debug().
					Str("user", entry.User).
					Msg("Skipping non-bucket-specific usage ('-')")
				continue
			}

			// Serialize the usage data.
			bucketDataJSON, err := json.Marshal(bucket)
			if err != nil {
				log.Error().
					Str("user", entry.User).
					Str("bucket", bucket.Bucket).
					Err(err).
					Msg("Error serializing bucket usage data")
				bucketsFailed++
				continue
			}

			// Build a key using a helper that encodes components safely.
			user, tenant := SplitUserTenant(entry.User) // Example helper: splits user string into user & tenant parts.
			bucketKey := BuildUserTenantBucketKey(user, tenant, bucketName)

			// Write the serialized data to the KV store.
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
