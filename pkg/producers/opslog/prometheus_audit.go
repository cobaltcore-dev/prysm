// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and prysm contributors
//
// SPDX-License-Identifier: Apache-2.0

package opslog

import "github.com/prometheus/client_golang/prometheus"

// auditEventsDropped counts audit events that were not published, labelled by
// the reason they were dropped (e.g. "no_tenant"). The metric is always
// defined so the drop path can record regardless of whether the Prometheus
// endpoint is enabled; registration only affects exposure.
var auditEventsDropped = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "prysm_audit_events_dropped_total",
		Help: "Audit events dropped before publishing to RabbitMQ, by reason",
	},
	[]string{"reason"},
)

func registerAuditMetrics() {
	prometheus.MustRegister(auditEventsDropped)
}
