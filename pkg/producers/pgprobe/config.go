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
	CephConfigPath string
	CephUser       string

	// Multiple index pools supported (one per zone), same probe bucket in each.
	IndexPools  []string
	ProbeBucket string

	// Credentials sourced exclusively from env vars (ACCESS_KEY/SECRET_KEY)
	// to avoid exposure in ps aux.
	AdminURL  string
	AccessKey string
	SecretKey string

	Interval               int
	MappingRefreshInterval int
	ProbeTimeoutMs         int

	Prometheus     bool
	PrometheusPort int

	NodeName   string
	InstanceID string
}
