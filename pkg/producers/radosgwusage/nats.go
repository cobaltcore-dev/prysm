// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and prysm contributors
//
// SPDX-License-Identifier: Apache-2.0

package radosgwusage

import (
	"encoding/json"

	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog/log"
)

func publishToNATS(nc *nats.Conn, subject string, usages []UsageEntry) {
	usageJSON, err := json.Marshal(usages)
	if err != nil {
		log.Error().
			Err(err).
			Msg("error marshalling usage to JSON")
		return
	}

	err = nc.Publish(subject, usageJSON)
	if err != nil {
		log.Error().
			Err(err).
			Str("subject", subject).
			Msg("error publishing usage to NATS")
	}
}
