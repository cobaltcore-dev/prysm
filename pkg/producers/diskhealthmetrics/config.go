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
