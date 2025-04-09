// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and prysm contributors
//
// SPDX-License-Identifier: Apache-2.0

package opslog

type OpsLogConfig struct {
	LogFilePath             string
	TruncateLogOnStart      bool
	SocketPath              string
	NatsURL                 string
	NatsSubject             string
	NatsMetricsSubject      string
	UseNats                 bool
	LogToStdout             bool
	LogRetentionDays        int   // Number of days to keep old log files
	MaxLogFileSize          int64 // Maximum log file size in bytes before rotation
	Prometheus              bool
	PrometheusPort          int
	PodName                 string
	IgnoreAnonymousRequests bool
	MetricsConfig           MetricsConfig
}

type MetricsConfig struct {
	// Request Tracking
	TrackRequestsByIP                   bool // High cardinality, should be optional
	TrackRequestsByIPBucketMethodTenant bool // High cardinality, should be optional
	TrackRequestsByUser                 bool
	TrackRequestsByBucket               bool
	TrackRequestsByMethod               bool
	TrackRequestsByOperation            bool
	TrackRequestsByStatus               bool
	TrackRequestsByTenant               bool // Potentially useful for multi-tenant insights

	// Data Usage Tracking
	TrackBytesSentByIP         bool // High cardinality, should be optional
	TrackBytesReceivedByIP     bool // High cardinality, should be optional
	TrackBytesSentByUser       bool
	TrackBytesReceivedByUser   bool
	TrackBytesSentByBucket     bool
	TrackBytesReceivedByBucket bool

	// Error Tracking
	TrackErrorsByIP     bool // High cardinality, should be optional
	TrackErrorsByUser   bool
	TrackErrorsByBucket bool
	TrackErrorsByStatus bool

	// Latency & Performance
	TrackLatencyByUser            bool // Fine-grained user tracking
	TrackLatencyByBucket          bool // Latency grouped per bucket
	TrackLatencyByTenant          bool // Aggregated latency by tenant
	TrackLatencyByMethod          bool // Per HTTP method
	TrackLatencyByBucketAndMethod bool // Combination of both (detailed)
}
