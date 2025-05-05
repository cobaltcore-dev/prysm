// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and prysm contributors
//
// SPDX-License-Identifier: Apache-2.0

package radosgwusage

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/nats-io/nats.go"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog/log"
)

var (
	prysmTartgetUp = newGaugeVec("prysm_target_up", "Indicates if the exporter can reach the target (1 = up, 0 = down).", []string{})
	scrapeErrors   = newCounterVec("exporter_scrape_errors_total", "Total number of errors during scraping.", []string{})

	// User-level metrics
	userMetadata = newGaugeVec("radosgw_user_metadata", "User metadata", []string{"user", "display_name", "email", "storage_class", "rgw_cluster_id", "node", "instance_id"})

	userLabels        = []string{"user", "rgw_cluster_id", "node", "instance_id"}
	userBucketsTotal  = newGaugeVec("radosgw_user_buckets_total", "Total number of buckets for each user", userLabels)
	userObjectsTotal  = newGaugeVec("radosgw_user_objects_total", "Total number of objects for each user", userLabels)
	userDataSizeTotal = newGaugeVec("radosgw_user_data_size_bytes", "Total size of data for each user in bytes", userLabels)

	// User quota metrics
	userQuotaEnabled    = newGaugeVec("radosgw_usage_user_quota_enabled", "User quota enabled", userLabels)
	userQuotaMaxSize    = newGaugeVec("radosgw_usage_user_quota_size", "Maximum allowed size for user", userLabels)
	userQuotaMaxObjects = newGaugeVec("radosgw_usage_user_quota_size_objects", "Maximum allowed number of objects across all user buckets", userLabels)

	// Bucket-level metrics
	bucketLabels      = []string{"bucket", "owner", "zonegroup", "rgw_cluster_id", "node", "instance_id"}
	bucketSize        = newGaugeVec("radosgw_usage_bucket_size", "Size of bucket", bucketLabels)
	bucketObjectCount = newGaugeVec("radosgw_usage_bucket_objects", "Number of objects in bucket", bucketLabels)
	bucketShards      = newGaugeVec("radosgw_usage_bucket_shards", "Number of shards in bucket", bucketLabels)

	// Quota metrics
	bucketQuotaEnabled    = newGaugeVec("radosgw_usage_bucket_quota_enabled", "Quota enabled for bucket", bucketLabels)
	bucketQuotaMaxSize    = newGaugeVec("radosgw_usage_bucket_quota_size", "Maximum allowed bucket size", bucketLabels)
	bucketQuotaMaxObjects = newGaugeVec("radosgw_usage_bucket_quota_size_objects", "Maximum allowed bucket size in number of objects", bucketLabels)
)

func newCounterVec(name, help string, labels []string) *prometheus.CounterVec {
	return prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: name,
		Help: help,
	}, labels)
}

func newGaugeVec(name, help string, labels []string) *prometheus.GaugeVec {
	return prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: name,
		Help: help,
	}, labels)
}

func newHistogramVec(name, help string, labels []string) *prometheus.HistogramVec {
	return prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    name,
		Help:    help,
		Buckets: prometheus.DefBuckets,
	}, labels)
}

func init() {
	// Register all metrics with Prometheus's default registry
	prometheus.MustRegister(prysmTartgetUp, scrapeErrors)

	prometheus.MustRegister(userMetadata)
	prometheus.MustRegister(userBucketsTotal)
	prometheus.MustRegister(userObjectsTotal)
	prometheus.MustRegister(userDataSizeTotal)

	prometheus.MustRegister(userQuotaEnabled)
	prometheus.MustRegister(userQuotaMaxSize)
	prometheus.MustRegister(userQuotaMaxObjects)

	prometheus.MustRegister(bucketSize)
	prometheus.MustRegister(bucketObjectCount)
	prometheus.MustRegister(bucketShards)
	prometheus.MustRegister(bucketQuotaEnabled)
	prometheus.MustRegister(bucketQuotaMaxSize)
	prometheus.MustRegister(bucketQuotaMaxObjects)
}

func startPrometheusMetricsServer(port int) {
	go func() {
		http.Handle("/metrics", promhttp.Handler())
		log.Info().Msgf("starting prometheus metrics server on :%d", port)
		err := http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
		if err != nil {
			log.Fatal().Err(err).Msg("error starting prometheus metrics server")
		}
	}()
}

func populateStatus(status *PrysmStatus) {
	log.Trace().Msg("Starting to populate prysmStatus")
	// Safely get the current status snapshot
	up, errors := status.GetSnapshot()

	// Update Prometheus metrics
	prysmTartgetUp.With(prometheus.Labels{}).Set(up)
	scrapeErrors.With(prometheus.Labels{}).Add(float64(errors))
	log.Trace().Msg("Completed populating prysmStatus")
}

func populateMetricsFromKV(userMetrics, bucketMetrics, clusterMetrics nats.KeyValue, cfg RadosGWUsageConfig) {
	log.Info().Msg("Starting to populate metrics from KV")

	// Process user metrics
	populateUserMetricsFromKV(userMetrics, cfg)

	// Process bucket metrics
	populateBucketMetricsFromKV(bucketMetrics, cfg)

	log.Info().Msg("Completed populating metrics from KV")
}

func populateUserMetricsFromKV(userMetrics nats.KeyValue, cfg RadosGWUsageConfig) {
	keys, err := userMetrics.Keys()
	if err != nil {
		log.Error().Err(err).Msg("Failed to fetch keys from user metrics KV")
		return
	}

	for _, key := range keys {
		entry, err := userMetrics.Get(key)
		if err != nil {
			log.Warn().Str("key", key).Err(err).Msg("Failed to fetch user metric")
			continue
		}

		var metrics UserLevelMetrics
		if err := json.Unmarshal(entry.Value(), &metrics); err != nil {
			log.Warn().Str("key", key).Err(err).Msg("Failed to unmarshal user metric")
			continue
		}

		userMetadata.With(prometheus.Labels{
			"user":           metrics.GetUserIdentification(),
			"display_name":   metrics.DisplayName,
			"email":          metrics.Email,
			"storage_class":  metrics.DefaultStorageClass,
			"rgw_cluster_id": cfg.ClusterID,
			"node":           cfg.NodeName,
			"instance_id":    cfg.InstanceID,
		}).Set(1)

		labels := prometheus.Labels{
			"user":           metrics.GetUserIdentification(),
			"rgw_cluster_id": cfg.ClusterID,
			"node":           cfg.NodeName,
			"instance_id":    cfg.InstanceID,
		}

		userBucketsTotal.With(labels).Set(float64(metrics.BucketsTotal))
		userObjectsTotal.With(labels).Set(float64(metrics.ObjectsTotal))
		userDataSizeTotal.With(labels).Set(float64(metrics.DataSizeTotal))

		// User quota metrics
		userQuotaEnabled.With(labels).Set(boolToFloat64(&metrics.UserQuotaEnabled))
		if metrics.UserQuotaMaxSize != nil && *metrics.UserQuotaMaxSize > 0 {
			userQuotaMaxSize.With(labels).Set(float64(*metrics.UserQuotaMaxSize))
		}
		if metrics.UserQuotaMaxObjects != nil && *metrics.UserQuotaMaxObjects > 0 {
			userQuotaMaxObjects.With(labels).Set(float64(*metrics.UserQuotaMaxObjects))
		}
	}
}

func populateBucketMetricsFromKV(bucketMetrics nats.KeyValue, cfg RadosGWUsageConfig) {
	keys, err := bucketMetrics.Keys()
	if err != nil {
		log.Error().Err(err).Msg("Failed to fetch keys from bucket metrics KV")
		return
	}

	for _, key := range keys {
		entry, err := bucketMetrics.Get(key)
		if err != nil {
			log.Warn().Str("key", key).Err(err).Msg("Failed to fetch bucket metric")
			continue
		}

		var metrics UserBucketMetrics
		if err := json.Unmarshal(entry.Value(), &metrics); err != nil {
			log.Warn().Str("key", key).Err(err).Msg("Failed to unmarshal bucket metric")
			continue
		}

		labels := prometheus.Labels{
			"bucket":         metrics.BucketID,
			"owner":          metrics.GetUserIdentification(),
			"zonegroup":      metrics.Zonegroup,
			"rgw_cluster_id": cfg.ClusterID,
			"node":           cfg.NodeName,
			"instance_id":    cfg.InstanceID,
		}

		bucketSize.With(labels).Set(float64(metrics.BucketSize))
		bucketObjectCount.With(labels).Set(float64(metrics.ObjectCount))

		if metrics.NumShards != nil {
			bucketShards.With(labels).Set(float64(*metrics.NumShards))
		}

		// Set quota information
		bucketQuotaEnabled.With(labels).Set(boolToFloat64(&metrics.QuotaEnabled))
		if metrics.QuotaMaxSize != nil && *metrics.QuotaMaxSize > 0 {
			bucketQuotaMaxSize.With(labels).Set(float64(*metrics.QuotaMaxSize))
		}
		if metrics.QuotaMaxObjects != nil && *metrics.QuotaMaxObjects > 0 {
			bucketQuotaMaxObjects.With(labels).Set(float64(*metrics.QuotaMaxObjects))
		}
	}
}

func boolToFloat64(b *bool) float64 {
	if b != nil && *b {
		return 1.0
	}
	return 0.0
}
