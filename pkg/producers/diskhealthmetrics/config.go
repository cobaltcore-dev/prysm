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

package diskhealthmetrics

type DiskHealthMetricsConfig struct {
	NatsURL           string
	NatsSubject       string
	UseNats           bool
	Prometheus        bool
	PrometheusPort    int
	AllAttributes     bool
	Disks             []string
	IncludeZeroValues bool
	Interval          int // in seconds
	NodeName          string
	InstanceID        string

	// NATS event thresholds
	GrownDefectsThreshold       int64
	PendingSectorsThreshold     int64
	ReallocatedSectorsThreshold int64
	LifetimeUsedThreshold       int64 // percentage
}
