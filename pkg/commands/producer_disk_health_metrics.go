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

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"gitlab.clyso.com/clyso/radosguard/pkg/producers/diskhealthmetrics"
)

var (
	dhmNatsURL                     string
	dhmNatsSubject                 string
	dhmUseNats                     bool
	dhmPromEnabled                 bool
	dhmPromPort                    int
	dhmAllAttributes               bool
	dhmDisksFlag                   string
	dhmNodeName                    string
	dhmInstanceID                  string
	dhmIncludeZeroValues           bool
	dhmInterval                    int
	dhmGrownDefectsThreshold       int64
	dhmPendingSectorsThreshold     int64
	dhmReallocatedSectorsThreshold int64
	dhmLifetimeUsedThreshold       int64
)

var diskHealthMetricsCmd = &cobra.Command{
	Use:   "disk-health-metrics",
	Short: "Disk health metrics collector and media error logger",
	Run: func(cmd *cobra.Command, args []string) {
		config := diskhealthmetrics.DiskHealthMetricsConfig{
			NatsURL:                     dhmNatsURL,
			NatsSubject:                 dhmNatsSubject,
			UseNats:                     dhmUseNats,
			Prometheus:                  dhmPromEnabled,
			PrometheusPort:              dhmPromPort,
			AllAttributes:               dhmAllAttributes,
			Disks:                       strings.Split(dhmDisksFlag, ","),
			NodeName:                    dhmNodeName,
			InstanceID:                  dhmInstanceID,
			IncludeZeroValues:           dhmIncludeZeroValues,
			Interval:                    dhmInterval,
			GrownDefectsThreshold:       dhmGrownDefectsThreshold,
			PendingSectorsThreshold:     dhmPendingSectorsThreshold,
			ReallocatedSectorsThreshold: dhmReallocatedSectorsThreshold,
			LifetimeUsedThreshold:       dhmLifetimeUsedThreshold,
		}

		config = mergeDiskHealthMetricsConfigWithEnv(config)

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

		event.Bool("all_attributes", config.AllAttributes).
			Str("disks", fmt.Sprintf("%v", config.Disks)).
			Str("node_name", config.NodeName).
			Str("instance_id", config.InstanceID).
			Int("interval_seconds", config.Interval)
		// Finalize the log message with the main message

		event.Msg("configuration_loaded")

		validateDiskHealthMetricsConfig(config)

		diskhealthmetrics.StartMonitoring(config)
	},
}

func mergeDiskHealthMetricsConfigWithEnv(cfg diskhealthmetrics.DiskHealthMetricsConfig) diskhealthmetrics.DiskHealthMetricsConfig {
	cfg.NatsURL = getEnv("NATS_URL", cfg.NatsURL)
	cfg.NatsSubject = getEnv("NATS_SUBJECT", cfg.NatsSubject)
	cfg.PrometheusPort = getEnvInt("PROMETHEUS_PORT", cfg.PrometheusPort)
	cfg.AllAttributes = getEnvBool("ALL_ATTR", cfg.AllAttributes)
	disksEnv := getEnv("DISKS", "")
	if disksEnv != "" {
		cfg.Disks = strings.Split(disksEnv, ",")
	}
	cfg.NodeName = getEnv("NODE_NAME", cfg.NodeName)
	cfg.InstanceID = getEnv("INSTANCE_ID", cfg.InstanceID)
	cfg.IncludeZeroValues = getEnvBool("INCLUDE_ZERO_VALUES", cfg.IncludeZeroValues)
	cfg.Interval = getEnvInt("INTERVAL", cfg.Interval)
	cfg.GrownDefectsThreshold = getEnvInt64("GROWN_DEFECTS_THRESHOLD", cfg.GrownDefectsThreshold)
	cfg.PendingSectorsThreshold = getEnvInt64("PENDING_SECTORS_THRESHOLD", cfg.PendingSectorsThreshold)
	cfg.ReallocatedSectorsThreshold = getEnvInt64("REALLOCATED_SECTORS_THRESHOLD", cfg.ReallocatedSectorsThreshold)
	cfg.LifetimeUsedThreshold = getEnvInt64("LIFETIME_USED_THRESHOLD", cfg.LifetimeUsedThreshold)

	return cfg
}

func init() {
	diskHealthMetricsCmd.Flags().StringVar(&dhmNatsURL, "nats-url", "", "NATS server URL")
	diskHealthMetricsCmd.Flags().StringVar(&dhmNatsSubject, "nats-subject", "osd.disk.health", "NATS subject to publish metrics")
	diskHealthMetricsCmd.Flags().BoolVar(&dhmPromEnabled, "prometheus", false, "Enable Prometheus metrics")
	diskHealthMetricsCmd.Flags().IntVar(&dhmPromPort, "prometheus-port", 8080, "Prometheus metrics port")
	// diskHealthMetricsCmd.Flags().BoolVar(&dhmAllAttributes, "all-attr", false, "Monitor all SMART attributes")
	diskHealthMetricsCmd.Flags().StringVar(&dhmDisksFlag, "disks", "/dev/sda,/dev/sdb", "Comma separated list of disks to monitor")
	// diskHealthMetricsCmd.Flags().BoolVar(&dhmIncludeZeroValues, "include-zero-values", false, "Include attributes with zero values")
	diskHealthMetricsCmd.Flags().IntVar(&dhmInterval, "interval", 10, "Interval in seconds between metric collections")
	diskHealthMetricsCmd.Flags().Int64Var(&dhmGrownDefectsThreshold, "grown-defects-threshold", 10, "Threshold for grown defects to trigger a warning")
	diskHealthMetricsCmd.Flags().Int64Var(&dhmPendingSectorsThreshold, "pending-sectors-threshold", 3, "Threshold for pending sectors to trigger a warning")
	diskHealthMetricsCmd.Flags().Int64Var(&dhmReallocatedSectorsThreshold, "reallocated-sectors-threshold", 10, "Threshold for reallocated sectors to trigger a warning")
	diskHealthMetricsCmd.Flags().Int64Var(&dhmLifetimeUsedThreshold, "lifetime-used-threshold", 80, "Threshold for SSD lifetime used percentage to trigger a critical alert")
}

func validateDiskHealthMetricsConfig(config diskhealthmetrics.DiskHealthMetricsConfig) {
	missingParams := false

	if len(config.Disks) == 0 {
		fmt.Println("Warning: --disks or DISKS must be set")
		missingParams = true
	}

	if missingParams {
		fmt.Println("One or more required parameters are missing. Please provide them through flags or environment variables.")
		os.Exit(1)
	}
}
