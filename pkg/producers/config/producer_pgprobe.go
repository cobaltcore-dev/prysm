// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and prysm contributors
//
// SPDX-License-Identifier: Apache-2.0

//go:build ceph

package config

import (
	"github.com/cobaltcore-dev/prysm/pkg/producers/pgprobe"
	"github.com/rs/zerolog/log"
)

func init() {
	registerProducerHandler("pg_probe", startPGProbe)
}

func startPGProbe(producer ProducerConfig, globalConfig GlobalConfig) {
	nodeName := GetStringSetting(producer.Settings, "node_name", globalConfig.NodeName)
	instanceID := GetStringSetting(producer.Settings, "instance_id", globalConfig.InstanceID)
	adminURL := GetStringSetting(producer.Settings, "admin_url", globalConfig.AdminURL)
	accessKey := GetStringSetting(producer.Settings, "access_key", globalConfig.AccessKey)
	secretKey := GetStringSetting(producer.Settings, "secret_key", globalConfig.SecretKey)
	indexPools := GetStringSliceSetting(producer.Settings, "index_pools", []string{})
	probeBucket := GetStringSetting(producer.Settings, "probe_bucket", pgprobe.DefaultProbeBucket)
	cephConfigPath := GetStringSetting(producer.Settings, "ceph_config_path", pgprobe.DefaultCephConfigPath)
	cephUser := GetStringSetting(producer.Settings, "ceph_user", pgprobe.DefaultCephUser)
	interval := GetIntSetting(producer.Settings, "interval", pgprobe.DefaultInterval)
	mappingRefresh := GetIntSetting(producer.Settings, "mapping_refresh_interval", pgprobe.DefaultMappingRefreshInterval)
	probeTimeout := GetIntSetting(producer.Settings, "probe_timeout_ms", pgprobe.DefaultProbeTimeoutMs)
	prometheus := GetBoolSetting(producer.Settings, "prometheus", true)
	prometheusPort := GetIntSetting(producer.Settings, "prometheus_port", pgprobe.DefaultPrometheusPort)

	settings := pgprobe.PGProbeConfig{
		CephConfigPath:         cephConfigPath,
		CephUser:               cephUser,
		IndexPools:             indexPools,
		ProbeBucket:            probeBucket,
		AdminURL:               adminURL,
		AccessKey:              accessKey,
		SecretKey:              secretKey,
		Interval:               interval,
		MappingRefreshInterval: mappingRefresh,
		ProbeTimeoutMs:         probeTimeout,
		Prometheus:             prometheus,
		PrometheusPort:         prometheusPort,
		NodeName:               nodeName,
		InstanceID:             instanceID,
	}

	log.Info().Strs("pools", indexPools).Str("probe_bucket", probeBucket).Msg("--- pg probe ---")
	pgprobe.StartMonitoring(settings)
}
