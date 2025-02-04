// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and prysm contributors
//
// SPDX-License-Identifier: Apache-2.0

package quotausageconsumer

type QuotaUsage struct {
	UserID         string `json:"user_id"`
	TotalQuota     uint64 `json:"total_quota"`
	UsedQuota      uint64 `json:"used_quota"`
	RemainingQuota uint64 `json:"remaining_quota"`
	NodeName       string `json:"node_name"`
	InstanceID     string `json:"instance_id"`
}

func StartQuotaUsageConsumer(cfg QuotaUsageConsumerConfig) {

	if cfg.Prometheus {
		StartPrometheusServer(cfg.PrometheusPort)
	}

	StartNatsConsumer(cfg)
}
