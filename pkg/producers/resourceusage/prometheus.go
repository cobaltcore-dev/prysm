// Copyright 2024 Clyso GmbH
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
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
