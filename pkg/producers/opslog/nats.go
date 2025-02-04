// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and prysm contributors
//
// SPDX-License-Identifier: Apache-2.0

package opslog

import (
	"encoding/json"

	"github.com/nats-io/nats.go"
)

func PublishToNATS(nc *nats.Conn, msg interface{}, natsSubject string) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	return nc.Publish(natsSubject, data)
}
