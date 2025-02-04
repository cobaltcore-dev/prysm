// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and prysm contributors
//
// SPDX-License-Identifier: Apache-2.0

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
