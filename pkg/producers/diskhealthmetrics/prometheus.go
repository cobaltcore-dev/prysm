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

import (
	"fmt"
	"net/http"

	"github.com/rs/zerolog/log"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	smartAttributesGaugeVec = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "smart_attributes",
			Help: "SMART attributes of the disk",
		},
		[]string{"disk", "attribute", "node", "instance"},
	)

	temperatureGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "disk_temperature_celsius",
			Help: "Disk temperature in Celsius",
		},
		[]string{"disk", "node", "instance"},
	)

	reallocatedSectorsGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "disk_reallocated_sectors",
			Help: "Number of reallocated sectors",
		},
		[]string{"disk", "node", "instance"},
	)

	pendingSectorsGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "disk_pending_sectors",
			Help: "Number of pending sectors",
		},
		[]string{"disk", "node", "instance"},
	)

	powerOnHoursGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "disk_power_on_hours",
			Help: "Number of hours the disk has been powered on",
		},
		[]string{"disk", "node", "instance"},
	)

	ssdLifeUsedGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "ssd_life_used_percentage",
			Help: "Percentage of SSD life used",
		},
		[]string{"disk", "node", "instance"},
	)

	errorCountsGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "disk_error_counts",
			Help: "Various error counts for the disk",
		},
		[]string{"disk", "node", "instance", "error_type"},
	)

	diskCapacityGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "disk_capacity_gb",
			Help: "Capacity of the disk in GB",
		},
		[]string{"disk", "node", "instance"},
	)
)

func init() {
	// Register all metrics with Prometheus's default registry
	prometheus.MustRegister(smartAttributesGaugeVec)
	prometheus.MustRegister(temperatureGauge)
	prometheus.MustRegister(reallocatedSectorsGauge)
	prometheus.MustRegister(pendingSectorsGauge)
	prometheus.MustRegister(powerOnHoursGauge)
	prometheus.MustRegister(ssdLifeUsedGauge)
	prometheus.MustRegister(errorCountsGauge)
	prometheus.MustRegister(diskCapacityGauge)
}

// PublishToPrometheus publishes the SMART data to Prometheus
func PublishToPrometheus(metrics []NormalizedSmartData, cfg DiskHealthMetricsConfig) {
	for _, metric := range metrics {
		if metric.TemperatureCelsius != nil {
			temperatureGauge.With(prometheus.Labels{
				"disk":     metric.Device,
				"node":     metric.NodeName,
				"instance": metric.InstanceID,
			}).Set(float64(*metric.TemperatureCelsius))
		}

		if metric.ReallocatedSectors != nil {
			reallocatedSectorsGauge.With(prometheus.Labels{
				"disk":     metric.Device,
				"node":     metric.NodeName,
				"instance": metric.InstanceID,
			}).Set(float64(*metric.ReallocatedSectors))
		}

		if metric.PendingSectors != nil {
			pendingSectorsGauge.With(prometheus.Labels{
				"disk":     metric.Device,
				"node":     metric.NodeName,
				"instance": metric.InstanceID,
			}).Set(float64(*metric.PendingSectors))
		}

		if metric.PowerOnHours != nil {
			powerOnHoursGauge.With(prometheus.Labels{
				"disk":     metric.Device,
				"node":     metric.NodeName,
				"instance": metric.InstanceID,
			}).Set(float64(*metric.PowerOnHours))
		}

		if metric.SSDLifeUsed != nil {
			ssdLifeUsedGauge.With(prometheus.Labels{
				"disk":     metric.Device,
				"node":     metric.NodeName,
				"instance": metric.InstanceID,
			}).Set(float64(*metric.SSDLifeUsed))
		}

		diskCapacityGauge.With(prometheus.Labels{
			"disk":     metric.Device,
			"node":     metric.NodeName,
			"instance": metric.InstanceID,
		}).Set(float64(metric.CapacityGB))

		for errorType, count := range metric.ErrorCounts {
			errorCountsGauge.With(prometheus.Labels{
				"disk":       metric.Device,
				"node":       metric.NodeName,
				"instance":   metric.InstanceID,
				"error_type": errorType,
			}).Set(float64(count))
		}

		for attrName, attrValue := range metric.Attributes {
			smartAttributesGaugeVec.With(prometheus.Labels{
				"disk":      metric.Device,
				"attribute": attrName,
				"node":      metric.NodeName,
				"instance":  metric.InstanceID,
			}).Set(float64(attrValue.RawValue))
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
