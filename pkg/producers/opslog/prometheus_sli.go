// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and prysm contributors
//
// SPDX-License-Identifier: Apache-2.0

package opslog

import (
	"math"
	"sync"
	"sync/atomic"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog/log"
)

// SLICollectorConfig holds configuration for the SLI collector.
type SLICollectorConfig struct {
	// StaleTTL is the duration after which a series with no updates stops
	// being emitted and is reaped from memory. This prevents unbounded
	// growth from tenants with no traffic. Reaped counters leave a tombstone
	// behind (see sliTombstoneTTL) so their cumulative value survives the
	// idle period.
	StaleTTL time.Duration
	// ReapInterval controls how often the reaper goroutine runs. Defaults to StaleTTL/4.
	ReapInterval time.Duration
	// Region is the deployment region label (e.g. "eu-de-1").
	Region string
}

// sliCounterKey uniquely identifies a label combination for the SLI counter metric.
// Per ADR: keyed by tenant (not bucket), with protocol and operation labels.
type sliCounterKey struct {
	Tenant      string
	Protocol    string
	Operation   string
	StatusClass string
}

// sliLatencyKey uniquely identifies a label combination for the SLI latency histogram.
type sliLatencyKey struct {
	Tenant    string
	Protocol  string
	Operation string
}

type sliCounterState struct {
	value    uint64
	lastSeen time.Time
}

// sliCounterTombstone preserves the cumulative value of a reaped counter so
// that a tenant resuming traffic after an idle period continues from its
// previous total instead of restarting at zero. Without this, the Prometheus
// series would reset to zero on resume and the documented 30-day
// increase()-based SLO would silently lose the requests recorded before the
// idle period (counter-reset compensation is not guaranteed across a
// staleness gap).
type sliCounterTombstone struct {
	value    uint64
	reapedAt time.Time
}

// sliTombstoneTTL bounds how long a reaped counter's cumulative value is
// remembered. It must be at least as long as the longest SLO query window
// (the documented monthly SLO uses increase(...[30d])):
//
//   - Idle ≤ TTL: the tombstone survives, the series resumes monotonically,
//     and increase() is exact across the gap.
//   - Idle > TTL: the tombstone expires and the series restarts at zero —
//     but a 30-day window cannot then contain both pre- and post-gap
//     samples (the gap alone already exceeds the window), so no 30-day
//     query observes the discontinuity.
//
// Memory stays bounded to roughly one TTL of idle counter keys.
const sliTombstoneTTL = 30 * 24 * time.Hour

// sliLatencyState holds histogram bucket accumulators.
type sliLatencyState struct {
	// Fixed histogram buckets per ADR: 0.05, 0.1, 0.5, 1, 5, 30, +Inf
	buckets  [7]float64 // cumulative count per le boundary
	sum      float64
	count    uint64
	lastSeen time.Time
}

// sliHistogramBounds defines the fixed le boundaries per ADR.
var sliHistogramBounds = [7]float64{0.05, 0.1, 0.5, 1.0, 5.0, 30.0, math.MaxFloat64}

// sliCollector implements prometheus.Collector with stale series reaping.
// Emits:
//   - radosgw_request_total{tenant, protocol, operation, status_class, region}
//   - radosgw_request_duration_seconds{tenant, protocol, operation, region} (histogram)
//   - radosgw_sli_stale_series_reaped_total (housekeeping counter)
type sliCollector struct {
	mu         sync.RWMutex
	counters   map[sliCounterKey]*sliCounterState
	latency    map[sliLatencyKey]*sliLatencyState
	tombstones map[sliCounterKey]sliCounterTombstone

	config        SLICollectorConfig
	counterDesc   *prometheus.Desc
	histogramDesc *prometheus.Desc
	reapedDesc    *prometheus.Desc
	reapedTotal   atomic.Uint64
	stopReaper    chan struct{}
}

// globalSLICollector is the package-level instance used by observeSLI.
var globalSLICollector *sliCollector

func newSLICollector(cfg SLICollectorConfig) *sliCollector {
	// Reap interval normalization:
	//   - unset (0) → defaults to StaleTTL/4
	//   - clamped to a minimum of one minute so the reaper can never spin
	//     on the hot path. An explicitly configured interval below one
	//     minute is therefore NOT honored; the effective interval is
	//     max(StaleTTL/4, 1m) when unset, and at least 1m when set.
	//     Operators wanting tighter reap deadlines should lower StaleTTL
	//     instead (series are only reaped once they exceed StaleTTL).
	if cfg.ReapInterval == 0 {
		cfg.ReapInterval = cfg.StaleTTL / 4
	}
	if cfg.ReapInterval < time.Minute {
		cfg.ReapInterval = time.Minute
	}
	if cfg.Region == "" {
		cfg.Region = "unknown"
	}

	c := &sliCollector{
		counters:   make(map[sliCounterKey]*sliCounterState),
		latency:    make(map[sliLatencyKey]*sliLatencyState),
		tombstones: make(map[sliCounterKey]sliCounterTombstone),
		config:     cfg,
		counterDesc: prometheus.NewDesc(
			"radosgw_request_total",
			"RGW data-plane requests for SLI/SLO evaluation, aggregated per tenant. Only data-plane operations (get, put, list, delete, head, multipart) are counted; control-plane ops (create_bucket, delete_bucket, ACL, policy, lifecycle, etc.) are excluded from both numerator and denominator.",
			[]string{"tenant", "protocol", "operation", "status_class", "region"},
			nil,
		),
		histogramDesc: prometheus.NewDesc(
			"radosgw_request_duration_seconds",
			"RGW data-plane request latency histogram for SLI/SLO evaluation. Same operation scope as radosgw_request_total.",
			[]string{"tenant", "protocol", "operation", "region"},
			nil,
		),
		reapedDesc: prometheus.NewDesc(
			"radosgw_sli_stale_series_reaped_total",
			"Total number of stale SLI series that have been reaped",
			nil,
			nil,
		),
		stopReaper: make(chan struct{}),
	}

	return c
}

// Describe implements prometheus.Collector.
func (c *sliCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.counterDesc
	ch <- c.histogramDesc
	ch <- c.reapedDesc
}

// Collect implements prometheus.Collector.
func (c *sliCollector) Collect(ch chan<- prometheus.Metric) {
	now := time.Now()
	region := c.config.Region

	c.mu.RLock()
	defer c.mu.RUnlock()

	// Emit counter metrics
	for key, state := range c.counters {
		if now.Sub(state.lastSeen) > c.config.StaleTTL {
			continue // stale — skip (reaper will clean up)
		}
		m, err := prometheus.NewConstMetric(
			c.counterDesc,
			prometheus.CounterValue,
			float64(state.value),
			key.Tenant, key.Protocol, key.Operation, key.StatusClass, region,
		)
		if err == nil {
			ch <- m
		}
	}

	// Emit histogram metrics
	for key, state := range c.latency {
		if now.Sub(state.lastSeen) > c.config.StaleTTL {
			continue
		}

		// Build cumulative bucket map for prometheus.NewConstHistogram
		buckets := make(map[float64]uint64, 7)
		for i := 0; i < 6; i++ {
			buckets[sliHistogramBounds[i]] = uint64(state.buckets[i])
		}
		buckets[math.Inf(1)] = state.count

		m, err := prometheus.NewConstHistogram(
			c.histogramDesc,
			uint64(state.count),
			state.sum,
			buckets,
			key.Tenant, key.Protocol, key.Operation, region,
		)
		if err == nil {
			ch <- m
		}
	}

	// Emit reaper counter
	m, err := prometheus.NewConstMetric(
		c.reapedDesc,
		prometheus.CounterValue,
		float64(c.reapedTotal.Load()),
	)
	if err == nil {
		ch <- m
	}
}

// observeCounter records a request count into the collector.
func (c *sliCollector) observeCounter(tenant, protocol, operation, statusClass string) {
	now := time.Now()

	counterKey := sliCounterKey{
		Tenant:      tenant,
		Protocol:    protocol,
		Operation:   operation,
		StatusClass: statusClass,
	}

	c.mu.Lock()
	cs, ok := c.counters[counterKey]
	if !ok {
		cs = &sliCounterState{}
		// Resume from a reaped counter's tombstone (if any) so the exported
		// series stays monotonic across idle periods — see sliCounterTombstone.
		if ts, ok := c.tombstones[counterKey]; ok {
			cs.value = ts.value
			delete(c.tombstones, counterKey)
		}
		c.counters[counterKey] = cs
	}
	cs.value++
	cs.lastSeen = now
	c.mu.Unlock()
}

// observeLatency records a latency observation into the histogram.
func (c *sliCollector) observeLatency(tenant, protocol, operation string, seconds float64) {
	now := time.Now()

	latencyKey := sliLatencyKey{
		Tenant:    tenant,
		Protocol:  protocol,
		Operation: operation,
	}

	c.mu.Lock()
	ls, ok := c.latency[latencyKey]
	if !ok {
		ls = &sliLatencyState{}
		c.latency[latencyKey] = ls
	}

	// Increment cumulative histogram buckets
	for i, bound := range sliHistogramBounds {
		if seconds <= bound {
			// Cumulative: increment this bucket and all higher ones
			for j := i; j < len(sliHistogramBounds); j++ {
				ls.buckets[j]++
			}
			break
		}
	}
	ls.sum += seconds
	ls.count++
	ls.lastSeen = now
	c.mu.Unlock()
}

// startReaper starts the background goroutine that removes stale series.
func (c *sliCollector) startReaper() {
	go func() {
		ticker := time.NewTicker(c.config.ReapInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				c.reap()
			case <-c.stopReaper:
				return
			}
		}
	}()
}

// reap removes series that haven't been updated within StaleTTL.
// Reaped counters leave a tombstone carrying their cumulative value, so a
// tenant returning after an idle period resumes its previous total rather
// than resetting the Prometheus series to zero. Latency histograms are
// deleted outright: they are consumed via short-window rate() queries,
// which an idle gap already excludes, and resuming stale bucket counts
// would skew fresh latency quantiles.
func (c *sliCollector) reap() {
	now := time.Now()
	var reaped uint64

	c.mu.Lock()
	for key, state := range c.counters {
		if now.Sub(state.lastSeen) > c.config.StaleTTL {
			// Preserve the cumulative value so a later resume continues the
			// series monotonically instead of resetting to zero (which
			// would drop pre-idle requests from the monthly increase()-
			// based SLO).
			c.tombstones[key] = sliCounterTombstone{value: state.value, reapedAt: now}
			delete(c.counters, key)
			reaped++
		}
	}
	// Expire tombstones that have outlived the SLO query window they exist
	// to protect (see sliTombstoneTTL).
	for key, ts := range c.tombstones {
		if now.Sub(ts.reapedAt) > sliTombstoneTTL {
			delete(c.tombstones, key)
		}
	}
	for key, state := range c.latency {
		if now.Sub(state.lastSeen) > c.config.StaleTTL {
			delete(c.latency, key)
			reaped++
		}
	}
	c.mu.Unlock()

	if reaped > 0 {
		c.reapedTotal.Add(reaped)
		log.Info().Uint64("reaped_series", reaped).Msg("SLI collector reaped stale series")
	}
}

// seriesCount returns the current number of tracked counter series.
func (c *sliCollector) seriesCount() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.counters)
}

// counterValue returns the current counter value for a given label combination (for testing).
func (c *sliCollector) counterValue(tenant, protocol, operation, statusClass string) float64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	key := sliCounterKey{Tenant: tenant, Protocol: protocol, Operation: operation, StatusClass: statusClass}
	if s, ok := c.counters[key]; ok {
		return float64(s.value)
	}
	return 0
}

// latencyCount returns the observation count for a given latency key (for testing).
func (c *sliCollector) latencyCount(tenant, protocol, operation string) float64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	key := sliLatencyKey{Tenant: tenant, Protocol: protocol, Operation: operation}
	if s, ok := c.latency[key]; ok {
		return float64(s.count)
	}
	return 0
}

// registerSLIMetrics creates the global SLI collector and registers it with Prometheus.
func registerSLIMetrics(cfg SLICollectorConfig) {
	globalSLICollector = newSLICollector(cfg)
	prometheus.MustRegister(globalSLICollector)
	globalSLICollector.startReaper()
	log.Info().
		Dur("stale_ttl", globalSLICollector.config.StaleTTL).
		Dur("reap_interval", globalSLICollector.config.ReapInterval).
		Str("region", globalSLICollector.config.Region).
		Msg("SLI collector registered (radosgw_request_total + radosgw_request_duration_seconds)")
}
