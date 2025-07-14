// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and prysm contributors
//
// SPDX-License-Identifier: Apache-2.0

package diskhealthmetrics

import (
	"fmt"
	"net/http"
	"sync"

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
		[]string{"disk", "attribute", "node", "instance", "osd_id"},
	)

	temperatureGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "disk_temperature_celsius",
			Help: "Disk temperature in Celsius",
		},
		[]string{"disk", "node", "instance", "osd_id"},
	)

	reallocatedSectorsGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "disk_reallocated_sectors",
			Help: "Number of reallocated sectors",
		},
		[]string{"disk", "node", "instance", "osd_id"},
	)

	pendingSectorsGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "disk_pending_sectors",
			Help: "Number of pending sectors",
		},
		[]string{"disk", "node", "instance", "osd_id"},
	)

	// Counter for cumulative power-on hours
	powerOnHoursCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "disk_power_on_hours_total",
			Help: "Total number of hours the disk has been powered on",
		},
		[]string{"disk", "node", "instance", "osd_id"},
	)

	ssdLifeUsedGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "ssd_life_used_percentage",
			Help: "Percentage of SSD life used",
		},
		[]string{"disk", "node", "instance", "osd_id"},
	)

	// Counter for cumulative error counts
	errorCountsCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "disk_error_counts_total",
			Help: "Total error counts for the disk",
		},
		[]string{"disk", "node", "instance", "error_type", "osd_id"},
	)

	diskCapacityGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "disk_capacity_gb",
			Help: "Capacity of the disk in GB",
		},
		[]string{"disk", "node", "instance", "osd_id"},
	)

	// State management for counters
	previousValues      = make(map[string]previousMetricState)
	previousValuesMutex sync.RWMutex
)

type previousMetricState struct {
	PowerOnHours int64
	ErrorCounts  map[string]int64
}

func init() {
	// Register all metrics with Prometheus's default registry
	prometheus.MustRegister(smartAttributesGaugeVec)
	prometheus.MustRegister(temperatureGauge)
	prometheus.MustRegister(reallocatedSectorsGauge)
	prometheus.MustRegister(pendingSectorsGauge)
	prometheus.MustRegister(powerOnHoursCounter)
	prometheus.MustRegister(ssdLifeUsedGauge)
	prometheus.MustRegister(errorCountsCounter)
	prometheus.MustRegister(diskCapacityGauge)
}

// PublishToPrometheus publishes the SMART data to Prometheus
func PublishToPrometheus(metrics []NormalizedSmartData, cfg DiskHealthMetricsConfig) {
	for _, metric := range metrics {
		labels := prometheus.Labels{
			"disk":     metric.Device,
			"node":     metric.NodeName,
			"instance": metric.InstanceID,
			"osd_id":   metric.OSDID,
		}

		if metric.TemperatureCelsius != nil {
			temperatureGauge.With(labels).Set(float64(*metric.TemperatureCelsius))
		}

		if metric.ReallocatedSectors != nil {
			reallocatedSectorsGauge.With(labels).Set(float64(*metric.ReallocatedSectors))
		}

		if metric.PendingSectors != nil {
			pendingSectorsGauge.With(labels).Set(float64(*metric.PendingSectors))
		}

		if metric.PowerOnHours != nil {
			updatePowerOnHoursCounter(metric.Device, *metric.PowerOnHours, labels)
		}

		if metric.SSDLifeUsed != nil {
			ssdLifeUsedGauge.With(labels).Set(float64(*metric.SSDLifeUsed))
		}

		diskCapacityGauge.With(labels).Set(metric.CapacityGB)

		for errorType, count := range metric.ErrorCounts {
			errorLabels := prometheus.Labels{
				"disk":       metric.Device,
				"node":       metric.NodeName,
				"instance":   metric.InstanceID,
				"error_type": errorType,
				"osd_id":     metric.OSDID,
			}
			updateErrorCountsCounter(metric.Device, errorType, count, errorLabels)
		}

		for attrName, attrValue := range metric.Attributes {
			attrLabels := prometheus.Labels{
				"disk":      metric.Device,
				"attribute": attrName,
				"node":      metric.NodeName,
				"instance":  metric.InstanceID,
				"osd_id":    metric.OSDID,
			}
			smartAttributesGaugeVec.With(attrLabels).Set(float64(attrValue.RawValue))
		}
	}
}

func updatePowerOnHoursCounter(diskKey string, currentValue int64, labels prometheus.Labels) {
	previousValuesMutex.Lock()
	defer previousValuesMutex.Unlock()

	prevState, exists := previousValues[diskKey]
	if !exists {
		previousValues[diskKey] = previousMetricState{
			PowerOnHours: currentValue,
			ErrorCounts:  make(map[string]int64),
		}
		if currentValue > 0 {
			powerOnHoursCounter.With(labels).Add(float64(currentValue))
		}
		return
	}

	if currentValue < prevState.PowerOnHours {
		log.Warn().Msgf("Power on hours decreased for disk %s: %d -> %d (possible disk replacement)",
			diskKey, prevState.PowerOnHours, currentValue)
		if currentValue > 0 {
			powerOnHoursCounter.With(labels).Add(float64(currentValue))
		}
	} else {
		// Normal case: add the delta
		delta := currentValue - prevState.PowerOnHours
		if delta > 0 {
			powerOnHoursCounter.With(labels).Add(float64(delta))
		}
	}

	// Update stored value
	prevState.PowerOnHours = currentValue
	previousValues[diskKey] = prevState
}

func updateErrorCountsCounter(diskKey, errorType string, currentValue int64, labels prometheus.Labels) {
	previousValuesMutex.Lock()
	defer previousValuesMutex.Unlock()

	prevState, exists := previousValues[diskKey]
	if !exists {
		// First time seeing this disk, initialize
		previousValues[diskKey] = previousMetricState{
			PowerOnHours: 0,
			ErrorCounts:  map[string]int64{errorType: currentValue},
		}
		// Set counter to current value for first time (only if positive)
		if currentValue > 0 {
			errorCountsCounter.With(labels).Add(float64(currentValue))
		}
		return
	}

	if prevState.ErrorCounts == nil {
		prevState.ErrorCounts = make(map[string]int64)
	}

	prevValue, exists := prevState.ErrorCounts[errorType]
	if !exists {
		// First time seeing this error type for this disk
		prevState.ErrorCounts[errorType] = currentValue
		if currentValue > 0 {
			errorCountsCounter.With(labels).Add(float64(currentValue))
		}
	} else {
		// Handle potential counter reset
		if currentValue < prevValue {
			log.Warn().Msgf("Error count decreased for disk %s, error type %s: %d -> %d",
				diskKey, errorType, prevValue, currentValue)
			// Reset counter by adding current value
			if currentValue > 0 {
				errorCountsCounter.With(labels).Add(float64(currentValue))
			}
		} else {
			// Normal case: add the delta
			delta := currentValue - prevValue
			if delta > 0 {
				errorCountsCounter.With(labels).Add(float64(delta))
			}
		}
		prevState.ErrorCounts[errorType] = currentValue
	}

	previousValues[diskKey] = prevState
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
