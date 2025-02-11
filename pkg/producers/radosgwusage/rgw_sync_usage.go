// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and prysm contributors
//
// SPDX-License-Identifier: Apache-2.0
package radosgwusage

import (
	"context"
	"encoding/json"
	"fmt"
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

func syncUsage(userUsageData nats.KeyValue, cfg RadosGWUsageConfig, status *PrysmStatus) error {
	log.Info().Msg("Starting usage sync process")

	// Create a new RadosGW admin client.
	co, err := createRadosGWClient(cfg, status)
	if err != nil {
		log.Error().Err(err).Msg("Failed to create RadosGW admin client")
		return err
	}

	// Fetch global usage (for all users).
	usage, err := fetchUserUsageGlobal(co)
	if err != nil {
		log.Error().Err(err).Msg("Failed to fetch global user usage")
		return err
	}

	// Store the full usage data in the KV store.
	err = storeUserUsageInKV(usage, userUsageData)
	if err != nil {
		log.Error().Err(err).Msg("Failed to store user usage in KV")
		return err
	}

	log.Info().Msg("Usage sync process completed")
	return nil
}

func fetchUserUsageGlobal(co *admin.API) (admin.Usage, error) {
	var aggregatedUsage admin.Usage

	// For global sync, we do not pass a specific userID and no start time.
	usageRequest := admin.Usage{
		UserID: "",
		Start:  "",
	}
	showEntries := true
	usageRequest.ShowEntries = &showEntries

	// Fetch the initial global usage data.
	globalUsage, err := co.GetUsage(context.Background(), usageRequest)
	if err != nil {
		return admin.Usage{}, fmt.Errorf("failed to fetch global usage: %w", err)
	}

	// Aggregate the initial data.
	aggregatedUsage.Entries = append(aggregatedUsage.Entries, globalUsage.Entries...)
	aggregatedUsage.Summary = append(aggregatedUsage.Summary, globalUsage.Summary...)

	// For each user in the global usage, fetch detailed usage data.
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
				Msg("Failed to fetch detailed usage for user; using global snapshot")
			continue
		}

		// If detailed usage is available, replace the global entry.
		if len(detailedUsage.Entries) > 0 {
			aggregatedUsage.Entries[idx] = detailedUsage.Entries[0]
		}
	}

	return aggregatedUsage, nil
}

func storeUserUsageInKV(userUsage admin.Usage, userUsageData nats.KeyValue) error {
	bucketsFailed := 0
	skippedBuckets := 0

	// Iterate through each user entry and store usage for each bucket
	for _, entry := range userUsage.Entries {
		for _, bucket := range entry.Buckets {
			bucketName := bucket.Bucket
			if bucketName == "" {
				bucketName = rootBucketPlaceholder // Placeholder for root/unnamed buckets
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

			user, tenant := SplitUserTenant(entry.User)
			bucketKey := BuildUserTenantBucketKey(user, tenant, bucketName)

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
