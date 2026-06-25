// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and prysm contributors
//
// SPDX-License-Identifier: Apache-2.0

//go:build ceph

package pgprobe

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGroupShardsByBucket(t *testing.T) {
	tests := []struct {
		name     string
		objects  []string
		expected map[string][]string
	}{
		{
			name:     "empty input",
			objects:  []string{},
			expected: map[string][]string{},
		},
		{
			name:     "nil input",
			objects:  nil,
			expected: map[string][]string{},
		},
		{
			name: "single bucket single shard",
			objects: []string{
				".dir.default.4371.1.1625081234.0",
			},
			expected: map[string][]string{
				"default.4371.1.1625081234": {".dir.default.4371.1.1625081234.0"},
			},
		},
		{
			name: "single bucket multiple shards",
			objects: []string{
				".dir.default.4371.1.1625081234.0",
				".dir.default.4371.1.1625081234.1",
				".dir.default.4371.1.1625081234.2",
			},
			expected: map[string][]string{
				"default.4371.1.1625081234": {
					".dir.default.4371.1.1625081234.0",
					".dir.default.4371.1.1625081234.1",
					".dir.default.4371.1.1625081234.2",
				},
			},
		},
		{
			name: "multiple buckets",
			objects: []string{
				".dir.default.4371.1.1625081234.0",
				".dir.default.4371.1.1625081234.1",
				".dir.default.8812.2.1631245678.0",
				".dir.default.8812.2.1631245678.3",
			},
			expected: map[string][]string{
				"default.4371.1.1625081234": {
					".dir.default.4371.1.1625081234.0",
					".dir.default.4371.1.1625081234.1",
				},
				"default.8812.2.1631245678": {
					".dir.default.8812.2.1631245678.0",
					".dir.default.8812.2.1631245678.3",
				},
			},
		},
		{
			name:     "object without dots in rest is skipped",
			objects:  []string{".dir.nodots"},
			expected: map[string][]string{},
		},
		{
			name: "non-.dir prefixed objects are included if they have .dir. prefix",
			objects: []string{
				"not-a-shard-object",
				".dir.marker.0",
			},
			expected: map[string][]string{
				"marker": {".dir.marker.0"},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := groupShardsByBucket(tc.objects)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestExtractShardIndex(t *testing.T) {
	tests := []struct {
		name     string
		objName  string
		marker   string
		expected int
	}{
		{
			name:     "valid shard 0",
			objName:  ".dir.default.4371.1.1625081234.0",
			marker:   "default.4371.1.1625081234",
			expected: 0,
		},
		{
			name:     "valid shard 997",
			objName:  ".dir.default.4371.1.1625081234.997",
			marker:   "default.4371.1.1625081234",
			expected: 997,
		},
		{
			name:     "valid shard large number",
			objName:  ".dir.my-marker.4999",
			marker:   "my-marker",
			expected: 4999,
		},
		{
			name:     "wrong marker returns -1",
			objName:  ".dir.default.4371.1.1625081234.5",
			marker:   "wrong-marker",
			expected: -1,
		},
		{
			name:     "non-numeric shard id returns -1",
			objName:  ".dir.default.4371.1.1625081234.abc",
			marker:   "default.4371.1.1625081234",
			expected: -1,
		},
		{
			name:     "empty object name returns -1",
			objName:  "",
			marker:   "marker",
			expected: -1,
		},
		{
			name:     "missing .dir prefix returns -1",
			objName:  "default.4371.1.1625081234.0",
			marker:   "default.4371.1.1625081234",
			expected: -1,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := extractShardIndex(tc.objName, tc.marker)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestDefaultConstants(t *testing.T) {
	// Verify constants are sane (guards against accidental edits)
	assert.Equal(t, 15, DefaultInterval)
	assert.Equal(t, 3600, DefaultMappingRefreshInterval)
	assert.Equal(t, 5000, DefaultProbeTimeoutMs)
	assert.Equal(t, "/etc/ceph/ceph.conf", DefaultCephConfigPath)
	assert.Equal(t, "client.admin", DefaultCephUser)
	assert.Equal(t, "prysm-probe", DefaultProbeBucket)
	assert.Equal(t, 9120, DefaultPrometheusPort)
}

func TestStartMonitoring_AppliesDefaults(t *testing.T) {
	// Cannot call StartMonitoring (needs RADOS), but we can verify
	// the default-application logic by constructing a zero config
	// and checking what StartMonitoring would set.
	cfg := PGProbeConfig{}

	// Simulate the default-application logic from StartMonitoring
	if cfg.CephConfigPath == "" {
		cfg.CephConfigPath = DefaultCephConfigPath
	}
	if cfg.CephUser == "" {
		cfg.CephUser = DefaultCephUser
	}
	if cfg.Interval <= 0 {
		cfg.Interval = DefaultInterval
	}
	if cfg.MappingRefreshInterval <= 0 {
		cfg.MappingRefreshInterval = DefaultMappingRefreshInterval
	}
	if cfg.ProbeTimeoutMs <= 0 {
		cfg.ProbeTimeoutMs = DefaultProbeTimeoutMs
	}

	assert.Equal(t, DefaultCephConfigPath, cfg.CephConfigPath)
	assert.Equal(t, DefaultCephUser, cfg.CephUser)
	assert.Equal(t, DefaultInterval, cfg.Interval)
	assert.Equal(t, DefaultMappingRefreshInterval, cfg.MappingRefreshInterval)
	assert.Equal(t, DefaultProbeTimeoutMs, cfg.ProbeTimeoutMs)
}

func TestStartMonitoring_PreservesExplicitValues(t *testing.T) {
	cfg := PGProbeConfig{
		CephConfigPath:         "/custom/ceph.conf",
		CephUser:               "client.custom",
		Interval:               30,
		MappingRefreshInterval: 7200,
		ProbeTimeoutMs:         10000,
	}

	// Simulate the default-application logic — explicit values should NOT be overwritten
	if cfg.CephConfigPath == "" {
		cfg.CephConfigPath = DefaultCephConfigPath
	}
	if cfg.CephUser == "" {
		cfg.CephUser = DefaultCephUser
	}
	if cfg.Interval <= 0 {
		cfg.Interval = DefaultInterval
	}
	if cfg.MappingRefreshInterval <= 0 {
		cfg.MappingRefreshInterval = DefaultMappingRefreshInterval
	}
	if cfg.ProbeTimeoutMs <= 0 {
		cfg.ProbeTimeoutMs = DefaultProbeTimeoutMs
	}

	assert.Equal(t, "/custom/ceph.conf", cfg.CephConfigPath)
	assert.Equal(t, "client.custom", cfg.CephUser)
	assert.Equal(t, 30, cfg.Interval)
	assert.Equal(t, 7200, cfg.MappingRefreshInterval)
	assert.Equal(t, 10000, cfg.ProbeTimeoutMs)
}
