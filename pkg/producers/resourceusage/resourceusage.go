// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and prysm contributors
//
// SPDX-License-Identifier: Apache-2.0

package resourceusage

import (
	"time"

	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog/log"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/disk"
	"github.com/shirou/gopsutil/mem"
	"github.com/shirou/gopsutil/net"
)

type ResourceUsage struct {
	CPUUsage    float64 `json:"cpu_usage"`
	MemoryUsage float64 `json:"memory_usage"`
	DiskIO      uint64  `json:"disk_io"`
	NetworkIO   uint64  `json:"network_io"`
	NodeName    string  `json:"node_name"`
	InstanceID  string  `json:"instance_id"`
}

func CollectResourceUsage(cfg ResourceUsageConfig) (ResourceUsage, error) {
	cpuUsage, err := cpu.Percent(time.Second, false)
	if err != nil {
		return ResourceUsage{}, err
	}

	memory, err := mem.VirtualMemory()
	if err != nil {
		return ResourceUsage{}, err
	}

	diskIO, err := disk.IOCounters()
	if err != nil {
		return ResourceUsage{}, err
	}

	var totalDiskIO uint64
	for _, disk := range cfg.Disks {
		if io, exists := diskIO[disk]; exists {
			totalDiskIO += io.WriteBytes + io.ReadBytes
		} else {
			log.Warn().Str("disk", disk).Msg("disk not found")
		}
	}

	networkIO, err := net.IOCounters(false)
	if err != nil {
		return ResourceUsage{}, err
	}

	usage := ResourceUsage{
		CPUUsage:    cpuUsage[0],
		MemoryUsage: memory.UsedPercent,
		DiskIO:      totalDiskIO,
		NetworkIO:   networkIO[0].BytesSent + networkIO[0].BytesRecv,
	}

	return usage, nil
}

func StartMonitoring(cfg ResourceUsageConfig) {
	var nc *nats.Conn
	var err error
	if cfg.UseNats {
		nc, err = nats.Connect(cfg.NatsURL)
		if err != nil {
			log.Fatal().Err(err).Msg("error connecting to nats")
		}
		defer nc.Close()
	}

	if cfg.Prometheus {
		StartPrometheusServer(cfg.PrometheusPort)
	}

	ticker := time.NewTicker(time.Duration(cfg.Interval) * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		usage, err := CollectResourceUsage(cfg)
		if err != nil {
			log.Error().Err(err).Msg("error collecting resource usage")
			continue
		}

		usage.NodeName = cfg.NodeName
		usage.InstanceID = cfg.InstanceID

		if cfg.Prometheus {
			PublishToPrometheus(usage, cfg)
		}

		if cfg.UseNats {
			if err := PublishToNATS(nc, usage, cfg); err != nil {
				log.Error().Err(err).Msg("error publishing to nats")
			}
		} else {
			log.Info().Interface("resource_usage", usage).Msg("resource usage")
		}
	}
}
