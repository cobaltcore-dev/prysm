// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and prysm contributors
//
// SPDX-License-Identifier: Apache-2.0

package quotausageconsumer

import (
	"fmt"
	"net/http"

	"github.com/rs/zerolog/log"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	quotaUsageGaugeVec = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "quota_usage",
			Help: "Quota usage for users",
		},
		[]string{"user_id", "node", "instance"},
	)
)

func init() {
	prometheus.MustRegister(quotaUsageGaugeVec)
}

func PublishToPrometheus(quotas []QuotaUsage, cfg QuotaUsageConsumerConfig) {
	for _, quota := range quotas {
		if quota.TotalQuota > 0 {
			usedQuota := quota.UsedQuota
			if usedQuota > quota.TotalQuota {
				usedQuota = quota.TotalQuota
			}
			usagePercent := (float64(usedQuota) / float64(quota.TotalQuota)) * 100
			if usagePercent >= cfg.QuotaUsagePercent {
				quotaUsageGaugeVec.With(prometheus.Labels{
					"user_id":  quota.UserID,
					"node":     quota.NodeName,
					"instance": quota.InstanceID,
				}).Set(usagePercent)
			}
		}
	}
}

func StartPrometheusServer(port int) {
	go func() {
		http.Handle("/metrics", promhttp.Handler())
		log.Info().Msgf("starting prometheus metrics server on :%d", port)
		err := http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
		if err != nil {
			log.Fatal().Err(err).Msg("error starting prometheus metrics server")
		}
	}()
}
