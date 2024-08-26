// Copyright (c) 2024 Clyso GmbH
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program. If not, see <https://www.gnu.org/licenses/>.

package quotausageconsumer

import (
	"encoding/json"

	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog/log"
)

func StartNatsConsumer(cfg QuotaUsageConsumerConfig) {
	nc, err := nats.Connect(cfg.NatsURL)
	if err != nil {
		log.Fatal().Err(err).Msg("error connecting to nats")
	}
	defer nc.Close()

	_, err = nc.Subscribe(cfg.NatsSubject, func(m *nats.Msg) {
		var quotas []QuotaUsage
		err := json.Unmarshal(m.Data, &quotas)
		if err != nil {
			log.Error().Err(err).Msg("error unmarshalling quotas")
			return
		}

		if cfg.Prometheus {
			PublishToPrometheus(quotas, cfg)
		} else {
			for _, quota := range quotas {
				if quota.TotalQuota > 0 {
					usedQuota := quota.UsedQuota
					if usedQuota > quota.TotalQuota {
						usedQuota = quota.TotalQuota
					}
					usagePercent := (float64(usedQuota) / float64(quota.TotalQuota)) * 100
					if usagePercent >= cfg.QuotaUsagePercent {
						log.Info().
							Str("userid", quota.UserID).
							Float64("usagepercent", usagePercent).
							Msgf("user: %s, usage: %.2f%%", quota.UserID, usagePercent)
					}
				}
			}
		}
	})
	if err != nil {
		log.Fatal().Err(err).Msg("error subscribing to nats subject")
	}

	select {}
}
