// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and prysm contributors
//
// SPDX-License-Identifier: Apache-2.0

package bucketnotify

import (
	"encoding/json"

	"github.com/nats-io/nats.go"
)

func PublishToNATS(nc *nats.Conn, msg interface{}, cfg BucketNotifyConfig) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	return nc.Publish(cfg.NatsSubject, data)
}
