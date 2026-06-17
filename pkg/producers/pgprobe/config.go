// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and prysm contributors
//
// SPDX-License-Identifier: Apache-2.0

//go:build ceph

package pgprobe

// PGProbeConfig holds all configuration for the PG probe producer.
type PGProbeConfig struct {
	// RADOS connection
	CephConfigPath string // Path to ceph.conf (default: /etc/ceph/ceph.conf)
	CephUser       string // Ceph auth user (default: client.admin)

	// Probe targets — multiple index pools (one per zone), same probe bucket in each.
	// Example: ["default.rgw.buckets.index", "us-east.rgw.buckets.index"]
	IndexPools  []string // List of index pool names to probe
	ProbeBucket string   // Name of the dedicated probe bucket (e.g., "prysm-probe")

	// RGW Admin API (retained for future use but not used for per-bucket SLI)
	AdminURL  string // RGW Admin API endpoint
	AccessKey string // RGW admin access key
	SecretKey string // RGW admin secret key

	// Probe behavior
	Interval               int // Probe interval in seconds (default: 15)
	MappingRefreshInterval int // How often to refresh probe targets in seconds (default: 3600)
	ProbeTimeoutMs         int // Per-probe timeout in milliseconds (default: 5000)

	// Prometheus
	Prometheus     bool
	PrometheusPort int

	// Identity
	NodeName   string
	InstanceID string
}
