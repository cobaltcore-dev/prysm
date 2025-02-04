// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and prysm contributors
//
// SPDX-License-Identifier: Apache-2.0

package resourceusage

type ResourceUsageConfig struct {
	NatsURL        string
	NatsSubject    string
	UseNats        bool
	Prometheus     bool
	PrometheusPort int
	Interval       int // in seconds
	Disks          []string
	NodeName       string
	InstanceID     string
}
