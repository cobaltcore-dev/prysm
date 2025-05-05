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

type UserBucketMetrics struct {
	BucketID        string
	User            string
	Tenant          string
	Zonegroup       string
	ObjectCount     uint64  // Number of objects in a bucket. Important for understanding the storage object count.
	BucketSize      uint64  // Total size consumed by the bucket, including all objects. Important for capacity tracking.
	CreationTime    string  // Knowing when a bucket was created can be useful for tracking lifecycle and access management.
	NumShards       *uint64 // Shards
	QuotaEnabled    bool
	QuotaMaxSize    *int64
	QuotaMaxObjects *int64
}

func (m *UserBucketMetrics) GetUserIdentification() string {
	if len(m.Tenant) > 0 {
		return fmt.Sprintf("%s$%s", m.User, m.Tenant)
	}
	return m.User
}

func updateBucketMetricsInKV(bucketData, userUsageData, bucketMetrics nats.KeyValue) error {
	log.Debug().Msg("Starting bucket-level metrics aggregation")

	bucketKeys, err := bucketData.Keys()
	if err != nil {
		log.Error().Err(err).Msg("Failed to fetch keys from bucket data")
		return fmt.Errorf("failed to fetch keys from bucket data: %w", err)
	}

	// Create a worker pool to process buckets concurrently.
	const numWorkers = 10
	bucketCh := make(chan string, len(bucketKeys))
	var wg sync.WaitGroup

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for key := range bucketCh {
				processBucketMetrics(key, bucketData, userUsageData, bucketMetrics)
			}
		}()
	}

	// Feed the channel.
	for _, key := range bucketKeys {
		bucketCh <- key
	}
	close(bucketCh)
	wg.Wait()

	log.Info().Msg("Completed bucket metrics aggregation and storage")
	return nil
}

func processBucketMetrics(key string, bucketData, userUsageData, bucketMetrics nats.KeyValue) {
	// Fetch bucket metadata
	entry, err := bucketData.Get(key)
	if err != nil {
		log.Warn().Str("bucket_key", key).Err(err).Msg("Failed to fetch bucket data from KV")
		return
	}

	var bucket rgwadmin.Bucket
	if err := json.Unmarshal(entry.Value(), &bucket); err != nil {
		log.Warn().Str("bucket_key", key).Err(err).Msg("Failed to unmarshal bucket data")
		return
	}

	log.Debug().
		Str("bucket_id", bucket.Bucket).
		Str("owner", bucket.Owner).
		Msg("Processing bucket metrics")

	// Initialize metrics.
	userID := bucket.Owner
	if strings.Index(userID, "$") > 0 { // if tenant is part of owner with devider $
		userID = userID[:strings.Index(userID, "$")]
	}
	metrics := UserBucketMetrics{
		BucketID:     bucket.Bucket,
		User:         userID,
		Tenant:       bucket.Tenant,
		CreationTime: bucket.Mtime, // Using Mtime as a substitute for creation time.
		Zonegroup:    bucket.Zonegroup,
	}

	// (Populate other static fields as needed.)
	if bucket.Usage.RgwMain.NumObjects != nil {
		metrics.ObjectCount = *bucket.Usage.RgwMain.NumObjects
	}
	if bucket.Usage.RgwMain.SizeActual != nil {
		metrics.BucketSize = *bucket.Usage.RgwMain.SizeActual
	}

	usageEntry, err := userUsageData.Get(key)
	if err != nil {
		log.Warn().Str("usage_key", key).Err(err).Msg("Failed to fetch usage data")
		return
	}

	var usage rgwadmin.UsageEntryBucket
	if err := json.Unmarshal(usageEntry.Value(), &usage); err != nil {
		log.Warn().Str("usage_key", key).Err(err).Msg("Failed to unmarshal usage data")
		return
	}

	// Set quota information.
	metrics.QuotaEnabled = false
	if bucket.BucketQuota.Enabled != nil && *bucket.BucketQuota.Enabled {
		metrics.QuotaEnabled = true
		metrics.QuotaMaxSize = bucket.BucketQuota.MaxSize
		metrics.QuotaMaxObjects = bucket.BucketQuota.MaxObjects
	}

	// Prepare the KV key for bucket metrics.
	metricsJSON, err := json.Marshal(metrics)
	if err != nil {
		log.Error().
			Str("bucket_id", bucket.Bucket).
			Err(err).Msg("Failed to serialize bucket metrics")
		return
	}

	if _, err := bucketMetrics.Put(key, metricsJSON); err != nil {
		log.Error().
			Str("bucket_id", bucket.Bucket).
			Err(err).Msg("Failed to store bucket metrics in KV")
	} else {
		log.Debug().
			Str("bucket_id", bucket.Bucket).
			Str("key", key).
			Msg("Bucket metrics stored in KV successfully")
	}
}
