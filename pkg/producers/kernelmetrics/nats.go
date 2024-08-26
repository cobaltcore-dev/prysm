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

package kernelmetrics

import (
	"encoding/json"

	"github.com/nats-io/nats.go"
)

func PublishToNATS(nc *nats.Conn, metrics KernelMetrics, cfg KernelMetricsConfig) error {
	data, err := json.Marshal(metrics)
	if err != nil {
		return err
	}

	return nc.Publish(cfg.NatsSubject, data)
}
