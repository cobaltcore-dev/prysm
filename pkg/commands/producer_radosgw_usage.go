// Copyright (c) 2024 Clyso GmbH
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program. If not, see <https://www.gnu.org/licenses/>.

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
	rgwuStore          string
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
			Store:          rgwuStore,
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
		event.Str("store", config.Store)

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
	cfg.Store = getEnv("STORE", cfg.Store)

	return cfg
}

func init() {
	radosGWUsageCmd.Flags().StringVar(&rgwuAdminURL, "admin-url", "", "Admin URL for the RadosGW instance")
	radosGWUsageCmd.Flags().StringVar(&rgwuAccessKey, "access-key", "", "Access key for the RadosGW admin")
	radosGWUsageCmd.Flags().StringVar(&rgwuSecretKey, "secret-key", "", "Secret key for the RadosGW admin")
	radosGWUsageCmd.Flags().StringVar(&rgwuNatsURL, "nats-url", "", "NATS server URL")
	radosGWUsageCmd.Flags().StringVar(&rgwuNatsSubject, "nats-subject", "rgw.usage", "NATS subject to publish usage")
	radosGWUsageCmd.Flags().StringVar(&rgwuStore, "store", "us-east-1", "Store name added to metrics")
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

	if missingParams {
		fmt.Println("One or more required parameters are missing. Please provide them through flags or environment variables.")
		os.Exit(1)
	}
}
