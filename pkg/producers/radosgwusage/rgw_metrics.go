// Copyright 2025 Clyso GmbH
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
