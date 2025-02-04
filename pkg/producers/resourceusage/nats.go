// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and prysm contributors
//
// SPDX-License-Identifier: Apache-2.0

package resourceusage

import (
	"encoding/json"

	"github.com/nats-io/nats.go"
)

func PublishToNATS(nc *nats.Conn, usage ResourceUsage, cfg ResourceUsageConfig) error {
	data, err := json.Marshal(usage)
	if err != nil {
		return err
	}

	return nc.Publish(cfg.NatsSubject, data)
}
