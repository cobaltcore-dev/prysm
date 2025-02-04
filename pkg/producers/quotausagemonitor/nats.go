// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and prysm contributors
//
// SPDX-License-Identifier: Apache-2.0

package quotausagemonitor

import (
	"encoding/json"

	"github.com/nats-io/nats.go"
)

func PublishToNATS(nc *nats.Conn, quotas []QuotaUsage, cfg QuotaUsageMonitorConfig) error {
	data, err := json.Marshal(quotas)
	if err != nil {
		return err
	}

	return nc.Publish(cfg.NatsSubject, data)
}
