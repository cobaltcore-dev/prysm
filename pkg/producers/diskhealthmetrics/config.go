// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and prysm contributors
//
// SPDX-License-Identifier: Apache-2.0

package diskhealthmetrics

type DiskHealthMetricsConfig struct {
	NatsURL           string
	NatsSubject       string
	UseNats           bool
	Prometheus        bool
	PrometheusPort    int
	AllAttributes     bool
	Disks             []string
	IncludeZeroValues bool
	Interval          int // in seconds
	NodeName          string
	InstanceID        string

	// NATS event thresholds
	GrownDefectsThreshold       int64
	PendingSectorsThreshold     int64
	ReallocatedSectorsThreshold int64
	LifetimeUsedThreshold       int64 // percentage

	CephOSDBasePath string
}
