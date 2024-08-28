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
package kernelmetrics

import (
	"fmt"
	"net/http"

	"github.com/rs/zerolog/log"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	contextSwitchesGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "node_context_switches_total",
			Help: "Total number of context switches",
		},
		[]string{"node", "instance"},
	)
	entropyGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "node_entropy_available_bits",
			Help: "Available entropy in bits",
		},
		[]string{"node", "instance"},
	)
	netConnectionsGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "node_network_connections_total",
			Help: "Total number of network connections",
		},
		[]string{"node", "instance"},
	)
)

func init() {
	prometheus.MustRegister(contextSwitchesGauge)
	prometheus.MustRegister(entropyGauge)
	prometheus.MustRegister(netConnectionsGauge)
}

func PublishToPrometheus(metrics KernelMetrics, cfg KernelMetricsConfig) {
	contextSwitchesGauge.With(prometheus.Labels{
		"node":     cfg.NodeName,
		"instance": cfg.InstanceID,
	}).Set(float64(metrics.ContextSwitches))

	entropyGauge.With(prometheus.Labels{
		"node":     cfg.NodeName,
		"instance": cfg.InstanceID,
	}).Set(float64(metrics.Entropy))

	netConnectionsGauge.With(prometheus.Labels{
		"node":     cfg.NodeName,
		"instance": cfg.InstanceID,
	}).Set(float64(metrics.NetConnections))
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
