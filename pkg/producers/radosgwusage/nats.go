// Copyright (C) 2024 Clyso GmbH
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

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
