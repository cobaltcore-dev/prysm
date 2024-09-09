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

type QuotaUsage struct {
	UserID         string `json:"user_id"`
	TotalQuota     uint64 `json:"total_quota"`
	UsedQuota      uint64 `json:"used_quota"`
	RemainingQuota uint64 `json:"remaining_quota"`
	NodeName       string `json:"node_name"`
	InstanceID     string `json:"instance_id"`
}

func StartQuotaUsageConsumer(cfg QuotaUsageConsumerConfig) {

	if cfg.Prometheus {
		StartPrometheusServer(cfg.PrometheusPort)
	}

	StartNatsConsumer(cfg)
}
