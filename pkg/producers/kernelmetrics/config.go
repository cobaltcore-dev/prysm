// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and prysm contributors
//
// SPDX-License-Identifier: Apache-2.0

package kernelmetrics

type KernelMetricsConfig struct {
	NatsURL        string
	NatsSubject    string
	UseNats        bool
	NodeName       string
	InstanceID     string
	Prometheus     bool
	PrometheusPort int
	Interval       int
}
