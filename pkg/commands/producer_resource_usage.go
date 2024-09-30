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
	"strings"

	"github.com/cobaltcore-dev/prysm/pkg/producers/resourceusage"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var (
	ruNatsURL     string
	ruNatsSubject string
	ruUseNats     bool
	ruPromEnabled bool
	ruPromPort    int
	ruDisksFlag   string
	ruNodeName    string
	ruInstanceID  string
	ruInterval    int
)

var resourceUsageCmd = &cobra.Command{
	Use:   "resource-usage",
	Short: "Resource usage metrics collector",
	Run: func(cmd *cobra.Command, args []string) {
		config := resourceusage.ResourceUsageConfig{
			NatsURL:        ruNatsURL,
			NatsSubject:    ruNatsSubject,
			UseNats:        ruUseNats,
			Prometheus:     ruPromEnabled,
			PrometheusPort: ruPromPort,
			Disks:          strings.Split(ruDisksFlag, ","),
			NodeName:       ruNodeName,
			InstanceID:     ruInstanceID,
			Interval:       ruInterval,
		}

		config = mergeResourceUsageConfigWithEnv(config)

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
		event.Strs("disks", config.Disks)

		// Finalize the log message with the main message
		event.Msg("configuration_loaded")

		validateResourceUsageConfig(config)

		resourceusage.StartMonitoring(config)
	},
}

func mergeResourceUsageConfigWithEnv(cfg resourceusage.ResourceUsageConfig) resourceusage.ResourceUsageConfig {
	cfg.NatsURL = getEnv("NATS_URL", cfg.NatsURL)
	cfg.NatsSubject = getEnv("NATS_SUBJECT", cfg.NatsSubject)
	cfg.PrometheusPort = getEnvInt("PROMETHEUS_PORT", cfg.PrometheusPort)
	disksEnv := getEnv("DISKS", "")
	if disksEnv != "" {
		cfg.Disks = strings.Split(disksEnv, ",")
	}
	cfg.NodeName = getEnv("NODE_NAME", cfg.NodeName)
	cfg.InstanceID = getEnv("INSTANCE_ID", cfg.InstanceID)
	cfg.Interval = getEnvInt("INTERVAL", cfg.Interval)

	return cfg
}

func init() {
	resourceUsageCmd.Flags().StringVar(&ruNatsURL, "nats-url", "", "NATS server URL")
	resourceUsageCmd.Flags().StringVar(&ruNatsSubject, "nats-subject", "node.resource.usage", "NATS subject to publish metrics")
	resourceUsageCmd.Flags().BoolVar(&ruPromEnabled, "prometheus", false, "Enable Prometheus metrics")
	resourceUsageCmd.Flags().IntVar(&ruPromPort, "prometheus-port", 8080, "Prometheus metrics port")
	resourceUsageCmd.Flags().StringVar(&ruDisksFlag, "disks", "sda,sdb", "Comma separated list of disks to monitor")
	resourceUsageCmd.Flags().StringVar(&ruNodeName, "node-name", "", "Name of the node")
	resourceUsageCmd.Flags().StringVar(&ruInstanceID, "instance-id", "", "Instance ID")
	resourceUsageCmd.Flags().IntVar(&ruInterval, "interval", 10, "Interval in seconds between metric collections")
}

func validateResourceUsageConfig(config resourceusage.ResourceUsageConfig) {
	missingParams := false

	if len(config.Disks) == 0 {
		fmt.Println("Warning: --disks or DISKS must be set")
		missingParams = true
	}

	if config.Interval <= 0 {
		fmt.Println("Warning: --interval or INTERVAL must be set and greater than 0")
		missingParams = true
	}

	if missingParams {
		fmt.Println("One or more required parameters are missing. Please provide them through flags or environment variables.")
		os.Exit(1)
	}
}
