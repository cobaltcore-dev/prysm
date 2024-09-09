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

package config

import (
	"sync"

	"github.com/rs/zerolog/log"

	"gitlab.clyso.com/clyso/radosguard/pkg/producers/bucketnotify"
	"gitlab.clyso.com/clyso/radosguard/pkg/producers/resourceusage"
)

func StartProducers(producer ProducerConfig, globalConfig GlobalConfig, wg *sync.WaitGroup) {
	defer wg.Done()

	natsURL := GetStringSetting(producer.Settings, "nats_url", globalConfig.NatsURL)
	// adminURL := GetStringSetting(producer.Settings, "admin_url", globalConfig.AdminURL)
	// accessKey := GetStringSetting(producer.Settings, "access_key", globalConfig.AccessKey)
	// secretKey := GetStringSetting(producer.Settings, "secret_key", globalConfig.SecretKey)
	nodeName := GetStringSetting(producer.Settings, "node_name", globalConfig.NodeName)
	instanceID := GetStringSetting(producer.Settings, "instance_id", globalConfig.InstanceID)

	switch producer.Type {
	case "bucket_notify":
		natsSubject := GetStringSetting(producer.Settings, "nats_subject", "rgw.buckets.notify")
		endpointPort := GetIntSetting(producer.Settings, "endpoint_port", 8080)
		settings := bucketnotify.BucketNotifyConfig{
			EndpointPort: endpointPort,
			NatsURL:      natsURL,
			NatsSubject:  natsSubject,
			UseNats:      natsURL != "",
		}
		log.Info().Msg("--- bucket notify ---")
		bucketnotify.StartBucketNotifyServer(settings)
	case "disk_health_metrics":
		log.Info().Msg("--- disk health metrics ---")
	case "kernel_metrics":
		log.Info().Msg("--- kernel metrics ---")
	case "resource_usage":
		natsSubject := GetStringSetting(producer.Settings, "nats_subject", "rgw.buckets.notify")
		prometheus := GetBoolSetting(producer.Settings, "prometheus", false)
		prometheusPort := GetIntSetting(producer.Settings, "endpoint_port", 8080)
		interval := GetIntSetting(producer.Settings, "interval", 30)
		disks := GetStringSliceSetting(producer.Settings, "disks", []string{"sda", "sdb"})
		settings := resourceusage.ResourceUsageConfig{
			NatsURL:        natsURL,
			NatsSubject:    natsSubject,
			UseNats:        natsURL != "",
			Prometheus:     prometheus,
			PrometheusPort: prometheusPort,
			Interval:       interval,
			Disks:          disks,
			NodeName:       nodeName,
			InstanceID:     instanceID,
		}
		log.Info().Msg("--- resource usage ---")
		resourceusage.StartMonitoring(settings)
	// case "quota_usage":
	// 	natsSubject := GetStringSetting(producer.Settings, "nats_subject", "rgw.buckets.notify")
	// 	interval := GetIntSetting(producer.Settings, "monitor_interval", 30)
	// 	quotaUsagePercent := GetFloat64Setting(producer.Settings, "quota_usage_percent", 80)
	// 	settings := quotausagemonitor.QuotaUsageMonitorConfig{
	// 		NatsURL:           natsURL,
	// 		NatsSubject:       natsSubject,
	// 		UseNats:           natsURL != "",
	// 		AdminURL:          adminURL,
	// 		AccessKey:         accessKey,
	// 		SecretKey:         secretKey,
	// 		Interval:          interval,
	// 		NodeName:          nodeName,
	// 		InstanceID:        instanceID,
	// 		QuotaUsagePercent: quotaUsagePercent,
	// 	}
	// 	quotausagemonitor.StartMonitoring(settings)
	default:
		log.Warn().Msgf("unknown producer type: %s", producer.Type)
	}
}
