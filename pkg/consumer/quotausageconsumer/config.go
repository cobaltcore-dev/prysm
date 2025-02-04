// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and prysm contributors
//
// SPDX-License-Identifier: Apache-2.0

package quotausageconsumer

type QuotaUsageConsumerConfig struct {
	NatsURL           string
	NatsSubject       string
	Prometheus        bool
	PrometheusPort    int
	QuotaUsagePercent float64
	NodeName          string
	InstanceID        string
}
