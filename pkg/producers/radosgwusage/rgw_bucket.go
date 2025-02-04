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

func refreshBucketsInKV(syncControl nats.KeyValue, bucketData nats.KeyValue, co *admin.API, prefix string) error {
	syncBucketsKey := "sync_buckets"
	syncBucketsInProgressKey := "sync_buckets_in_progress"

	// Check if a global sync is already in progress
	_, err := syncControl.Get(syncBucketsInProgressKey)
	if err == nil {
		log.Debug().Msg("Global bucket sync already in progress; skipping")
		return nil
	}

	// Set the global sync in-progress key
	if _, err := syncControl.Put(syncBucketsInProgressKey, []byte("true")); err != nil {
		return fmt.Errorf("failed to set global sync in-progress key: %w", err)
	}
	defer func() {
		if err := syncControl.Delete(syncBucketsInProgressKey); err != nil {
			log.Warn().Err(err).Msg("Failed to delete global sync in-progress key")
		}
	}()

	// Check if global bucket sync is required
	globalSync := false
	value, err := syncControl.Get(syncBucketsKey)
	if err != nil {
		log.Warn().Msgf("Key %s not found; initializing with true", syncBucketsKey)
		if _, putErr := syncControl.Put(syncBucketsKey, []byte("true")); putErr != nil {
			return fmt.Errorf("failed to initialize %s key: %w", syncBucketsKey, putErr)
		}
		globalSync = true
	} else {
		globalSync = value != nil && string(value.Value()) == "true"
	}

	// Fetch granular bucket sync flags
	keys, err := syncControl.Keys()
	if err != nil {
		log.Warn().Msg("sync_control bucket is empty; initializing")
		keys = []string{}
	}

	// Prepare list of granular buckets to sync
	granularBucketIDs := []string{}
	for _, key := range keys {
		if strings.HasPrefix(key, "sync_bucket_") {
			bucketID := strings.TrimPrefix(key, "sync_bucket_")
			granularBucketIDs = append(granularBucketIDs, bucketID)
		}
	}

	// If no global or granular sync is required, return early
	if !globalSync && len(granularBucketIDs) == 0 {
		log.Debug().Msg("No buckets to sync; skipping")
		return nil
	}

	log.Debug().Msgf("Global sync: %v, Granular sync: %v", globalSync, granularBucketIDs)

	// Process buckets in a single pass
	bucketsFailed := 0
	if globalSync {
		log.Debug().Msg("Performing global bucket sync")
		allBuckets, err := fetchAllBuckets(co, syncControl, prefix)
		if err != nil {
			return fmt.Errorf("failed to fetch all buckets: %w", err)
		}

		for _, bucket := range allBuckets {
			if err := storeBucketInKV(bucket, bucketData); err != nil {
				bucketsFailed++
			}
		}

		// Reset the global sync flag
		if _, err := syncControl.Put(syncBucketsKey, []byte("false")); err != nil {
			log.Warn().Err(err).Msg("Failed to reset global sync_buckets flag")
		}
	}

	// Process granular bucket sync in the same loop
	for _, bucketID := range granularBucketIDs {
		bucketInfo, err := co.GetBucketInfo(context.Background(), admin.Bucket{Bucket: bucketID})
		if err != nil {
			log.Warn().Str("bucket", bucketID).Err(err).Msg("Failed to fetch bucket for granular sync")
			bucketsFailed++
			continue
		}

		if err := storeBucketInKV(bucketInfo, bucketData); err != nil {
			bucketsFailed++
			continue
		}

		// Reset the granular sync flag
		bucketKey := fmt.Sprintf("sync_bucket_%s", bucketID)
		if err := syncControl.Delete(bucketKey); err != nil {
			log.Warn().Str("bucket", bucketID).Err(err).Msg("Failed to reset granular sync flag")
		}
	}

	log.Debug().
		Int("bucketsFailed", bucketsFailed).
		Msg("Completed bucket sync in KV")

	return nil
}

func checkAndRefreshBuckets(syncControl nats.KeyValue, bucketData nats.KeyValue, cfg RadosGWUsageConfig, status *PrysmStatus) {
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
				log.Debug().Msg("Skipping full bucket sync; metric calculation in progress")
				continue
			}

			// Create a new RadosGW admin client
			co, err := createRadosGWClient(cfg, status)
			if err != nil {
				log.Error().Err(err).Msg("Failed to create RadosGW admin client")
				continue
			}

			// Trigger global sync
			log.Debug().Msg("Triggering periodic full bucket sync")
			if _, err := syncControl.Put("sync_buckets", []byte("true")); err != nil {
				log.Error().Err(err).Msg("Failed to trigger global bucket sync")
			}
			err = refreshBucketsInKV(syncControl, bucketData, co, cfg.SyncControlBucketPrefix)
			if err != nil {
				log.Error().Err(err).Msg("Error during incremental bucket sync")
			}

		case <-incrementalSyncTicker.C:
			if isFlagSet(syncControl, "metric_calc_in_progress") {
				log.Debug().Msg("Skipping incremental bucket sync; metric calculation in progress")
				continue
			}

			// Create a new RadosGW admin client
			co, err := createRadosGWClient(cfg, status)
			if err != nil {
				log.Error().Err(err).Msg("Failed to create RadosGW admin client")
				continue
			}

			// Perform incremental sync
			log.Debug().Msg("Starting incremental bucket sync")
			err = refreshBucketsInKV(syncControl, bucketData, co, cfg.SyncControlBucketPrefix)
			if err != nil {
				log.Error().Err(err).Msg("Error during incremental bucket sync")
			}
		}
	}
}

func triggerBucketSync(syncControl nats.KeyValue, bucketID string) error {
	bucketSyncKey := fmt.Sprintf("sync_bucket_%s", bucketID)

	// Set the sync flag to true
	if _, err := syncControl.Put(bucketSyncKey, []byte("true")); err != nil {
		return fmt.Errorf("failed to set sync flag for bucket %s: %w", bucketID, err)
	}

	log.Info().Str("bucket", bucketID).Msg("Triggered sync for bucket")
	return nil
}

func fetchAllBuckets(co *admin.API, syncControl nats.KeyValue, prefix string) ([]admin.Bucket, error) {
	// Step 1: Fetch the list of bucket names
	bucketNames, err := co.ListBuckets(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to list buckets: %w", err)
	}

	// Channels for collecting results and errors
	bucketDataCh := make(chan admin.Bucket, len(bucketNames))
	errCh := make(chan string, len(bucketNames))

	// Step 2: Fetch bucket details concurrently
	for _, bucketName := range bucketNames {
		go func(bucketName string) {
			if fetchBucketInfo(co, syncControl, bucketName, bucketDataCh, errCh) {
				log.Trace().
					Str("bucket", bucketName).
					Msg("successfully fetched bucket info")
			}
		}(bucketName)
	}

	// Step 3: Collect results from the channels
	var bucketData []admin.Bucket
	var bucketsProcessed, bucketsFailed int

	for i := 0; i < len(bucketNames); i++ {
		select {
		case data := <-bucketDataCh:
			bucketData = append(bucketData, data)
			bucketsProcessed++
		case bucketID := <-errCh:
			log.Warn().
				Str("bucket", bucketID).
				Msg("error received during bucket data collection; marking for next sync")
			bucketsFailed++
		}
	}

	// Close the channels
	close(bucketDataCh)
	close(errCh)

	// Step 4: Log the results and return
	log.Info().
		Int("buckets_processed", bucketsProcessed).
		Int("buckets_failed", bucketsFailed).
		Msg("bucket data collection completed")

	return bucketData, nil
}

func fetchBucketInfo(co *admin.API, syncControl nats.KeyValue, bucketName string, bucketDataCh chan admin.Bucket, errCh chan string) bool {
	const maxRetries = 3
	for attempt := 1; attempt <= maxRetries; attempt++ {
		bucketInfo, err := co.GetBucketInfo(context.Background(), admin.Bucket{Bucket: bucketName})
		if err != nil {
			log.Error().
				Str("bucket", bucketName).
				Int("attempt", attempt).
				Err(err).
				Msg("error fetching info for bucket")

			if outputToFile {
				logErrorToFile(err, bucketName)
			}

			// Retry if not on the last attempt
			if attempt < maxRetries {
				time.Sleep(2 * time.Second) // Backoff between retries
				continue
			}

			// Mark bucket for next sync if retries fail
			if _, syncErr := syncControl.Put(fmt.Sprintf("sync_bucket_%s", bucketName), []byte("true")); syncErr != nil {
				log.Error().Str("bucket", bucketName).Err(syncErr).Msg("Failed to set sync flag for bucket after retries")
			}
			errCh <- bucketName
			return false
		}

		// On success, send the bucket info to the channel
		bucketDataCh <- bucketInfo
		return true
	}

	return false
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
