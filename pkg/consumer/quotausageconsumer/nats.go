// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and prysm contributors
//
// SPDX-License-Identifier: Apache-2.0

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
