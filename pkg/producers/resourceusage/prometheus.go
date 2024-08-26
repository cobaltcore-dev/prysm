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

package resourceusage

import (
	"fmt"
	"net/http"

	"github.com/rs/zerolog/log"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	cpuUsageGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "node_cpu_usage_percent",
			Help: "CPU usage percentage of the node",
		},
		[]string{"node", "instance"},
	)
	memoryUsageGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "node_memory_usage_percent",
			Help: "Memory usage percentage of the node",
		},
		[]string{"node", "instance"},
	)
	diskIOGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "node_disk_io_bytes",
			Help: "Disk IO in bytes of the node",
		},
		[]string{"node", "instance"},
	)
	networkIOGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "node_network_io_bytes",
			Help: "Network IO in bytes of the node",
		},
		[]string{"node", "instance"},
	)
)

func init() {
	prometheus.MustRegister(cpuUsageGauge)
	prometheus.MustRegister(memoryUsageGauge)
	prometheus.MustRegister(diskIOGauge)
	prometheus.MustRegister(networkIOGauge)
}

func PublishToPrometheus(usage ResourceUsage, cfg ResourceUsageConfig) {
	cpuUsageGauge.With(prometheus.Labels{
		"node":     cfg.NodeName,
		"instance": cfg.InstanceID,
	}).Set(usage.CPUUsage)

	memoryUsageGauge.With(prometheus.Labels{
		"node":     cfg.NodeName,
		"instance": cfg.InstanceID,
	}).Set(usage.MemoryUsage)

	diskIOGauge.With(prometheus.Labels{
		"node":     cfg.NodeName,
		"instance": cfg.InstanceID,
	}).Set(float64(usage.DiskIO))

	networkIOGauge.With(prometheus.Labels{
		"node":     cfg.NodeName,
		"instance": cfg.InstanceID,
	}).Set(float64(usage.NetworkIO))
}

func StartPrometheusServer(port int) {
	go func() {
		http.Handle("/metrics", promhttp.Handler())
		log.Info().Msgf("starting prometheus metrics server on :%d", port)
		err := http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
		if err != nil {
			log.Fatal().Err(err).Msg("error starting prometheus metrics server")
		}
	}()
}
