// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and prysm contributors
//
// SPDX-License-Identifier: Apache-2.0

//go:build ceph

package pgprobe

import (
	"fmt"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog/log"
)

var (
	// Per-PG probe result: 1 = success, 0 = failure
	pgProbeSuccess = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "radosgw_index_probe_success",
			Help: "Whether the RADOS stat probe for this PG succeeded (1=ok, 0=failed)",
		},
		[]string{"pgid", "pool", "node", "instance"},
	)

	// Per-PG probe latency in seconds
	pgProbeLatency = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "radosgw_index_probe_latency_seconds",
			Help: "Latency of the RADOS stat probe for this PG in seconds",
		},
		[]string{"pgid", "pool", "node", "instance"},
	)

	// Aggregate: ratio of available PGs in the index pool
	pgPoolAvailableRatio = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "radosgw_index_pool_available_pgs_ratio",
			Help: "Ratio of healthy PGs in the index pool (0.0 to 1.0)",
		},
		[]string{"pool", "node", "instance"},
	)

	// Total PGs in pool (informational)
	pgPoolTotalPGs = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "radosgw_index_pool_total_pgs",
			Help: "Total number of PGs in the index pool",
		},
		[]string{"pool", "node", "instance"},
	)

	// Covered PGs (informational)
	pgPoolCoveredPGs = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "radosgw_index_pool_covered_pgs",
			Help: "Number of PGs covered by the probe bucket",
		},
		[]string{"pool", "node", "instance"},
	)

	// Probe cycle metadata
	probeCycleDuration = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "radosgw_index_probe_cycle_duration_seconds",
			Help: "Duration of the last complete probe cycle in seconds",
		},
		[]string{"pool", "node", "instance"},
	)

	probeCycleTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "radosgw_index_probe_cycles_total",
			Help: "Total number of probe cycles executed",
		},
		[]string{"pool", "node", "instance"},
	)
)

func init() {
	prometheus.MustRegister(pgProbeSuccess)
	prometheus.MustRegister(pgProbeLatency)
	prometheus.MustRegister(pgPoolAvailableRatio)
	prometheus.MustRegister(pgPoolTotalPGs)
	prometheus.MustRegister(pgPoolCoveredPGs)
	prometheus.MustRegister(probeCycleDuration)
	prometheus.MustRegister(probeCycleTotal)
}

// PublishToPrometheus updates all Prometheus metrics from probe results.
func PublishToPrometheus(results []ProbeResult, targets *ProbeTargets, cfg PGProbeConfig) {
	successCount := 0
	var totalLatency float64

	for _, r := range results {
		labels := prometheus.Labels{
			"pgid":     r.PGID,
			"pool":     targets.Pool,
			"node":     cfg.NodeName,
			"instance": cfg.InstanceID,
		}

		if r.Success {
			pgProbeSuccess.With(labels).Set(1)
			successCount++
		} else {
			pgProbeSuccess.With(labels).Set(0)
		}

		pgProbeLatency.With(labels).Set(r.LatencyMs / 1000.0) // Convert ms to seconds
		totalLatency += r.LatencyMs
	}

	// Aggregate metrics
	poolLabels := prometheus.Labels{
		"pool":     targets.Pool,
		"node":     cfg.NodeName,
		"instance": cfg.InstanceID,
	}

	if len(results) > 0 {
		pgPoolAvailableRatio.With(poolLabels).Set(float64(successCount) / float64(len(results)))
	}
	pgPoolTotalPGs.With(poolLabels).Set(float64(targets.TotalPGs))
	pgPoolCoveredPGs.With(poolLabels).Set(float64(targets.CoveredPGs))
	probeCycleDuration.With(poolLabels).Set(totalLatency / 1000.0)
	probeCycleTotal.With(poolLabels).Inc()
}

// StartPrometheusServer starts an HTTP server exposing /metrics.
func StartPrometheusServer(port int) {
	go func() {
		http.Handle("/metrics", promhttp.Handler())
		log.Info().Int("port", port).Msg("starting prometheus metrics server for pg-probe")
		if err := http.ListenAndServe(fmt.Sprintf(":%d", port), nil); err != nil {
			log.Fatal().Err(err).Msg("prometheus metrics server failed")
		}
	}()
}
