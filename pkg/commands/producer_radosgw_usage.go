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

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"gitlab.clyso.com/clyso/radosguard/pkg/producers/radosgwusage"
)

var (
	rgwuAdminURL       string
	rgwuAccessKey      string
	rgwuSecretKey      string
	rgwuNatsURL        string
	rgwuNatsSubject    string
	rgwuUseNats        bool
	rgwuPrometheus     bool
	rgwuPrometheusPort int
	rgwuNodeName       string
	rgwuInstanceID     string
	rgwuInterval       int
	rgwuClusterID      string
)

var radosGWUsageCmd = &cobra.Command{
	Use:   "radosgw-usage",
	Short: "RadosGW usage exporter",
	Run: func(cmd *cobra.Command, args []string) {
		config := radosgwusage.RadosGWUsageConfig{
			AdminURL:       rgwuAdminURL,
			AccessKey:      rgwuAccessKey,
			SecretKey:      rgwuSecretKey,
			NatsURL:        rgwuNatsURL,
			NatsSubject:    rgwuNatsSubject,
			UseNats:        rgwuUseNats,
			Prometheus:     rgwuPrometheus,
			PrometheusPort: rgwuPrometheusPort,
			NodeName:       rgwuNodeName,
			InstanceID:     rgwuInstanceID,
			Interval:       rgwuInterval,
			ClusterID:      rgwuClusterID,
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

	if missingParams {
		fmt.Println("One or more required parameters are missing. Please provide them through flags or environment variables.")
		os.Exit(1)
	}
}
