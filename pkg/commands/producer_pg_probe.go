// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and prysm contributors
//
// SPDX-License-Identifier: Apache-2.0

//go:build ceph

package commands

import (
	"fmt"
	"os"
	"strings"

	"github.com/cobaltcore-dev/prysm/pkg/producers/pgprobe"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var (
	pgpCephConfigPath         string
	pgpCephUser               string
	pgpIndexPools             string // comma-separated list of index pools
	pgpProbeBucket            string
	pgpAdminURL               string
	pgpAccessKey              string
	pgpSecretKey              string
	pgpInterval               int
	pgpMappingRefreshInterval int
	pgpProbeTimeoutMs         int
	pgpPromEnabled            bool
	pgpPromPort               int
	pgpNodeName               string
	pgpInstanceID             string
)

var pgProbeCmd = &cobra.Command{
	Use:   "pg-probe",
	Short: "RGW index pool PG availability probe (L3 leading indicator)",
	Long: `Probes all Placement Groups in the RGW bucket index pools by executing
rados stat against representative shard objects from a pre-sharded probe bucket.

Supports multiple index pools (one per zone). The same probe bucket exists in
each pool via multisite replication or manual pre-sharding.

This is a Layer 3 infrastructure health leading indicator. It predicts
customer-facing impact before it appears in the L1/L2 SLI (radosgw_request_total).
It is NOT the SLI of record.

Metrics exposed:
  - radosgw_index_probe_success{pgid, pool} (per PG, per pool)
  - radosgw_index_probe_latency_seconds{pgid, pool} (per PG, per pool)
  - radosgw_index_pool_available_pgs_ratio{pool} (aggregate per pool)

Prerequisites:
  1. Create a probe bucket: radosgw-admin bucket create --bucket=prysm-probe --uid=admin
  2. Pre-shard it: radosgw-admin bucket reshard --bucket=prysm-probe --num-shards=997
  3. Verify coverage per pool: all PGs should have at least one shard`,
	Run: func(cmd *cobra.Command, args []string) {
		// Parse comma-separated pool list
		pools := parsePoolList(pgpIndexPools)

		config := pgprobe.PGProbeConfig{
			CephConfigPath:         pgpCephConfigPath,
			CephUser:               pgpCephUser,
			IndexPools:             pools,
			ProbeBucket:            pgpProbeBucket,
			AdminURL:               pgpAdminURL,
			AccessKey:              pgpAccessKey,
			SecretKey:              pgpSecretKey,
			Interval:               pgpInterval,
			MappingRefreshInterval: pgpMappingRefreshInterval,
			ProbeTimeoutMs:         pgpProbeTimeoutMs,
			Prometheus:             pgpPromEnabled,
			PrometheusPort:         pgpPromPort,
			NodeName:               pgpNodeName,
			InstanceID:             pgpInstanceID,
		}

		config = mergePGProbeConfigWithEnv(config)

		event := log.Info()
		event.Strs("index_pools", config.IndexPools).
			Str("probe_bucket", config.ProbeBucket).
			Str("ceph_config", config.CephConfigPath).
			Str("ceph_user", config.CephUser).
			Int("interval_seconds", config.Interval).
			Int("mapping_refresh_seconds", config.MappingRefreshInterval).
			Bool("prometheus", config.Prometheus)
		if config.Prometheus {
			event.Int("prometheus_port", config.PrometheusPort)
		}
		event.Msg("pg-probe configuration loaded")

		validatePGProbeConfig(config)

		pgprobe.StartMonitoring(config)
	},
}

func mergePGProbeConfigWithEnv(cfg pgprobe.PGProbeConfig) pgprobe.PGProbeConfig {
	cfg.CephConfigPath = getEnv("CEPH_CONFIG_PATH", cfg.CephConfigPath)
	cfg.CephUser = getEnv("CEPH_USER", cfg.CephUser)
	cfg.ProbeBucket = getEnv("PROBE_BUCKET", cfg.ProbeBucket)
	cfg.AdminURL = getEnv("ADMIN_URL", cfg.AdminURL)
	cfg.AccessKey = getEnv("ACCESS_KEY", cfg.AccessKey)
	cfg.SecretKey = getEnv("SECRET_KEY", cfg.SecretKey)
	cfg.Interval = getEnvInt("PROBE_INTERVAL", cfg.Interval)
	cfg.MappingRefreshInterval = getEnvInt("MAPPING_REFRESH_INTERVAL", cfg.MappingRefreshInterval)
	cfg.ProbeTimeoutMs = getEnvInt("PROBE_TIMEOUT_MS", cfg.ProbeTimeoutMs)
	cfg.Prometheus = getEnvBool("PROMETHEUS", cfg.Prometheus)
	cfg.PrometheusPort = getEnvInt("PROMETHEUS_PORT", cfg.PrometheusPort)
	cfg.NodeName = getEnv("NODE_NAME", cfg.NodeName)
	cfg.InstanceID = getEnv("INSTANCE_ID", cfg.InstanceID)

	// INDEX_POOLS env var overrides flag value
	indexPoolsEnv := getEnv("INDEX_POOLS", "")
	if indexPoolsEnv != "" {
		cfg.IndexPools = parsePoolList(indexPoolsEnv)
	}

	return cfg
}

// parsePoolList splits a comma-separated pool list and trims whitespace.
func parsePoolList(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	pools := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			pools = append(pools, p)
		}
	}
	return pools
}

func init() {
	localProducerCmd.AddCommand(pgProbeCmd)

	pgProbeCmd.Flags().StringVar(&pgpCephConfigPath, "ceph-config", "/etc/ceph/ceph.conf", "Path to ceph.conf")
	pgProbeCmd.Flags().StringVar(&pgpCephUser, "ceph-user", "client.admin", "Ceph auth user")
	pgProbeCmd.Flags().StringVar(&pgpIndexPools, "index-pools", "", "Comma-separated list of RGW bucket index pool names (required)")
	pgProbeCmd.Flags().StringVar(&pgpProbeBucket, "probe-bucket", "prysm-probe", "Name of the pre-sharded probe bucket (same bucket in all pools)")
	pgProbeCmd.Flags().StringVar(&pgpAdminURL, "admin-url", "", "RGW Admin API URL")
	pgProbeCmd.Flags().StringVar(&pgpAccessKey, "access-key", "", "RGW admin access key")
	pgProbeCmd.Flags().StringVar(&pgpSecretKey, "secret-key", "", "RGW admin secret key")
	pgProbeCmd.Flags().IntVar(&pgpInterval, "interval", 15, "Probe interval in seconds")
	pgProbeCmd.Flags().IntVar(&pgpMappingRefreshInterval, "mapping-refresh-interval", 3600, "Probe target refresh interval in seconds")
	pgProbeCmd.Flags().IntVar(&pgpProbeTimeoutMs, "probe-timeout-ms", 5000, "Per-probe timeout in milliseconds")
	pgProbeCmd.Flags().BoolVar(&pgpPromEnabled, "prometheus", true, "Enable Prometheus metrics")
	pgProbeCmd.Flags().IntVar(&pgpPromPort, "prometheus-port", 9120, "Prometheus metrics port")
	pgProbeCmd.Flags().StringVar(&pgpNodeName, "node-name", "", "Node name for metric labels")
	pgProbeCmd.Flags().StringVar(&pgpInstanceID, "instance-id", "", "Instance ID for metric labels")
}

func validatePGProbeConfig(config pgprobe.PGProbeConfig) {
	missingParams := false

	if len(config.IndexPools) == 0 {
		fmt.Println("Error: --index-pools or INDEX_POOLS is required (comma-separated)")
		missingParams = true
	}

	if config.ProbeBucket == "" {
		fmt.Println("Error: --probe-bucket or PROBE_BUCKET is required")
		missingParams = true
	}

	if missingParams {
		fmt.Println("\nRequired parameters are missing. See --help for usage.")
		os.Exit(1)
	}
}
