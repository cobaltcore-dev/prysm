// Copyright 2024 Clyso GmbH
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
