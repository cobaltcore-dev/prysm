// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and prysm contributors
//
// SPDX-License-Identifier: Apache-2.0
package radosgwusage

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/ceph/go-ceph/rgw/admin"
	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog/log"
)

type KVBucket struct {
	Bucket            string            `json:"bucket"`
	NumShards         *uint64           `json:"num_shards"`
	Tenant            string            `json:"tenant"`
	Zonegroup         string            `json:"zonegroup"`
	PlacementRule     string            `json:"placementRule"`
	ExplicitPlacement ExplicitPlacement `json:"explicit_placement"`
	ID                string            `json:"id"`
	Marker            string            `json:"marker"`
	IndexType         string            `json:"index_type"`
	Owner             string            `json:"owner"`
	Ver               string            `json:"ver"`
	MasterVer         string            `json:"master_ver"`
	Mtime             string            `json:"mtime"`
	CreationTime      *time.Time        `json:"creationTime"`
	MaxMarker         string            `json:"max_marker"`
	Usage             BucketUsageSpec   `json:"usage"`
	BucketQuota       admin.QuotaSpec   `json:"bucketQuota"`
}

func (bucket *KVBucket) IsOwnedBy(owner string) bool {
	tmpOwner := strings.ReplaceAll(owner, "_tenant_", "$")
	return bucket.Owner == tmpOwner
}

type ExplicitPlacement struct {
	DataPool      string `json:"data_pool"`
	DataExtraPool string `json:"data_extra_pool"`
	IndexPool     string `json:"index_pool"`
}

type BucketUsageSpec struct {
	Main      RgwMain      `json:"rgw.main"`
	Multimeta RgwMultimeta `json:"rgw.multimeta"`
}

type RgwMain struct {
	Size           *uint64 `json:"size"`
	SizeActual     *uint64 `json:"size_actual"`
	SizeUtilized   *uint64 `json:"size_utilized"`
	SizeKb         *uint64 `json:"size_kb"`
	SizeKbActual   *uint64 `json:"size_kb_actual"`
	SizeKbUtilized *uint64 `json:"size_kb_utilized"`
	NumObjects     *uint64 `json:"num_objects"`
}

type RgwMultimeta struct {
	Size           *uint64 `json:"size"`
	SizeActual     *uint64 `json:"size_actual"`
	SizeUtilized   *uint64 `json:"size_utilized"`
	SizeKb         *uint64 `json:"size_kb"`
	SizeKbActual   *uint64 `json:"size_kb_actual"`
	SizeKbUtilized *uint64 `json:"size_kb_utilized"`
	NumObjects     *uint64 `json:"num_objects"`
}

func syncBuckets(bucketData nats.KeyValue, cfg RadosGWUsageConfig, status *PrysmStatus) error {
	log.Info().Msg("Starting bucket sync process")

	// Initialize the RadosGW client
	co, err := createRadosGWClient(cfg, status)
	if err != nil {
		log.Error().Err(err).Msg("Failed to create RadosGW admin client")
		return err
	}

	// Fetch all buckets
	buckets, err := fetchAllBuckets(co)
	if err != nil {
		log.Error().Err(err).Msg("Failed to fetch all buckets")
		return err
	}

	// Store each bucket in KV
	bucketsFailed := 0
	for _, bucket := range buckets {
		if err := storeBucketInKV(bucket, bucketData); err != nil {
			bucketsFailed++
		}
	}

	log.Info().
		Int("total_buckets", len(buckets)).
		Int("failed_buckets", bucketsFailed).
		Msg("Completed bucket sync process")

	return nil
}

func fetchAllBuckets(co *admin.API) ([]admin.Bucket, error) {
	// Step 1: Fetch the list of bucket names
	bucketNames, err := co.ListBuckets(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to list buckets: %w", err)
	}

	log.Info().Int("total_buckets", len(bucketNames)).Msg("Fetched bucket names")

	// Step 2: Create channels for results and errors
	bucketDataCh := make(chan admin.Bucket, len(bucketNames))
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
	var bucketData []admin.Bucket
	var bucketsProcessed, bucketsFailed int

	for bucket := range bucketDataCh {
		bucketData = append(bucketData, bucket)
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

	return bucketData, nil
}

func fetchBucketInfo(co *admin.API, bucketName string) (admin.Bucket, error) {
	const maxRetries = 3
	var bucketInfo admin.Bucket
	var err error

	for attempt := 1; attempt <= maxRetries; attempt++ {
		bucketInfo, err = co.GetBucketInfo(context.Background(), admin.Bucket{Bucket: bucketName})
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
	return admin.Bucket{}, fmt.Errorf("failed to fetch bucket %s after %d retries: %w", bucketName, maxRetries, err)
}

func storeBucketInKV(bucket admin.Bucket, bucketData nats.KeyValue) error {
	bucketKey := fmt.Sprintf("bucket_%s", bucket.Bucket)
	kvBucket := KVBucket{
		Bucket:            bucket.Bucket,
		NumShards:         bucket.NumShards,
		Tenant:            bucket.Tenant,
		Zonegroup:         bucket.Zonegroup,
		PlacementRule:     bucket.PlacementRule,
		ExplicitPlacement: bucket.ExplicitPlacement,
		ID:                bucket.ID,
		Marker:            bucket.Marker,
		IndexType:         bucket.IndexType,
		Owner:             bucket.Owner,
		Ver:               bucket.Ver,
		MasterVer:         bucket.MasterVer,
		Mtime:             bucket.Mtime,
		CreationTime:      bucket.CreationTime,
		MaxMarker:         bucket.MaxMarker,
		Usage: BucketUsageSpec{
			Main: RgwMain{
				Size:           bucket.Usage.RgwMain.Size,
				SizeActual:     bucket.Usage.RgwMain.SizeActual,
				SizeUtilized:   bucket.Usage.RgwMain.SizeUtilized,
				SizeKb:         bucket.Usage.RgwMain.SizeKb,
				SizeKbActual:   bucket.Usage.RgwMain.SizeKbActual,
				SizeKbUtilized: bucket.Usage.RgwMain.SizeKbUtilized,
				NumObjects:     bucket.Usage.RgwMain.NumObjects,
			},
			Multimeta: RgwMultimeta{
				Size:           bucket.Usage.RgwMultimeta.Size,
				SizeActual:     bucket.Usage.RgwMultimeta.SizeActual,
				SizeUtilized:   bucket.Usage.RgwMultimeta.SizeUtilized,
				SizeKb:         bucket.Usage.RgwMultimeta.SizeKb,
				SizeKbActual:   bucket.Usage.RgwMultimeta.SizeKbActual,
				SizeKbUtilized: bucket.Usage.RgwMultimeta.SizeKbUtilized,
				NumObjects:     bucket.Usage.RgwMultimeta.NumObjects,
			},
		},
		BucketQuota: bucket.BucketQuota,
	}

	bucketDataJSON, err := json.Marshal(kvBucket)
	if err != nil {
		log.Error().
			Str("bucket", bucket.Bucket).
			Err(err).
			Msg("Error serializing bucket data")
		return err
	}

	if _, err := bucketData.Put(bucketKey, bucketDataJSON); err != nil {
		log.Warn().
			Str("bucket", bucket.Bucket).
			Err(err).
			Msg("Failed to update KV for bucket")
		return err
	}

	return nil
}
