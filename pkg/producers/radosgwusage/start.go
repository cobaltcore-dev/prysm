// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and prysm contributors
//
// SPDX-License-Identifier: Apache-2.0

package radosgwusage

import (
	"context"
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

// StartRadosGWUsageExporter starts the process of exporting RadosGW usage metrics.
// It supports Prometheus output and sync control using NATS-KV.
func StartRadosGWUsageExporter(cfg RadosGWUsageConfig) {
	if !cfg.SyncControlNats {
		log.Fatal().Msg("sync-control-nats=false is not supported by radosgw-usage yet")
	}

	// Initialize Prometheus server if enabled
	if cfg.Prometheus {
		go startPrometheusMetricsServer(cfg.PrometheusPort)
	}
	var err error

	var natsServer *server.Server
	var nc *nats.Conn
	var js nats.JetStreamContext
	// Start NATS based on configuration
	if cfg.SyncExternalNats {
		nc, err = nats.Connect(cfg.SyncControlURL)
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
	kvStores, err = initializeKeyValueStores(cfg, js)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize Key-Value stores")
	}

	// Start the metric collection loop
	startMetricCollectionLoop(cfg, nc, kvStores)
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
		log.Fatal().Msg("cluster_metrics bucket not found in Key-Value stores")
	}
	return userData, userUsageData, bucketData, userMetrics, bucketMetrics, clusterMetrics
}

func startMetricCollectionLoop(cfg RadosGWUsageConfig, nc *nats.Conn, kvStores map[string]nats.KeyValue) {

	var wg sync.WaitGroup
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

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

	userData, userUsageData, bucketData, userMetrics, bucketMetrics, _ := ensureKeyValueStores(cfg, kvStores)

	wg.Go(func() {
		for {
			select {
			case <-ctx.Done():
				log.Info().Msg("Stopping metric collection loop")
				return
			default:
			}

			if err := syncUsers(userData, cfg, prysmStatus); err != nil {
				prysmStatus.IncrementScrapeErrors()
				log.Error().Err(err).Msg("syncUsers failed")
				select {
				case <-ctx.Done():
					return
				case <-time.After(time.Duration(cfg.CooldownInterval) * time.Second):
				}
				continue
			}
			if err := syncBuckets(bucketData, cfg, prysmStatus); err != nil {
				prysmStatus.IncrementScrapeErrors()
				log.Error().Err(err).Msg("syncBuckets failed")
				select {
				case <-ctx.Done():
					return
				case <-time.After(time.Duration(cfg.CooldownInterval) * time.Second):
				}
				continue
			}
			if err := syncUsage(userUsageData, cfg, prysmStatus); err != nil {
				prysmStatus.IncrementScrapeErrors()
				log.Error().Err(err).Msg("syncUsage failed")
				select {
				case <-ctx.Done():
					return
				case <-time.After(time.Duration(cfg.CooldownInterval) * time.Second):
				}
				continue
			}
			if err := updateUserMetricsInKV(userData, userUsageData, bucketData, userMetrics); err != nil {
				prysmStatus.IncrementScrapeErrors()
				log.Error().Err(err).Msg("updateUserMetricsInKV failed")
				select {
				case <-ctx.Done():
					return
				case <-time.After(time.Duration(cfg.CooldownInterval) * time.Second):
				}
				continue
			}
			if err := updateBucketMetricsInKV(bucketData, userUsageData, bucketMetrics); err != nil {
				prysmStatus.IncrementScrapeErrors()
				log.Error().Err(err).Msg("updateBucketMetricsInKV failed")
				select {
				case <-ctx.Done():
					return
				case <-time.After(time.Duration(cfg.CooldownInterval) * time.Second):
				}
				continue
			}
			if cfg.Prometheus {
				populateMetricsFromKV(userMetrics, bucketMetrics, cfg)
			}
			select {
			case <-ctx.Done():
				return
			case <-time.After(time.Duration(cfg.CooldownInterval) * time.Second):
			}
		}
	})

	// Update prysm status
	if cfg.Prometheus {
		wg.Go(func() {
			ticker := time.NewTicker(2 * time.Second)
			defer ticker.Stop()
			for {
				select {
				case <-ticker.C:
					populateStatus(prysmStatus)
				case <-ctx.Done():
					return
				}
			}
		})
	}

	// Wait for termination signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sigChan)

	log.Info().Msg("Metric collection loop started. Waiting for termination signal.")
	<-sigChan
	log.Info().Msg("Termination signal received. Exiting...")
	cancel()

	// Wait for all goroutines to finish
	wg.Wait()
	log.Info().Msg("All tasks completed. Exiting.")
}

// Ptr returns a pointer to the given value (generic version for any type)
func ptr[T any](v T) *T {
	return &v
}
