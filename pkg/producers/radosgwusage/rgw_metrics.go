// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and prysm contributors
//
// SPDX-License-Identifier: Apache-2.0

package radosgwusage

import (
	"time"

	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog/log"
)

func updateMetricsPeriodically(
	cfg RadosGWUsageConfig,
	syncControl, userData, userUsageData, bucketData, userMetrics, bucketMetrics, clusterMetrics nats.KeyValue,
) {

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		log.Info().Msg("Updating metrics for Prometheus")

		// Ensure no sync processes are in progress
		if !areAllFlagsUnset(syncControl, []string{
			"sync_users_in_progress",
			"sync_buckets_in_progress",
			"sync_usages_in_progress",
		}) {
			log.Debug().Msg("Skipping metric calculation; data sync in progress")
			continue
		}

		setFlag(syncControl, "metric_calc_in_progress", true)
		log.Debug().Msg("Starting metrics update")

		// Placeholder for user metrics calculation
		if err := updateUserMetricsInKV(userData, userUsageData, bucketData, userMetrics); err != nil {
			log.Error().Err(err).Msg("Error updating user metrics")
		}

		// Placeholder for bucket metrics calculation
		if err := updateBucketMetricsInKV(bucketData, userUsageData, bucketMetrics); err != nil {
			log.Error().Err(err).Msg("Error updating bucket metrics")
		}

		// Placeholder for cluster metrics calculation
		if err := updateClusterMetricsInKV(userMetrics, bucketMetrics, clusterMetrics); err != nil {
			log.Error().Err(err).Msg("Error updating cluster metrics")
		}

		setFlag(syncControl, "metric_calc_in_progress", false)
		log.Info().Msg("Metrics update completed")
	}
}

func populateMetricsPeriodically(
	cfg RadosGWUsageConfig,
	syncControl, userMetrics, bucketMetrics, clusterMetrics nats.KeyValue,
	status *PrysmStatus,
) {

	statusTicker := time.NewTicker(10 * time.Second)
	metricTicker := time.NewTicker(1 * time.Minute)
	defer statusTicker.Stop()
	defer metricTicker.Stop()

	for {
		select {
		case <-statusTicker.C:
			populateStatus(status)
		case <-metricTicker.C:
			// Ensure metrics are fully calculated
			if isFlagSet(syncControl, "metric_calc_in_progress") {
				log.Debug().Msg("Skipping Prometheus metric population; metrics calculation in progress")
				continue
			}

			log.Info().Msg("Populate metrics for Prometheus")
			populateMetricsFromKV(userMetrics, bucketMetrics, clusterMetrics, cfg)
			log.Info().Msg("Metrics update completed")
		}
	}
}
