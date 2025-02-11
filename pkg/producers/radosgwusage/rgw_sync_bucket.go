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

func syncBuckets(bucketData nats.KeyValue, cfg RadosGWUsageConfig, status *PrysmStatus) error {
	log.Info().Msg("Starting bucket sync process")

	// Initialize the RadosGW client
	co, err := createRadosGWClient(cfg, status)
	if err != nil {
		log.Error().Err(err).Msg("Failed to create RadosGW admin client")
		return err
	}

	// Fetch all buckets
	err = fetchAllBuckets(co, bucketData)
	if err != nil {
		log.Error().Err(err).Msg("Failed to fetch all buckets")
		return err
	}

	log.Info().Msg("Bucket synchronization completed")

	return nil
}

func fetchAllBuckets(co *rgwadmin.API, bucketData nats.KeyValue) error {
	// Step 1: Fetch the list of bucket names
	bucketNames, err := co.ListBuckets(context.Background())
	if err != nil {
		return fmt.Errorf("failed to list buckets: %w", err)
	}

	log.Info().Int("total_buckets", len(bucketNames)).Msg("Fetched bucket names")

	// Step 2: Create channels for results and errors
	bucketDataCh := make(chan rgwadmin.Bucket, len(bucketNames))
	errCh := make(chan string, len(bucketNames))

	// Step 3: Use a WaitGroup and semaphore to fetch bucket details concurrently
	var wg sync.WaitGroup
	const maxConcurrency = 10 // Limit concurrent requests
	sem := make(chan struct{}, maxConcurrency)

	for _, bucketName := range bucketNames {
		wg.Add(1)
		sem <- struct{}{} // Acquire a semaphore token
		go func(bucketName string) {
			defer wg.Done()
			defer func() { <-sem }() // Release the token when done

			bucketInfo, err := fetchBucketInfo(co, bucketName)
			if err != nil {
				errCh <- bucketName
				return
			}
			bucketDataCh <- bucketInfo
		}(bucketName)
	}

	// Wait for all goroutines to finish
	wg.Wait()
	close(bucketDataCh)
	close(errCh)

	// Step 4: Collect results from channels
	// var bucketData []rgwadmin.Bucket
	var bucketsProcessed, bucketsFailed int

	for bucket := range bucketDataCh {
		// bucketData = append(bucketData, bucket)
		storeBucketInKV(bucket, bucketData)
		bucketsProcessed++
	}

	for bucketName := range errCh {
		log.Warn().Str("bucket", bucketName).Msg("Failed to fetch bucket details")
		bucketsFailed++
	}

	// Step 5: Log a summary and return results
	log.Info().
		Int("buckets_processed", bucketsProcessed).
		Int("buckets_failed", bucketsFailed).
		Msg("Bucket data collection completed")

	return nil
}

func fetchBucketInfo(co *rgwadmin.API, bucketName string) (rgwadmin.Bucket, error) {
	const maxRetries = 3
	var bucketInfo rgwadmin.Bucket
	var err error

	for attempt := 1; attempt <= maxRetries; attempt++ {
		bucketInfo, err = co.GetBucketInfo(context.Background(), rgwadmin.Bucket{Bucket: bucketName})
		if err == nil {
			return bucketInfo, nil // Success!
		}

		log.Warn().
			Str("bucket", bucketName).
			Int("attempt", attempt).
			Err(err).
			Msg("Error fetching bucket info, retrying...")

		// Exponential backoff: wait longer on each retry
		time.Sleep(time.Duration(attempt*2) * time.Second)
	}

	log.Error().
		Str("bucket", bucketName).
		Err(err).
		Msg("Failed to fetch bucket info after retries")
	return rgwadmin.Bucket{}, fmt.Errorf("failed to fetch bucket %s after %d retries: %w", bucketName, maxRetries, err)
}

func storeBucketInKV(bucket rgwadmin.Bucket, bucketData nats.KeyValue) error {
	bucketDataJSON, err := json.Marshal(bucket)
	if err != nil {
		log.Error().
			Str("bucket", bucket.Bucket).
			Err(err).
			Msg("Error serializing bucket data")
		return err
	}

	bucketKey := BuildUserTenantBucketKey(bucket.Owner, bucket.Tenant, bucket.Bucket)
	if _, err := bucketData.Put(bucketKey, bucketDataJSON); err != nil {
		log.Warn().
			Str("bucket", bucket.Bucket).
			Err(err).
			Msg("Failed to update KV for bucket")
		return err
	}

	log.Debug().Str("bucket", bucket.Bucket).Msg("Successfully stored bucket in KV")
	return nil
}
