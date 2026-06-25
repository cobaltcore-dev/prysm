// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and prysm contributors
//
// SPDX-License-Identifier: Apache-2.0

//go:build ceph

package pgprobe

// Default configuration values. These are the single source of truth —
// referenced by both StartMonitoring guards and CLI flag defaults.
const (
	DefaultCephConfigPath         = "/etc/ceph/ceph.conf"
	DefaultCephUser               = "client.admin"
	DefaultInterval               = 15
	DefaultMappingRefreshInterval = 3600
	DefaultProbeTimeoutMs         = 5000
	DefaultProbeBucket            = "prysm-probe"
	DefaultPrometheusPort         = 9120
)

// PGProbeConfig holds all configuration for the PG probe producer.
type PGProbeConfig struct {
	// RADOS connection
	CephConfigPath string // Path to ceph.conf
	CephUser       string // Ceph auth user

	// Probe targets — multiple index pools (one per zone), same probe bucket in each.
	// Example: ["default.rgw.buckets.index", "us-east.rgw.buckets.index"]
	IndexPools  []string // List of index pool names to probe
	ProbeBucket string   // Name of the dedicated probe bucket

	// RGW Admin API (credentials via ACCESS_KEY/SECRET_KEY env vars only)
	AdminURL  string // RGW Admin API endpoint
	AccessKey string // RGW admin access key (env-only, not exposed via CLI)
	SecretKey string // RGW admin secret key (env-only, not exposed via CLI)

	// Probe behavior
	Interval               int // Probe interval in seconds
	MappingRefreshInterval int // How often to refresh probe targets in seconds
	ProbeTimeoutMs         int // Per-probe timeout in milliseconds

	// Prometheus
	Prometheus     bool
	PrometheusPort int

	// Identity
	NodeName   string
	InstanceID string
}
