// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and prysm contributors
//
// SPDX-License-Identifier: Apache-2.0

package radosgwusage

type RadosGWUsageConfig struct {
	AdminURL                string
	AccessKey               string
	SecretKey               string
	NatsURL                 string // For exporting metrics
	NatsSubject             string // For exporting metrics
	UseNats                 bool   // Indicates if NATS is used for metrics export
	Prometheus              bool
	PrometheusPort          int
	NodeName                string
	InstanceID              string
	Interval                int // in seconds
	ClusterID               string
	SyncControlNats         bool   // Enable NATS for sync control
	SyncExternalNats        bool   // Use external NATS for sync control
	SyncControlURL          string // URL for the external NATS server (if applicable)
	SyncControlBucketPrefix string // NATS-KV bucket prefix for sync data
}
