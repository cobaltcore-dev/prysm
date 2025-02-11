// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and prysm contributors
//
// SPDX-License-Identifier: Apache-2.0

package radosgwusage

import (
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog/log"
)

var (
	taskMutex      sync.Mutex
	taskInProgress = make(map[string]bool)
)

func isTaskRunning(taskName string) bool {
	taskMutex.Lock()
	defer taskMutex.Unlock()
	return taskInProgress[taskName]
}

func setTaskRunning(taskName string, running bool) {
	taskMutex.Lock()
	defer taskMutex.Unlock()
	taskInProgress[taskName] = running
}

// StartRadosGWUsageExporter starts the process of exporting RadosGW usage metrics.
// It supports exporting to Prometheus, NATS, or stdout and includes sync control using NATS-KV.
func StartRadosGWUsageExporter(cfg RadosGWUsageConfig) {
	// Initialize Prometheus server if enabled
	if cfg.Prometheus {
		go startPrometheusMetricsServer(cfg.PrometheusPort)
	}
	var err error

	// Initialize NATS connection for metrics export
	var exportNatsConn *nats.Conn
	if cfg.UseNats {
		exportNatsConn, err = nats.Connect(cfg.NatsURL)
		if err != nil {
			log.Fatal().Err(err).Msg("error connecting to NATS")
		}
		defer exportNatsConn.Close()
	}

	var natsServer *server.Server
	var nc *nats.Conn
	var js nats.JetStreamContext
	// Start NATS based on configuration
	if cfg.SyncExternalNats {
		nc, err := nats.Connect(cfg.SyncControlURL)
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to connect to external NATS")
		}
		js, err = nc.JetStream()
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to initialize JetStream for external NATS")
		}
	} else {
		natsServer, nc, js, err = startEmbeddedNATS()
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to start embedded NATS")
		}
		defer natsServer.Shutdown()
	}
	defer nc.Close()

	// Initialize NATS-KVs for sync control (if enabled)
	var kvStores map[string]nats.KeyValue
	if cfg.SyncControlNats {
		kvStores, err = initializeKeyValueStores(cfg, js)
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to initialize Key-Value stores")
		}
	}

	// Start the metric collection loop
	startMetricCollectionLoop(cfg, exportNatsConn, nc, js, kvStores)
}

// Start embedded NATS with JetStream
func startEmbeddedNATS() (*server.Server, *nats.Conn, nats.JetStreamContext, error) {
	opts := &server.Options{
		JetStream: true,
		StoreDir:  "/tmp/nats", // Ensure this directory exists
	}

	s, err := server.NewServer(opts)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create NATS server: %w", err)
	}

	// Run NATS in a goroutine
	go s.Start()

	if !s.ReadyForConnections(10 * time.Second) {
		return nil, nil, nil, fmt.Errorf("NATS Server did not start in time")
	}

	// Connect to the embedded NATS server
	nc, err := nats.Connect(s.ClientURL())
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}

	// Initialize JetStream
	js, err := nc.JetStream()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to initialize JetStream: %w", err)
	}

	return s, nc, js, nil
}

func initializeKeyValueStores(cfg RadosGWUsageConfig, js nats.JetStreamContext) (map[string]nats.KeyValue, error) {
	// Define the buckets we need
	bucketNames := []string{
		// fmt.Sprintf("%s_sync_control", cfg.SyncControlBucketPrefix),    // Sync control
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

func ensureKeyValueStores(cfg RadosGWUsageConfig, kvStores map[string]nats.KeyValue) (userData, userUsageData, bucketData, userMetrics, bucketMetrics, clusterMetrics nats.KeyValue) {
	// Ensure required buckets are available
	userData, ok := kvStores[fmt.Sprintf("%s_user_data", cfg.SyncControlBucketPrefix)]
	if !ok {
		log.Fatal().Msg("user_data bucket not found in Key-Value stores")
	}
	userUsageData, ok = kvStores[fmt.Sprintf("%s_user_usage_data", cfg.SyncControlBucketPrefix)]
	if !ok {
		log.Fatal().Msg("user_usage_data bucket not found in Key-Value stores")
	}
	bucketData, ok = kvStores[fmt.Sprintf("%s_bucket_data", cfg.SyncControlBucketPrefix)]
	if !ok {
		log.Fatal().Msg("bucket_data bucket not found in Key-Value stores")
	}
	// metrics
	userMetrics, ok = kvStores[fmt.Sprintf("%s_user_metrics", cfg.SyncControlBucketPrefix)]
	if !ok {
		log.Fatal().Msg("user_metrics bucket not found in Key-Value stores")
	}
	bucketMetrics, ok = kvStores[fmt.Sprintf("%s_bucket_metrics", cfg.SyncControlBucketPrefix)]
	if !ok {
		log.Fatal().Msg("bucket_metrics bucket not found in Key-Value stores")
	}
	clusterMetrics, ok = kvStores[fmt.Sprintf("%s_cluster_metrics", cfg.SyncControlBucketPrefix)]
	if !ok {
		log.Fatal().Msg("user_metrics bucket not found in Key-Value stores")
	}
	return userData, userUsageData, bucketData, userMetrics, bucketMetrics, clusterMetrics
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

func startMetricCollectionLoop(cfg RadosGWUsageConfig, exportNatsConn *nats.Conn, nc *nats.Conn, js nats.JetStreamContext, kvStores map[string]nats.KeyValue) {

	var wg sync.WaitGroup

	// Initialize thread-safe status
	prysmStatus := &PrysmStatus{}

	js, err := nc.JetStream()
	if err != nil {
		log.Fatal().Msg("Failed to initialize JetStream")
	}

	// Ensure the stream exists
	if err := ensureStream(js, "notifications"); err != nil {
		log.Fatal().Msg("Failed to setup notification stream")
	}

	userData, userUsageData, bucketData, userMetrics, bucketMetrics, clusterMetrics := ensureKeyValueStores(cfg, kvStores)

	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			syncUsers(userData, cfg, prysmStatus)
			syncBuckets(bucketData, cfg, prysmStatus)
			syncUsage(userUsageData, cfg, prysmStatus)
			updateUserMetricsInKV(userData, userUsageData, bucketData, userMetrics)
			updateBucketMetricsInKV(bucketData, userUsageData, bucketMetrics)
			updateClusterMetricsInKV(userMetrics, bucketMetrics, clusterMetrics)
			if cfg.Prometheus {
				populateStatus(prysmStatus)
				populateMetricsFromKV(userMetrics, bucketMetrics, clusterMetrics, cfg)
			}
		}
	}()

	// sub, err := js.QueueSubscribe("notifications", "worker-group", func(msg *nats.Msg) {
	// 	_ = msg.Ack() //FIXME
	// 	var event map[string]interface{}
	// 	if err := json.Unmarshal(msg.Data, &event); err != nil {
	// 		log.Error().Err(err).Msg("Failed to parse event")
	// 		_ = msg.Ack() // ACK to prevent infinite loop
	// 		return
	// 	}
	// 	eventType := event["event"].(string)
	// 	status := event["status"].(string)

	// 	switch eventType {
	// 	case "sync_users":
	// 		if status == "in_progress" {
	// 			if isTaskRunning("sync_users") {
	// 				log.Debug().Msg("sync_users already in progress; skipping this event")
	// 				_ = msg.Ack()
	// 				return
	// 			}
	// 			setTaskRunning("sync_users", true)

	// 			// Start a goroutine to periodically extend the ack deadline.
	// 			done := make(chan struct{})
	// 			go func() {
	// 				ticker := time.NewTicker(30 * time.Second) // extend every 30 seconds
	// 				defer ticker.Stop()
	// 				for {
	// 					select {
	// 					case <-ticker.C:
	// 						// Extend the ack deadline by calling InProgress without a specific timeout,
	// 						// or you can specify a duration if needed.
	// 						msg.InProgress(nats.AckWait(30 * time.Second))
	// 					case <-done:
	// 						return
	// 					}
	// 				}
	// 			}()
	// 			syncUsers(userData, cfg, prysmStatus, nc)
	// 			// Signal that processing is done so the ticker can stop.
	// 			close(done)

	// 			// Publish event: Users sync completed
	// 			publishEvent(nc, "sync_users", "completed", nil, nil)
	// 			// Publish next event: Start bucket synchronization
	// 			publishEvent(nc, "sync_buckets", "in_progress", nil, nil)

	// 			// Acknowledge the message.
	// 			if err := msg.Ack(); err != nil {
	// 				log.Error().Err(err).Msg("Failed to acknowledge sync_usage message")
	// 			}

	// 			setTaskRunning("sync_users", false)
	// 		}
	// 	case "sync_buckets":
	// 		if status == "in_progress" {
	// 			if isTaskRunning("sync_buckets") {
	// 				log.Debug().Msg("sync_buckets already in progress; skipping this event")
	// 				_ = msg.Ack()
	// 				return
	// 			}
	// 			setTaskRunning("sync_buckets", true)

	// 			// Start a goroutine to periodically extend the ack deadline.
	// 			done := make(chan struct{})
	// 			go func() {
	// 				ticker := time.NewTicker(30 * time.Second) // extend every 30 seconds
	// 				defer ticker.Stop()
	// 				for {
	// 					select {
	// 					case <-ticker.C:
	// 						// Extend the ack deadline by calling InProgress without a specific timeout,
	// 						// or you can specify a duration if needed.
	// 						msg.InProgress(nats.AckWait(30 * time.Second))
	// 					case <-done:
	// 						return
	// 					}
	// 				}
	// 			}()
	// 			err = syncBuckets(bucketData, cfg, prysmStatus)
	// 			if err != nil{
	// 			 	publishEvent(nc, "sync_buckets", "failed", nil, nil)
	// 			}
	// 			// Signal that processing is done so the ticker can stop.
	// 			close(done)

	// 			// Notify that sync is completed
	// 			publishEvent(nc, "sync_buckets", "completed", nil, nil)
	// 			// Publish next event: Start usage synchronization
	// 			publishEvent(nc, "sync_usage", "in_progress", nil, nil)

	// 			// Acknowledge the message.
	// 			if err := msg.Ack(); err != nil {
	// 				log.Error().Err(err).Msg("Failed to acknowledge sync_usage message")
	// 			}

	// 			setTaskRunning("sync_buckets", false)
	// 		}
	// 	case "sync_usage":
	// 		if status == "in_progress" {
	// 			if isTaskRunning("sync_usage") {
	// 				log.Debug().Msg("sync_usage already in progress; skipping this event")
	// 				_ = msg.Ack()
	// 				return
	// 			}
	// 			setTaskRunning("sync_usage", true)

	// 			// Start a goroutine to periodically extend the ack deadline.
	// 			done := make(chan struct{})
	// 			go func() {
	// 				ticker := time.NewTicker(30 * time.Second) // extend every 30 seconds
	// 				defer ticker.Stop()
	// 				for {
	// 					select {
	// 					case <-ticker.C:
	// 						// Extend the ack deadline by calling InProgress without a specific timeout,
	// 						// or you can specify a duration if needed.
	// 						msg.InProgress(nats.AckWait(30 * time.Second))
	// 					case <-done:
	// 						return
	// 					}
	// 				}
	// 			}()
	// 			// Perform the long-running sync process.
	// 			syncUsage(userUsageData, cfg, prysmStatus, nc)
	// 			// Signal that processing is done so the ticker can stop.
	// 			close(done)

	// 			// Publish a notification that sync is completed.
	// 			publishEvent(nc, "sync_usage", "completed", nil, nil)
	// 			// Publish next event: Start metric generation
	// 			publishEvent(nc, "generate_metrics", "in_progress", nil, nil)

	// 			// Acknowledge the message.
	// 			if err := msg.Ack(); err != nil {
	// 				log.Error().Err(err).Msg("Failed to acknowledge sync_usage message")
	// 			}

	// 			setTaskRunning("sync_usage", false)
	// 		}
	// 	case "generate_metrics":
	// 		if status == "in_progress" {
	// 			if isTaskRunning("generate_metrics") {
	// 				log.Debug().Msg("generate_metrics already in progress; skipping this event")
	// 				_ = msg.Ack()
	// 				return
	// 			}
	// 			setTaskRunning("generate_metrics", true)

	// 			// Start a goroutine to periodically extend the ack deadline.
	// 			done := make(chan struct{})
	// 			go func() {
	// 				ticker := time.NewTicker(30 * time.Second) // extend every 30 seconds
	// 				defer ticker.Stop()
	// 				for {
	// 					select {
	// 					case <-ticker.C:
	// 						// Extend the ack deadline by calling InProgress without a specific timeout,
	// 						// or you can specify a duration if needed.
	// 						msg.InProgress(nats.AckWait(30 * time.Second))
	// 					case <-done:
	// 						return
	// 					}
	// 				}
	// 			}()
	// 			updateUserMetricsInKV(userData, userUsageData, bucketData, userMetrics)
	// 			updateBucketMetricsInKV(bucketData, userUsageData, bucketMetrics)
	// 			updateClusterMetricsInKV(userMetrics, bucketMetrics, clusterMetrics)
	// 			if cfg.Prometheus {
	// 				populateStatus(prysmStatus)
	// 				populateMetricsFromKV(userMetrics, bucketMetrics, clusterMetrics, cfg)
	// 			}
	// 			// Signal that processing is done so the ticker can stop.
	// 			close(done)

	// 			publishEvent(nc, "generate_metrics", "completed", nil, nil)
	// 			//restart
	// 			publishEvent(nc, "sync_users", "in_progress", nil, map[string]string{"sync_mode": "full"})

	// 			// Acknowledge the message.
	// 			if err := msg.Ack(); err != nil {
	// 				log.Error().Err(err).Msg("Failed to acknowledge sync_usage message")
	// 			}

	// 			setTaskRunning("generate_metrics", false)
	// 		}
	// 	default:
	// 		log.Warn().Str("event", eventType).Msg("Unknown event received")
	// 	}

	// 	_ = msg.Ack() // Explicitly acknowledge successful processing
	// }, nats.ManualAck()) // Use Manual Acknowledgment

	// if err != nil {
	// 	log.Fatal().Err(err).Msg("Failed to subscribe to notifications")
	// }
	// defer sub.Unsubscribe()

	// publishEvent(nc, "generate_metrics", "in_progress", nil, map[string]string{"sync_mode": "full"})

	// Launch goroutines for refreshing users, usages, and buckets
	// wg.Add(1)
	// go func() {
	// 	defer wg.Done()
	// 	checkAndRefreshUsers(syncControl, userData, cfg, status)
	// }()

	// wg.Add(1)
	// go func() {
	// 	defer wg.Done()
	// 	checkAndRefreshUserUsages(syncControl, userUsageData, userData, cfg, status)
	// }()

	// wg.Add(1)
	// go func() {
	// 	defer wg.Done()
	// 	checkAndRefreshBuckets(syncControl, bucketData, cfg, status)
	// }()

	// // Launch a goroutine to update metrics periodically
	// wg.Add(1)
	// go func() {
	// 	defer wg.Done()
	// 	updateMetricsPeriodically(cfg, syncControl, userData, userUsageData, bucketData, userMetrics, bucketMetrics, clusterMetrics)
	// }()

	// // Launch a goroutine to populate metrics periodically
	// if cfg.Prometheus {
	// 	wg.Add(1)
	// 	go func() {
	// 		defer wg.Done()
	// 		populateMetricsPeriodically(
	// 			cfg, syncControl, userMetrics, bucketMetrics, clusterMetrics, status,
	// 		)
	// 	}()
	// }

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

// Ptr returns a pointer to the given value (generic version for any type)
func ptr[T any](v T) *T {
	return &v
}
func contains(list []string, item string) bool {
	for _, v := range list {
		if v == item {
			return true
		}
	}
	return false
}
