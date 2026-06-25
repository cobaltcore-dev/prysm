// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and prysm contributors
//
// SPDX-License-Identifier: Apache-2.0

//go:build ceph

package pgprobe

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/ceph/go-ceph/rados"
	"github.com/rs/zerolog/log"
)

// ProbeResult holds the result of probing a single PG via its representative shard object.
type ProbeResult struct {
	Pool       string    `json:"pool"`
	PGID       string    `json:"pgid"`
	ShardObj   string    `json:"shard_obj"`
	Success    bool      `json:"success"`
	LatencyMs  float64   `json:"latency_ms"`
	Error      string    `json:"error,omitempty"`
	ProbeTime  time.Time `json:"probe_time"`
	NodeName   string    `json:"node_name"`
	InstanceID string    `json:"instance_id"`
}

// ProbeTargets maps PG IDs to representative shard objects for a single pool.
type ProbeTargets struct {
	Pool         string         // Index pool name
	BucketMarker string         // Probe bucket marker (bucket instance ID)
	NumShards    int            // Total shards in probe bucket (in this pool)
	PGToShard    map[string]int // PGID -> representative shard index
	ShardToPG    map[int]string // shard index -> PGID
	TotalPGs     int            // Total PGs in pool
	CoveredPGs   int            // PGs covered by probe bucket shards
}

// PoolProbeState holds the runtime state for probing a single pool.
type PoolProbeState struct {
	Pool    string
	IOCtx   *rados.IOContext
	Targets *ProbeTargets
}

// StartMonitoring is the main entry point for the PG probe producer.
// It connects to the RADOS cluster, builds probe targets for all configured
// index pools, and runs the probe loop.
func StartMonitoring(cfg PGProbeConfig) {
	// Apply defaults
	if cfg.CephConfigPath == "" {
		cfg.CephConfigPath = "/etc/ceph/ceph.conf"
	}
	if cfg.CephUser == "" {
		cfg.CephUser = "client.admin"
	}
	if cfg.Interval <= 0 {
		cfg.Interval = 15
	}
	if cfg.MappingRefreshInterval <= 0 {
		cfg.MappingRefreshInterval = 3600
	}
	if cfg.ProbeTimeoutMs <= 0 {
		cfg.ProbeTimeoutMs = 5000
	}

	if len(cfg.IndexPools) == 0 {
		log.Fatal().Msg("no index pools configured; at least one --index-pools entry is required")
	}

	// Connect to RADOS cluster
	// go-ceph's NewConnWithUser expects the user name WITHOUT "client." prefix.
	cephUser := cfg.CephUser
	if strings.HasPrefix(cephUser, "client.") {
		cephUser = strings.TrimPrefix(cephUser, "client.")
	}

	conn, err := rados.NewConnWithUser(cephUser)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create RADOS connection")
	}
	if err := conn.ReadConfigFile(cfg.CephConfigPath); err != nil {
		log.Fatal().Err(err).Str("config", cfg.CephConfigPath).Msg("failed to read ceph config")
	}

	// Set per-operation timeout so rados stat calls don't block indefinitely.
	// This is the idiomatic RADOS timeout mechanism — applied at connection level.
	probeTimeoutSec := fmt.Sprintf("%d", (cfg.ProbeTimeoutMs+999)/1000) // ceil to seconds
	if err := conn.SetConfigOption("rados_osd_op_timeout", probeTimeoutSec); err != nil {
		log.Warn().Err(err).Str("timeout", probeTimeoutSec).Msg("failed to set rados_osd_op_timeout; probes may block indefinitely")
	}

	if err := conn.Connect(); err != nil {
		log.Fatal().Err(err).Msg("failed to connect to RADOS cluster")
	}
	defer conn.Shutdown()

	log.Info().
		Strs("pools", cfg.IndexPools).
		Str("probe_bucket", cfg.ProbeBucket).
		Str("ceph_user", cfg.CephUser).
		Int("interval_seconds", cfg.Interval).
		Msg("connected to RADOS cluster")

	// Initialize per-pool state
	poolStates := make([]*PoolProbeState, 0, len(cfg.IndexPools))

	for _, pool := range cfg.IndexPools {
		state, err := initPoolState(conn, pool, cfg)
		if err != nil {
			log.Fatal().Err(err).Str("pool", pool).Msg("failed to initialize pool state")
		}
		poolStates = append(poolStates, state)
		defer state.IOCtx.Destroy()
	}

	// Prometheus server
	if cfg.Prometheus {
		StartPrometheusServer(cfg.PrometheusPort)
	}

	// Main probe loop
	probeTicker := time.NewTicker(time.Duration(cfg.Interval) * time.Second)
	defer probeTicker.Stop()

	// Mapping refresh ticker
	mappingTicker := time.NewTicker(time.Duration(cfg.MappingRefreshInterval) * time.Second)
	defer mappingTicker.Stop()

	// Run first probe immediately
	runAllPoolProbes(poolStates, cfg)

	for {
		select {
		case <-probeTicker.C:
			runAllPoolProbes(poolStates, cfg)

		case <-mappingTicker.C:
			refreshAllPoolStates(poolStates, conn, cfg)
		}
	}
}

// initPoolState sets up the probe state for a single index pool.
func initPoolState(conn *rados.Conn, pool string, cfg PGProbeConfig) (*PoolProbeState, error) {
	ioctx, err := conn.OpenIOContext(pool)
	if err != nil {
		return nil, fmt.Errorf("failed to open IO context for pool %s: %w", pool, err)
	}

	targets, err := discoverProbeTargets(ioctx, conn, pool, cfg.ProbeBucket)
	if err != nil {
		ioctx.Destroy()
		return nil, fmt.Errorf("failed to discover probe targets in pool %s: %w", pool, err)
	}

	log.Info().
		Str("pool", pool).
		Int("total_pgs", targets.TotalPGs).
		Int("covered_pgs", targets.CoveredPGs).
		Int("probe_shards", targets.NumShards).
		Str("bucket_marker", targets.BucketMarker).
		Msg("probe targets discovered")

	if targets.CoveredPGs < targets.TotalPGs {
		log.Warn().
			Str("pool", pool).
			Int("missing_pgs", targets.TotalPGs-targets.CoveredPGs).
			Msg("probe bucket does not cover all PGs; consider increasing num_shards")
	}

	state := &PoolProbeState{
		Pool:    pool,
		IOCtx:   ioctx,
		Targets: targets,
	}

	return state, nil
}

// runAllPoolProbes executes probes across all pools and publishes results.
func runAllPoolProbes(poolStates []*PoolProbeState, cfg PGProbeConfig) {
	for _, state := range poolStates {
		results := runProbes(state.IOCtx, state.Targets, cfg)

		if cfg.Prometheus {
			PublishToPrometheus(results, state.Targets, cfg)
		}
	}
}

// refreshAllPoolStates re-discovers probe targets (handles PG splits).
func refreshAllPoolStates(poolStates []*PoolProbeState, conn *rados.Conn, cfg PGProbeConfig) {
	for _, state := range poolStates {
		newTargets, err := discoverProbeTargets(state.IOCtx, conn, state.Pool, cfg.ProbeBucket)
		if err != nil {
			log.Error().Err(err).Str("pool", state.Pool).Msg("failed to re-discover probe targets")
		} else {
			state.Targets = newTargets
			log.Info().Str("pool", state.Pool).Int("covered_pgs", newTargets.CoveredPGs).Msg("probe targets refreshed")
		}
	}
}

// discoverProbeTargets enumerates the probe bucket's index shard objects in a pool
// and maps each to its PG using `ceph osd map` via mon commands.
// It resolves the configured probeBucket name to its marker by scanning for a
// matching .bucket.meta.<name>:<marker> object prefix in the pool, then uses
// that marker to select the correct shard objects.
func discoverProbeTargets(ioctx *rados.IOContext, conn *rados.Conn, pool string, probeBucket string) (*ProbeTargets, error) {
	// List all .dir.* and .bucket.meta.* objects in the index pool
	var shardObjects []string
	var probeBucketMarker string

	metaPrefix := ".bucket.meta." + probeBucket + ":"

	iter, err := ioctx.Iter()
	if err != nil {
		return nil, fmt.Errorf("failed to create object iterator: %w", err)
	}
	defer iter.Close()

	for iter.Next() {
		objName := iter.Value()
		if strings.HasPrefix(objName, ".dir.") {
			shardObjects = append(shardObjects, objName)
		}
		// RGW stores .bucket.meta.<bucket_name>:<marker> in the index pool.
		// Use this to resolve the configured probe bucket name to its marker.
		if probeBucketMarker == "" && strings.HasPrefix(objName, metaPrefix) {
			probeBucketMarker = strings.TrimPrefix(objName, metaPrefix)
		}
	}
	if err := iter.Err(); err != nil {
		return nil, fmt.Errorf("error iterating pool objects: %w", err)
	}

	if len(shardObjects) == 0 {
		return nil, fmt.Errorf("no .dir.* objects found in pool %s", pool)
	}

	// Group by bucket marker
	bucketShards := groupShardsByBucket(shardObjects)
	if len(bucketShards) == 0 {
		return nil, fmt.Errorf("no bucket shards identified in pool %s", pool)
	}

	// If we resolved the marker from .bucket.meta, use it directly.
	// If not found, fail — we cannot safely guess which marker belongs to the
	// configured probe bucket without risking selection of a wrong bucket.
	var probeBucketObjList []string

	if probeBucketMarker == "" {
		return nil, fmt.Errorf(
			"probe bucket %q not found in pool %s: no .bucket.meta.%s:* object found; "+
				"verify the bucket exists and its index is in this pool",
			probeBucket, pool, probeBucket,
		)
	}

	shards, ok := bucketShards[probeBucketMarker]
	if !ok || len(shards) == 0 {
		return nil, fmt.Errorf(
			"probe bucket %q (marker %q) has no .dir.* shard objects in pool %s; "+
				"verify the bucket is pre-sharded: radosgw-admin bucket reshard --bucket=%s --num-shards=997",
			probeBucket, probeBucketMarker, pool, probeBucket,
		)
	}
	probeBucketObjList = shards

	log.Debug().
		Str("pool", pool).
		Str("probe_bucket", probeBucket).
		Str("marker", probeBucketMarker).
		Int("shards", len(probeBucketObjList)).
		Msg("identified probe bucket")

	// Map each shard to its PG using mon command "osd map"
	pgToShard := make(map[string]int)
	shardToPG := make(map[int]string)

	for _, obj := range probeBucketObjList {
		shardIdx := extractShardIndex(obj, probeBucketMarker)
		if shardIdx < 0 {
			continue
		}

		pgID, err := getObjectPG(conn, pool, obj)
		if err != nil {
			log.Warn().Err(err).Str("object", obj).Msg("failed to resolve PG for object")
			continue
		}

		// Only keep one representative shard per PG
		if _, exists := pgToShard[pgID]; !exists {
			pgToShard[pgID] = shardIdx
		}
		shardToPG[shardIdx] = pgID
	}

	totalPGs, err := getPoolPGCount(conn, pool)
	if err != nil {
		return nil, fmt.Errorf("failed to get pg_num for pool %s: %w", pool, err)
	}

	return &ProbeTargets{
		Pool:         pool,
		BucketMarker: probeBucketMarker,
		NumShards:    len(probeBucketObjList),
		PGToShard:    pgToShard,
		ShardToPG:    shardToPG,
		TotalPGs:     totalPGs,
		CoveredPGs:   len(pgToShard),
	}, nil
}

// runProbes executes rados stat against one representative shard object per PG.
func runProbes(ioctx *rados.IOContext, targets *ProbeTargets, cfg PGProbeConfig) []ProbeResult {
	results := make([]ProbeResult, 0, len(targets.PGToShard))
	now := time.Now()

	for pgID, shardIdx := range targets.PGToShard {
		objName := fmt.Sprintf(".dir.%s.%d", targets.BucketMarker, shardIdx)

		start := time.Now()
		_, err := ioctx.Stat(objName)
		latency := time.Since(start)

		result := ProbeResult{
			Pool:       targets.Pool,
			PGID:       pgID,
			ShardObj:   objName,
			Success:    err == nil,
			LatencyMs:  float64(latency.Microseconds()) / 1000.0,
			ProbeTime:  now,
			NodeName:   cfg.NodeName,
			InstanceID: cfg.InstanceID,
		}

		if err != nil {
			result.Error = err.Error()
			log.Warn().
				Str("pool", targets.Pool).
				Str("pgid", pgID).
				Str("object", objName).
				Err(err).
				Msg("probe failed")
		}

		results = append(results, result)
	}

	// Log summary
	successCount := 0
	for _, r := range results {
		if r.Success {
			successCount++
		}
	}

	log.Info().
		Str("pool", targets.Pool).
		Int("total_probes", len(results)).
		Int("success", successCount).
		Int("failed", len(results)-successCount).
		Float64("cycle_ms", float64(time.Since(now).Microseconds())/1000.0).
		Msg("probe cycle complete")

	return results
}

// groupShardsByBucket groups .dir.<marker>.<shard_id> objects by their bucket marker.
func groupShardsByBucket(objects []string) map[string][]string {
	groups := make(map[string][]string)
	for _, obj := range objects {
		// Format: .dir.<marker>.<shard_id>
		// Remove .dir. prefix
		rest := strings.TrimPrefix(obj, ".dir.")
		// The marker may contain dots; the shard ID is the last numeric segment
		lastDot := strings.LastIndex(rest, ".")
		if lastDot < 0 {
			continue
		}
		marker := rest[:lastDot]
		groups[marker] = append(groups[marker], obj)
	}
	return groups
}

// extractShardIndex extracts the shard number from an object name.
func extractShardIndex(objName, marker string) int {
	prefix := fmt.Sprintf(".dir.%s.", marker)
	if !strings.HasPrefix(objName, prefix) {
		return -1
	}
	shardStr := strings.TrimPrefix(objName, prefix)
	var idx int
	if _, err := fmt.Sscanf(shardStr, "%d", &idx); err != nil {
		return -1
	}
	return idx
}

// getObjectPG resolves the PG ID for an object using the "osd map" mon command.
// Returns format: "<pool_id>.<pg_hex>" (e.g., "11.2f")
func getObjectPG(conn *rados.Conn, pool, objName string) (string, error) {
	cmd, err := json.Marshal(map[string]string{
		"prefix": "osd map",
		"pool":   pool,
		"object": objName,
		"format": "json",
	})
	if err != nil {
		return "", fmt.Errorf("failed to marshal mon command: %w", err)
	}

	buf, _, err := conn.MonCommand(cmd)
	if err != nil {
		return "", fmt.Errorf("mon command failed: %w", err)
	}

	// Response: {"epoch":123,"pool":"poolname","pool_id":11,"objname":"...","raw_pgid":"11.abc123","pgid":"11.2f","up":[1,2,3],"acting":[1,2,3],...}
	var resp struct {
		PGID string `json:"pgid"`
	}
	if err := json.Unmarshal(buf, &resp); err != nil {
		return "", fmt.Errorf("failed to parse osd map response: %w", err)
	}

	if resp.PGID == "" {
		return "", fmt.Errorf("empty pgid in osd map response")
	}

	return resp.PGID, nil
}

// getPoolPGCount returns the number of PGs in a pool via mon command.
func getPoolPGCount(conn *rados.Conn, poolName string) (int, error) {
	cmd, err := json.Marshal(map[string]string{
		"prefix": "osd pool get",
		"pool":   poolName,
		"var":    "pg_num",
		"format": "json",
	})
	if err != nil {
		return 0, fmt.Errorf("failed to marshal pg_num command: %w", err)
	}

	buf, _, err := conn.MonCommand(cmd)
	if err != nil {
		return 0, fmt.Errorf("mon command failed: %w", err)
	}

	var resp struct {
		PGNum int `json:"pg_num"`
	}
	if err := json.Unmarshal(buf, &resp); err != nil {
		return 0, fmt.Errorf("failed to parse pg_num response: %w", err)
	}

	if resp.PGNum <= 0 {
		return 0, fmt.Errorf("invalid pg_num %d for pool %s", resp.PGNum, poolName)
	}

	return resp.PGNum, nil
}
