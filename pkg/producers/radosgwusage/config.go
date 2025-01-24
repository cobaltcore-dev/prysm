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

package radosgwusage

type RadosGWUsageConfig struct {
	AdminURL                string
	AccessKey               string
	SecretKey               string
	NatsURL                 string // For exporting metrics
	NatsSubject             string // For exporting metrics
	UseNats                 bool   // Indicates if NATS is used for metrics export
	Prometheus              bool
	PrometheusPort          int
	NodeName                string
	InstanceID              string
	Interval                int // in seconds
	ClusterID               string
	SyncControlNats         bool   // Enable NATS for sync control
	SyncExternalNats        bool   // Use external NATS for sync control
	SyncControlURL          string // URL for the external NATS server (if applicable)
	SyncControlBucketPrefix string // NATS-KV bucket prefix for sync data
}
