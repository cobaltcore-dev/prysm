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

	"github.com/cobaltcore-dev/prysm/pkg/producers/kernelmetrics"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var (
	kmNatsURL     string
	kmNatsSubject string
	kmUseNats     bool
	kmPromEnabled bool
	kmPromPort    int
	kmNodeName    string
	kmInstanceID  string
	kmInterval    int
)

var kernelMetricsCmd = &cobra.Command{
	Use:   "kernel-metrics",
	Short: "Kernel metrics collector",
	Run: func(cmd *cobra.Command, args []string) {
		config := kernelmetrics.KernelMetricsConfig{
			NatsURL:        kmNatsURL,
			NatsSubject:    kmNatsSubject,
			UseNats:        kmUseNats,
			Prometheus:     kmPromEnabled,
			PrometheusPort: kmPromPort,
			NodeName:       kmNodeName,
			InstanceID:     kmInstanceID,
			Interval:       kmInterval,
		}

		config = mergeKernelMetricsConfigWithEnv(config)
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

		// Finalize the log message with the main message
		event.Msg("configuration_loaded")

		validateKernelMetricsConfig(config)

		kernelmetrics.StartMonitoring(config)
	},
}

func mergeKernelMetricsConfigWithEnv(cfg kernelmetrics.KernelMetricsConfig) kernelmetrics.KernelMetricsConfig {
	cfg.NatsURL = getEnv("NATS_URL", cfg.NatsURL)
	cfg.NatsSubject = getEnv("NATS_SUBJECT", cfg.NatsSubject)
	cfg.NodeName = getEnv("NODE_NAME", cfg.NodeName)
	cfg.InstanceID = getEnv("INSTANCE_ID", cfg.InstanceID)
	cfg.Interval = getEnvInt("INTERVAL", cfg.Interval)
	cfg.PrometheusPort = getEnvInt("PROMETHEUS_PORT", cfg.PrometheusPort)

	return cfg
}

func init() {
	kernelMetricsCmd.Flags().StringVar(&kmNatsURL, "nats-url", "", "NATS server URL")
	kernelMetricsCmd.Flags().StringVar(&kmNatsSubject, "nats-subject", "node.kernel.metrics", "NATS subject to publish metrics")
	kernelMetricsCmd.Flags().BoolVar(&kmPromEnabled, "prometheus", false, "Enable Prometheus metrics")
	kernelMetricsCmd.Flags().IntVar(&kmPromPort, "prometheus-port", 8080, "Prometheus metrics port")
	kernelMetricsCmd.Flags().StringVar(&kmNodeName, "node-name", "", "Name of the node")
	kernelMetricsCmd.Flags().StringVar(&kmInstanceID, "instance-id", "", "Instance ID")
	kernelMetricsCmd.Flags().IntVar(&kmInterval, "interval", 10, "Interval in seconds between metric collections")
}

func validateKernelMetricsConfig(config kernelmetrics.KernelMetricsConfig) {
	missingParams := false

	if missingParams {
		fmt.Println("One or more required parameters are missing. Please provide them through flags or environment variables.")
		os.Exit(1)
	}
}
