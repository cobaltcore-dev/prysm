// Copyright 2024 Clyso GmbH
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package commands

import (
	"fmt"
	"os"

	"github.com/cobaltcore-dev/prysm/pkg/producers/radosgwusage"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var (
	rgwuAdminURL                string
	rgwuAccessKey               string
	rgwuSecretKey               string
	rgwuNatsURL                 string
	rgwuNatsSubject             string
	rgwuUseNats                 bool
	rgwuPrometheus              bool
	rgwuPrometheusPort          int
	rgwuNodeName                string
	rgwuInstanceID              string
	rgwuInterval                int
	rgwuClusterID               string
	rgwuSyncControlNats         bool
	rgwuSyncExternalNats        bool
	rgwuSyncControlURL          string
	rgwuSyncControlBucketPrefix string
)

var radosGWUsageCmd = &cobra.Command{
	Use:   "radosgw-usage",
	Short: "RadosGW usage exporter",
	Run: func(cmd *cobra.Command, args []string) {
		config := radosgwusage.RadosGWUsageConfig{
			AdminURL:                rgwuAdminURL,
			AccessKey:               rgwuAccessKey,
			SecretKey:               rgwuSecretKey,
			NatsURL:                 rgwuNatsURL,
			NatsSubject:             rgwuNatsSubject,
			UseNats:                 rgwuUseNats,
			Prometheus:              rgwuPrometheus,
			PrometheusPort:          rgwuPrometheusPort,
			NodeName:                rgwuNodeName,
			InstanceID:              rgwuInstanceID,
			Interval:                rgwuInterval,
			ClusterID:               rgwuClusterID,
			SyncControlNats:         rgwuSyncControlNats,
			SyncExternalNats:        rgwuSyncExternalNats,
			SyncControlURL:          rgwuSyncControlURL,
			SyncControlBucketPrefix: rgwuSyncControlBucketPrefix,
		}

		config = mergeRadosGWUsageConfigWithEnv(config)
		config.UseNats = config.NatsURL != ""

		event := log.Info()
		event.Bool("use_nats", config.UseNats)
		if config.UseNats {
			event.Str("nats_url", config.NatsURL)
			event.Str("nats_subject", config.NatsSubject)
		}

		event.Bool("prometheus_enabled", config.Prometheus)
		if config.Prometheus {
			event.Int("prometheus_port", config.PrometheusPort)
		}

		event.Str("node_name", config.NodeName)
		event.Str("instance_id", config.InstanceID)
		event.Int("interval_seconds", config.Interval)
		event.Str("cluster_id", config.ClusterID)

		event.Bool("sync_control_nats_enabled", config.SyncControlNats)
		if config.SyncControlNats {
			event.Bool("sync_external_nats_enabled", config.SyncExternalNats)
			if config.SyncExternalNats {
				event.Str("sync_control_url", config.SyncControlURL)
			}
			event.Str("sync_control_bucket_prefix", config.SyncControlBucketPrefix)
		}

		// Finalize the log message with the main message
		event.Msg("configuration_loaded")

		validateRadosGWUsageConfig(config)

		radosgwusage.StartRadosGWUsageExporter(config)
	},
}

func mergeRadosGWUsageConfigWithEnv(cfg radosgwusage.RadosGWUsageConfig) radosgwusage.RadosGWUsageConfig {
	cfg.AdminURL = getEnv("ADMIN_URL", cfg.AdminURL)
	cfg.AccessKey = getEnv("ACCESS_KEY", cfg.AccessKey)
	cfg.SecretKey = getEnv("SECRET_KEY", cfg.SecretKey)
	cfg.NatsURL = getEnv("NATS_URL", cfg.NatsURL)
	cfg.NatsSubject = getEnv("NATS_SUBJECT", cfg.NatsSubject)
	cfg.NodeName = getEnv("NODE_NAME", cfg.NodeName)
	cfg.InstanceID = getEnv("INSTANCE_ID", cfg.InstanceID)
	cfg.Prometheus = getEnvBool("PROMETHEUS_ENABLED", cfg.Prometheus)
	cfg.PrometheusPort = getEnvInt("PROMETHEUS_PORT", cfg.PrometheusPort)
	cfg.Interval = getEnvInt("INTERVAL", cfg.Interval)
	cfg.ClusterID = getEnv("RGW_CLUSTER_ID", cfg.ClusterID)
	// Sync control related parameters
	cfg.SyncControlNats = getEnvBool("SYNC_CONTROL_NATS", cfg.SyncControlNats)
	cfg.SyncExternalNats = getEnvBool("SYNC_EXTERNAL_NATS", cfg.SyncExternalNats)
	cfg.SyncControlURL = getEnv("SYNC_CONTROL_URL", cfg.SyncControlURL)
	cfg.SyncControlBucketPrefix = getEnv("SYNC_CONTROL_BUCKET_PREFIX", cfg.SyncControlBucketPrefix)

	return cfg
}

func init() {
	radosGWUsageCmd.Flags().StringVar(&rgwuAdminURL, "admin-url", "", "Admin URL for the RadosGW instance")
	radosGWUsageCmd.Flags().StringVar(&rgwuAccessKey, "access-key", "", "Access key for the RadosGW admin")
	radosGWUsageCmd.Flags().StringVar(&rgwuSecretKey, "secret-key", "", "Secret key for the RadosGW admin")
	radosGWUsageCmd.Flags().StringVar(&rgwuNatsURL, "nats-url", "", "NATS server URL")
	radosGWUsageCmd.Flags().StringVar(&rgwuNatsSubject, "nats-subject", "rgw.usage", "NATS subject to publish usage")
	radosGWUsageCmd.Flags().StringVar(&rgwuClusterID, "rgw-cluster-id", "", "RGW Cluster ID added to metrics")
	radosGWUsageCmd.Flags().StringVar(&rgwuNodeName, "node-name", "", "Name of the node")
	radosGWUsageCmd.Flags().StringVar(&rgwuInstanceID, "instance-id", "", "Instance ID")
	radosGWUsageCmd.Flags().BoolVar(&rgwuPrometheus, "prometheus", false, "Enable Prometheus metrics")
	radosGWUsageCmd.Flags().IntVar(&rgwuPrometheusPort, "prometheus-port", 8080, "Prometheus metrics port")
	radosGWUsageCmd.Flags().IntVar(&rgwuInterval, "interval", 10, "Interval in seconds between usage collections")
	// Sync control related flags
	radosGWUsageCmd.Flags().BoolVar(&rgwuSyncControlNats, "sync-control-nats", true, "Enable sync control using NATS")
	radosGWUsageCmd.Flags().BoolVar(&rgwuSyncExternalNats, "sync-external-nats", false, "Use external NATS server for sync control")
	radosGWUsageCmd.Flags().StringVar(&rgwuSyncControlURL, "sync-control-url", "", "URL of the external NATS server for sync control")
	radosGWUsageCmd.Flags().StringVar(&rgwuSyncControlBucketPrefix, "sync-control-bucket-prefix", "sync", "NATS KV bucket prefix for sync control")

}

func validateRadosGWUsageConfig(config radosgwusage.RadosGWUsageConfig) {
	missingParams := false

	if config.AdminURL == "" {
		fmt.Println("Warning: --admin-url or ADMIN_URL must be set")
		missingParams = true
	}
	if config.AccessKey == "" {
		fmt.Println("Warning: --access-key or ACCESS_KEY must be set")
		missingParams = true
	}
	if config.SecretKey == "" {
		fmt.Println("Warning: --secret-key or SECRET_KEY must be set")
		missingParams = true
	}
	if config.Interval <= 0 {
		fmt.Println("Warning: --interval or INTERVAL must be a positive duration")
		missingParams = true
	}

	if config.ClusterID == "" {
		fmt.Println("Warning: --rgw-cluster-id or RGW_CLUSTER_ID must be set")
		missingParams = true
	}

	// Validate sync control configuration
	if config.SyncControlNats {
		if config.SyncExternalNats && config.SyncControlURL == "" {
			fmt.Println("Warning: --sync-control-url must be set when using an external NATS server")
			missingParams = true
		}
		if config.SyncControlBucketPrefix == "" {
			fmt.Println("Warning: --sync-control-bucket-prefix must be set for sync control")
			missingParams = true
		}
	}

	if missingParams {
		fmt.Println("One or more required parameters are missing. Please provide them through flags or environment variables.")
		os.Exit(1)
	}
}
