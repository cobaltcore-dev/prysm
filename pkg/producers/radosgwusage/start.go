// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and prysm contributors
//
// SPDX-License-Identifier: Apache-2.0

package radosgwusage

import (
	"fmt"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog/log"
)

// StartRadosGWUsageExporter starts the process of exporting RadosGW usage metrics.
// It supports exporting to Prometheus, NATS, or stdout and includes sync control using NATS-KV.
func StartRadosGWUsageExporter(cfg RadosGWUsageConfig) {
	// Initialize Prometheus server if enabled
	if cfg.Prometheus {
		go startPrometheusMetricsServer(cfg.PrometheusPort)
	}

	// Initialize NATS connection for metrics export
	var natsConn *nats.Conn
	if cfg.UseNats {
		var err error
		natsConn, err = nats.Connect(cfg.NatsURL)
		if err != nil {
			log.Fatal().Err(err).Msg("error connecting to NATS")
		}
		defer natsConn.Close()
	}

	// Initialize NATS-KVs for sync control (if enabled)
	var kvStores map[string]nats.KeyValue
	var err error
	if cfg.SyncControlNats {
		kvStores, err = initializeKeyValueStores(cfg)
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to initialize Key-Value stores")
		}
	}

	// Start the metric collection loop
	startMetricCollectionLoop(cfg, natsConn, kvStores)
}

func initializeKeyValueStores(cfg RadosGWUsageConfig) (map[string]nats.KeyValue, error) {
	var natsConn *nats.Conn
	var err error

	// Start NATS based on configuration
	if cfg.SyncExternalNats {
		// Connect to external NATS server
		natsConn, err = nats.Connect(cfg.SyncControlURL)
	} else {
		// Start embedded NATS server
		embeddedServer := startEmbeddedNATSServer()
		natsConn, err = nats.Connect(embeddedServer.ClientURL())
	}

	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}

	// Initialize JetStream
	js, err := natsConn.JetStream()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize JetStream: %w", err)
	}

	// Define the buckets we need
	bucketNames := []string{
		fmt.Sprintf("%s_sync_control", cfg.SyncControlBucketPrefix),    // Sync control
		fmt.Sprintf("%s_user_data", cfg.SyncControlBucketPrefix),       // User information
		fmt.Sprintf("%s_user_usage_data", cfg.SyncControlBucketPrefix), // User Usage information
		fmt.Sprintf("%s_bucket_data", cfg.SyncControlBucketPrefix),     // Bucket information
		fmt.Sprintf("%s_user_metrics", cfg.SyncControlBucketPrefix),    // User metrics
		fmt.Sprintf("%s_bucket_metrics", cfg.SyncControlBucketPrefix),  // Bucket metrics
		fmt.Sprintf("%s_cluster_metrics", cfg.SyncControlBucketPrefix), // Cluster metrics
	}

	// Map to store Key-Value handles
	kvStores := make(map[string]nats.KeyValue)

	// Create or access each bucket
	for _, bucketName := range bucketNames {
		kv, err := js.KeyValue(bucketName)
		if err != nil {
			kv, err = js.CreateKeyValue(&nats.KeyValueConfig{
				Bucket: bucketName,
			})
			if err != nil {
				return nil, fmt.Errorf("failed to create/access bucket %s: %w", bucketName, err)
			}
		}
		kvStores[bucketName] = kv
	}

	return kvStores, nil
}

// startEmbeddedNATSServer initializes and starts an embedded NATS server.
func startEmbeddedNATSServer() *server.Server {
	opts := &server.Options{
		Port:      -1, // Automatically choose an available port
		JetStream: true,
		StoreDir:  "/tmp/nats",
	}

	natsServer, err := server.NewServer(opts)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create embedded NATS server")
	}

	go natsServer.Start()
	if !natsServer.ReadyForConnections(10 * time.Second) {
		log.Fatal().Err(err).Msg("embedded NATS server did not start in time")
	}

	log.Info().Str("url", natsServer.ClientURL()).Msg("embedded NATS server running")
	return natsServer
}

// func collectAndExportMetrics(cfg RadosGWUsageConfig, natsConn *nats.Conn, kv nats.KeyValue) {
// 	startTime := time.Now()

// 	// Collect usage metrics
// 	entries, err := collectUsageMetrics(cfg, startTime)
// 	if err != nil {
// 		log.Error().Err(err).Msg("error collecting usage metrics")
// 		return
// 	}

// 	duration := time.Since(startTime).Seconds()

// 	// Publish to Prometheus if enabled
// 	if cfg.Prometheus {
// 		publishToPrometheus(entries, duration, cfg)
// 	}

// 	// Publish to NATS if enabled
// 	if cfg.UseNats {
// 		publishToNATS(natsConn, cfg.NatsSubject, entries)
// 	} else {
// 		// Output to stdout if neither Prometheus nor NATS is used
// 		entriesJSON, err := json.MarshalIndent(entries, "", "  ")
// 		if err != nil {
// 			log.Error().Err(err).Msg("error marshalling entries to JSON")
// 			return
// 		}
// 		fmt.Println(string(entriesJSON))
// 		log.Trace().Msg(string(entriesJSON))
// 	}

// 	// Handle sync control flags if NATS-KV is enabled
// 	if kv != nil {
// 		processSyncControlFlags(kv)
// 	}
// }

func startMetricCollectionLoop(cfg RadosGWUsageConfig, natsConn *nats.Conn, kvStores map[string]nats.KeyValue) {

	var wg sync.WaitGroup

	// Initialize thread-safe status
	status := &PrysmStatus{}

	// Ensure required buckets are available
	syncControl, ok := kvStores[fmt.Sprintf("%s_sync_control", cfg.SyncControlBucketPrefix)]
	if !ok {
		log.Fatal().Msg("sync_control bucket not found in Key-Value stores")
	}
	userData, ok := kvStores[fmt.Sprintf("%s_user_data", cfg.SyncControlBucketPrefix)]
	if !ok {
		log.Fatal().Msg("user_data bucket not found in Key-Value stores")
	}
	userUsageData, ok := kvStores[fmt.Sprintf("%s_user_usage_data", cfg.SyncControlBucketPrefix)]
	if !ok {
		log.Fatal().Msg("user_usage_data bucket not found in Key-Value stores")
	}
	bucketData, ok := kvStores[fmt.Sprintf("%s_bucket_data", cfg.SyncControlBucketPrefix)]
	if !ok {
		log.Fatal().Msg("bucket_data bucket not found in Key-Value stores")
	}
	// metrics
	userMetrics, ok := kvStores[fmt.Sprintf("%s_user_metrics", cfg.SyncControlBucketPrefix)]
	if !ok {
		log.Fatal().Msg("user_metrics bucket not found in Key-Value stores")
	}
	bucketMetrics, ok := kvStores[fmt.Sprintf("%s_bucket_metrics", cfg.SyncControlBucketPrefix)]
	if !ok {
		log.Fatal().Msg("bucket_metrics bucket not found in Key-Value stores")
	}
	clusterMetrics, ok := kvStores[fmt.Sprintf("%s_cluster_metrics", cfg.SyncControlBucketPrefix)]
	if !ok {
		log.Fatal().Msg("user_metrics bucket not found in Key-Value stores")
	}

	// Cleanup
	syncControl.Delete("sync_users")
	syncControl.Delete("sync_users_in_progress")
	syncControl.Delete("sync_usages")
	syncControl.Delete("sync_usages_in_progress")
	syncControl.Delete("sync_buckets")
	syncControl.Delete("sync_buckets_in_progress")
	syncControl.Delete("metric_calc_in_progress")

	// Launch goroutines for refreshing users, usages, and buckets
	wg.Add(1)
	go func() {
		defer wg.Done()
		checkAndRefreshUsers(syncControl, userData, cfg, status)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		checkAndRefreshUserUsages(syncControl, userUsageData, userData, cfg, status)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		checkAndRefreshBuckets(syncControl, bucketData, cfg, status)
	}()

	// Launch a goroutine to update metrics periodically
	wg.Add(1)
	go func() {
		defer wg.Done()
		updateMetricsPeriodically(cfg, syncControl, userData, userUsageData, bucketData, userMetrics, bucketMetrics, clusterMetrics)
	}()

	// Launch a goroutine to populate metrics periodically
	if cfg.Prometheus {
		wg.Add(1)
		go func() {
			defer wg.Done()
			populateMetricsPeriodically(
				cfg, syncControl, userMetrics, bucketMetrics, clusterMetrics, status,
			)
		}()
	}

	// Wait for termination signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	log.Info().Msg("Metric collection loop started. Waiting for termination signal.")
	<-sigChan
	log.Info().Msg("Termination signal received. Exiting...")

	// Wait for all goroutines to finish
	wg.Wait()
	log.Info().Msg("All tasks completed. Exiting.")
}

func processSyncControlFlags(kv nats.KeyValue) bool {
	keys, err := kv.Keys()
	if err != nil {
		log.Error().Err(err).Msg("error fetching sync control keys")
		return false
	}

	processed := false
	for _, key := range keys {
		value, err := kv.Get(key)
		if err != nil || value == nil {
			continue
		}

		needsSync := string(value.Value()) == "true"
		if needsSync {
			parts := strings.Split(key, ":")
			if len(parts) != 2 {
				log.Warn().Msgf("invalid sync key format: %s", key)
				continue
			}

			// tenantID := parts[0]
			// accountID := parts[1]

			// Synchronize data for this tenant and account
			// if err := syncDataForAccount(tenantID, accountID); err == nil {
			// 	_ = kv.Delete(key) // Reset the sync flag
			// 	processed = true
			// }
		}
	}

	return processed
}

func startSyncListener(natsConn *nats.Conn, syncControl nats.KeyValue) error {
	if natsConn == nil || natsConn.Status() != nats.CONNECTED {
		return fmt.Errorf("invalid or uninitialized NATS connection")
	}

	// User Sync Listener
	_, err := natsConn.Subscribe("rgw-usage.sync.user", func(msg *nats.Msg) {
		userID := string(msg.Data)
		if userID == "" {
			log.Warn().Msg("Received empty user ID on rgw-usage.sync.user subject")
			return
		}

		log.Info().Str("user", userID).Msg("Received sync request for user")
		if err := triggerUserSync(syncControl, userID); err != nil {
			log.Error().Str("user", userID).Err(err).Msg("Failed to trigger user sync")
		}
	})
	if err != nil {
		return fmt.Errorf("failed to subscribe to rgw-usage.sync.user: %w", err)
	}

	// Bucket Sync Listener
	_, err = natsConn.Subscribe("rgw-usage.sync.bucket", func(msg *nats.Msg) {
		bucketName := string(msg.Data)
		if bucketName == "" {
			log.Warn().Msg("Received empty bucket name on rgw-usage.sync.bucket subject")
			return
		}

		log.Info().Str("bucket", bucketName).Msg("Received sync request for bucket")
		if err := triggerBucketSync(syncControl, bucketName); err != nil {
			log.Error().Str("bucket", bucketName).Err(err).Msg("Failed to trigger bucket sync")
		}
	})
	if err != nil {
		return fmt.Errorf("failed to subscribe to rgw-usage.sync.bucket: %w", err)
	}

	log.Info().Msg("NATS sync listeners started successfully")
	return nil
}

func setFlag(syncControl nats.KeyValue, key string, value bool) {
	if value {
		if _, err := syncControl.Put(key, []byte("true")); err != nil {
			log.Error().Err(err).Str("key", key).Msg("Failed to set flag")
		}
	} else {
		if err := syncControl.Delete(key); err != nil {
			log.Warn().Err(err).Str("key", key).Msg("Failed to clear flag")
		}
	}
}

func areAllFlagsUnset(syncControl nats.KeyValue, keys []string) bool {
	for _, key := range keys {
		if isFlagSet(syncControl, key) {
			return false
		}
	}
	return true
}

func isFlagSet(syncControl nats.KeyValue, key string) bool {
	_, err := syncControl.Get(key)
	return err == nil
}
