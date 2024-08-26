// Copyright (C) 2024 Clyso GmbH
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package kernelmetrics

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/nats-io/nats.go"
	"github.com/shirou/gopsutil/host"
	"github.com/shirou/gopsutil/mem"
	"github.com/shirou/gopsutil/net"
)

type KernelMetrics struct {
	ContextSwitches uint64 `json:"context_switches"`
	Entropy         uint64 `json:"entropy"`
	NetConnections  uint64 `json:"net_connections"`
	NodeName        string `json:"node_name"`
	InstanceID      string `json:"instance_id"`
}

func collectKernelMetrics(cfg KernelMetricsConfig) (KernelMetrics, error) {
	vmStat, err := mem.VirtualMemory()
	if err != nil {
		return KernelMetrics{}, err
	}

	connections, err := net.Connections("all")
	if err != nil {
		return KernelMetrics{}, err
	}

	hostInfo, err := host.Info()
	if err != nil {
		return KernelMetrics{}, err
	}

	metrics := KernelMetrics{
		ContextSwitches: vmStat.VMallocTotal,
		Entropy:         hostInfo.BootTime,
		NetConnections:  uint64(len(connections)),
		NodeName:        cfg.NodeName,
		InstanceID:      cfg.InstanceID,
	}

	return metrics, nil
}

func StartMonitoring(cfg KernelMetricsConfig) {
	var nc *nats.Conn
	var err error
	if cfg.UseNats {
		nc, err = nats.Connect(cfg.NatsURL)
		if err != nil {
			log.Fatal().
				Err(err).
				Msg("error connecting to NATS") // Log a fatal error if the connection fails and exit
		}
		defer nc.Close()
	}

	if cfg.Prometheus {
		StartPrometheusServer(cfg.PrometheusPort)
	}

	ticker := time.NewTicker(time.Duration(cfg.Interval) * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		metrics, err := collectKernelMetrics(cfg)
		if err != nil {
			log.Error().
				Err(err).
				Msg("error collecting kernel metrics")
			continue // Skip to the next iteration if an error occurs
		}

		if cfg.Prometheus {
			PublishToPrometheus(metrics, cfg)
		}

		if cfg.UseNats {
			if err := PublishToNATS(nc, metrics, cfg); err != nil {
				log.Printf("Error publishing to NATS: %v", err)
			}
		} else {
			metricsJSON, err := json.Marshal(metrics)
			if err != nil {
				log.Error().
					Err(err).
					Msg("error marshalling entries to JSON")
				continue // Skip to the next iteration if an error occurs
			}
			fmt.Println(string(metricsJSON))
		}
	}
}
