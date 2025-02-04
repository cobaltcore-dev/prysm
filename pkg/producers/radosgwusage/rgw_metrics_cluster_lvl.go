// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and prysm contributors
//
// SPDX-License-Identifier: Apache-2.0

package radosgwusage

import (
	"encoding/json"
	"fmt"

	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog/log"
)

type ClusterLevelMetrics struct {
	TotalReadOPs       uint64 // Collects cumulative read requests. Makes sense for analyzing overall read access patterns.
	TotalWriteOPs      uint64 // Collects cumulative write requests. Useful for understanding total write volume over time.
	TotalOPs           uint64 // Sum of reads and writes, useful for total activity tracking.
	BytesSentTotal     uint64 // Total Data Downloaded
	BytesReceivedTotal uint64 // Total Data Uploaded
	Throughput         uint64 // Throughput gives insight into how efficiently data is being processed across the cluster.
}

func updateClusterMetricsInKV(userMetrics, bucketMetrics, clusterMetrics nats.KeyValue) error {
	log.Debug().Msg("Starting cluster-level metrics aggregation")

	// Initialize cluster-level metrics
	metrics := ClusterLevelMetrics{
		TotalReadOPs:  0,
		TotalWriteOPs: 0,
		TotalOPs:      0,
		Throughput:    0,
	}

	// Aggregate metrics from user metrics
	userKeys, err := userMetrics.Keys()
	if err != nil {
		log.Error().Err(err).Msg("Failed to fetch keys from user metrics")
		return fmt.Errorf("failed to fetch keys from user metrics: %w", err)
	}

	for _, userKey := range userKeys {
		entry, err := userMetrics.Get(userKey)
		if err != nil {
			log.Warn().
				Str("key", userKey).
				Err(err).
				Msg("Failed to fetch user metric data from KV")
			continue
		}

		var userMetricsData UserLevelMetrics
		if err := json.Unmarshal(entry.Value(), &userMetricsData); err != nil {
			log.Warn().
				Str("key", userKey).
				Err(err).
				Msg("Failed to unmarshal user metric data")
			continue
		}

		// Aggregate user-level metrics into cluster-level metrics
		metrics.TotalReadOPs += userMetricsData.TotalReadOPs
		metrics.TotalWriteOPs += userMetricsData.TotalWriteOPs
		metrics.TotalOPs += userMetricsData.TotalOPs
		metrics.BytesSentTotal += userMetricsData.BytesSentTotal
		metrics.BytesReceivedTotal += userMetricsData.BytesReceivedTotal
		metrics.Throughput += userMetricsData.ThroughputBytesTotal
	}

	// Aggregate metrics from bucket metrics
	bucketKeys, err := bucketMetrics.Keys()
	if err != nil {
		log.Error().Err(err).Msg("Failed to fetch keys from bucket metrics")
		return fmt.Errorf("failed to fetch keys from bucket metrics: %w", err)
	}

	for _, bucketKey := range bucketKeys {
		entry, err := bucketMetrics.Get(bucketKey)
		if err != nil {
			log.Warn().
				Str("key", bucketKey).
				Err(err).
				Msg("Failed to fetch bucket metric data from KV")
			continue
		}

		var bucketMetricsData UserBucketMetrics
		if err := json.Unmarshal(entry.Value(), &bucketMetricsData); err != nil {
			log.Warn().
				Str("key", bucketKey).
				Err(err).
				Msg("Failed to unmarshal bucket metric data")
			continue
		}

		// Aggregate bucket-level metrics into cluster-level metrics
		metrics.TotalReadOPs += bucketMetricsData.TotalReadOPs
		metrics.TotalWriteOPs += bucketMetricsData.TotalWriteOPs
		metrics.TotalOPs += bucketMetricsData.TotalOPs
		metrics.BytesSentTotal += bucketMetricsData.BytesSentTotal
		metrics.BytesReceivedTotal += bucketMetricsData.BytesReceivedTotal
		metrics.Throughput += bucketMetricsData.BytesSentTotal + bucketMetricsData.BytesReceivedTotal
	}

	// Prepare the metrics key
	metricsKey := "cluster_metrics"

	// Serialize and store metrics
	metricsData, err := json.Marshal(metrics)
	if err != nil {
		log.Error().
			Err(err).
			Msg("Failed to serialize cluster metrics")
		return fmt.Errorf("failed to serialize cluster metrics: %w", err)
	}

	if _, err := clusterMetrics.Put(metricsKey, metricsData); err != nil {
		log.Error().
			Err(err).
			Msg("Failed to store cluster metrics in KV")
		return fmt.Errorf("failed to store cluster metrics in KV: %w", err)
	}

	log.Debug().
		Str("key", metricsKey).
		Msg("Cluster metrics stored in KV successfully")

	log.Info().Msg("Completed cluster metrics aggregation and storage")
	return nil
}
