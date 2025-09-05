// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and prysm contributors
//
// SPDX-License-Identifier: Apache-2.0

package commands

import (
	"fmt"
	"os"
	"strings"

	"github.com/cobaltcore-dev/prysm/pkg/producers/diskhealthmetrics"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
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
	dhmCephOSDBasePath             string
	dhmTestMode                    bool
	dhmTestDataPath                string
	dhmTestScenario                string
	dhmTestDevices                 string
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
			CephOSDBasePath:             dhmCephOSDBasePath,
			TestMode:                    dhmTestMode,
			TestDataPath:                dhmTestDataPath,
			TestScenario:                dhmTestScenario,
		}

		// Parse test devices if provided
		if dhmTestDevices != "" {
			config.TestDevices = strings.Split(dhmTestDevices, ",")
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
			Int("interval_seconds", config.Interval).
			Str("ceph_osd_base_path", config.CephOSDBasePath)
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
	cfg.CephOSDBasePath = getEnv("CEPH_OSD_BASE_PATH", cfg.CephOSDBasePath)
	
	// Test mode environment variables
	cfg.TestMode = getEnvBool("TEST_MODE", cfg.TestMode)
	cfg.TestDataPath = getEnv("TEST_DATA_PATH", cfg.TestDataPath)
	cfg.TestScenario = getEnv("TEST_SCENARIO", cfg.TestScenario)
	
	testDevicesEnv := getEnv("TEST_DEVICES", "")
	if testDevicesEnv != "" {
		cfg.TestDevices = strings.Split(testDevicesEnv, ",")
	}

	return cfg
}

func init() {
	diskHealthMetricsCmd.Flags().StringVar(&dhmNatsURL, "nats-url", "", "NATS server URL")
	diskHealthMetricsCmd.Flags().StringVar(&dhmNatsSubject, "nats-subject", "osd.disk.health", "NATS subject to publish metrics")
	diskHealthMetricsCmd.Flags().BoolVar(&dhmPromEnabled, "prometheus", false, "Enable Prometheus metrics")
	diskHealthMetricsCmd.Flags().IntVar(&dhmPromPort, "prometheus-port", 8080, "Prometheus metrics port")
	// diskHealthMetricsCmd.Flags().BoolVar(&dhmAllAttributes, "all-attr", false, "Monitor all SMART attributes")
	diskHealthMetricsCmd.Flags().StringVar(&dhmDisksFlag, "disks", "/dev/sda,/dev/sdb", "Comma-separated list of disks to monitor, e.g., \"/dev/sda,/dev/sdb\". Use \"*\" to monitor all available disks.")
	// diskHealthMetricsCmd.Flags().BoolVar(&dhmIncludeZeroValues, "include-zero-values", false, "Include attributes with zero values")
	diskHealthMetricsCmd.Flags().IntVar(&dhmInterval, "interval", 10, "Interval in seconds between metric collections")
	diskHealthMetricsCmd.Flags().Int64Var(&dhmGrownDefectsThreshold, "grown-defects-threshold", 10, "Threshold for grown defects to trigger a warning")
	diskHealthMetricsCmd.Flags().Int64Var(&dhmPendingSectorsThreshold, "pending-sectors-threshold", 3, "Threshold for pending sectors to trigger a warning")
	diskHealthMetricsCmd.Flags().Int64Var(&dhmReallocatedSectorsThreshold, "reallocated-sectors-threshold", 10, "Threshold for reallocated sectors to trigger a warning")
	diskHealthMetricsCmd.Flags().Int64Var(&dhmLifetimeUsedThreshold, "lifetime-used-threshold", 80, "Threshold for SSD lifetime used percentage to trigger a critical alert")
	diskHealthMetricsCmd.Flags().StringVar(&dhmCephOSDBasePath, "ceph-osd-base-path", "/var/lib/rook/rook-ceph/", "Base path for mapping devices to Ceph OSD numbers")
	
	// Test mode flags
	diskHealthMetricsCmd.Flags().BoolVar(&dhmTestMode, "test-mode", false, "Enable test mode with simulated data (no smartctl required)")
	diskHealthMetricsCmd.Flags().StringVar(&dhmTestDataPath, "test-data-path", "", "Path to test data directory (default: pkg/producers/diskhealthmetrics/testdata)")
	diskHealthMetricsCmd.Flags().StringVar(&dhmTestScenario, "test-scenario", "mixed", "Test scenario: healthy, failing, mixed")
	diskHealthMetricsCmd.Flags().StringVar(&dhmTestDevices, "test-devices", "", "Comma-separated list of test device names (default: nvme0,nvme1,sda,sdb)")
}

func validateDiskHealthMetricsConfig(config diskhealthmetrics.DiskHealthMetricsConfig) {
	missingParams := false

	// In test mode, disks are optional (will use default test devices)
	if !config.TestMode && len(config.Disks) == 0 {
		fmt.Println("Warning: --disks or DISKS must be set (or use --test-mode)")
		missingParams = true
	}

	if missingParams {
		fmt.Println("One or more required parameters are missing. Please provide them through flags or environment variables.")
		os.Exit(1)
	}
}
