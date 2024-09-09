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

import (
	"encoding/json"
	"fmt"

	"github.com/nats-io/nats.go"
)

// convertToNatsEvent converts NormalizedSmartData to a NatsEvent
func convertToNatsEvent(normalizedData NormalizedSmartData, config *DiskHealthMetricsConfig) NatsEvent {
	details := make(map[string]string)

	severity := "info"
	eventType := "health"

	// Normalize SSD wear attributes
	normalizedWear := normalizeSSDWear(normalizedData.Attributes)
	if normalizedWear != nil {
		details["SSDWearPercentage"] = fmt.Sprintf("%d", *normalizedWear)
	}

	if normalizedData.TemperatureCelsius != nil {
		details["TemperatureCelsius"] = fmt.Sprintf("%d", *normalizedData.TemperatureCelsius)
	}
	if normalizedData.ReallocatedSectors != nil {
		details["ReallocatedSectors"] = fmt.Sprintf("%d", *normalizedData.ReallocatedSectors)
	}
	if normalizedData.PendingSectors != nil {
		details["PendingSectors"] = fmt.Sprintf("%d", *normalizedData.PendingSectors)
	}
	if normalizedData.PowerOnHours != nil {
		details["PowerOnHours"] = fmt.Sprintf("%d", *normalizedData.PowerOnHours)
	}
	if normalizedData.SSDLifeUsed != nil {
		details["SSDLifeUsed"] = fmt.Sprintf("%d", *normalizedData.SSDLifeUsed)
	}

	// Handle critical SMART metrics with thresholds
	checkAndSetThresholds(&details, normalizedData, config, &severity, &eventType)

	// Add other SMART attributes
	for attrName, attrValue := range normalizedData.Attributes {
		// Ensure non-duplicate entry for normalized wear
		if attrName != "SSDWearPercentage" {
			details[attrName] = fmt.Sprintf("%d", attrValue.RawValue)
		}
	}

	return NatsEvent{
		NodeName:   normalizedData.NodeName,
		InstanceID: normalizedData.InstanceID,
		Device:     normalizedData.Device,
		EventType:  eventType,
		Severity:   severity,
		Message:    generateMessage(details),
		Details:    details,
	}
}

// normalizeSSDWear normalizes different SSD wear attribute labels into a single percentage used metric.
func normalizeSSDWear(attributes map[string]SmartAttribute) *int64 {
	wearAttributes := []string{
		"media_wearout_indicator",
		"wear_leveling_count",
		"perc_rated_life_used",
		"percent_life_used",
		"ssd_life_left_perc",
		"drive_life_used",
		"lifetime_used",
		"percent_lifetime_used",
	}

	for _, attr := range wearAttributes {
		if value, found := attributes[attr]; found {
			// Normalize to percentage used (e.g., 100 - remaining life)
			normalizedValue := 100 - value.RawValue
			return &normalizedValue
		}
	}
	return nil
}

// checkAndSetThresholds checks critical SMART metrics against thresholds and adjusts details and severity.
func checkAndSetThresholds(details *map[string]string, normalizedData NormalizedSmartData, config *DiskHealthMetricsConfig, severity *string, eventType *string) {
	// Grown defects
	if normalizedData.Attributes["grown_defects_count"].RawValue > config.GrownDefectsThreshold {
		(*details)["GrownDefects"] = fmt.Sprintf("%d (Warning: Exceeds threshold of %d)", normalizedData.Attributes["grown_defects_count"].RawValue, config.GrownDefectsThreshold)
		*severity = "warning"
		*eventType = "health_alert"
	}

	// Pending sectors
	if normalizedData.Attributes["current_pending_sector"].RawValue > config.PendingSectorsThreshold {
		(*details)["PendingSectors"] = fmt.Sprintf("%d (Warning: Exceeds threshold of %d)", normalizedData.Attributes["current_pending_sector"].RawValue, config.PendingSectorsThreshold)
		*severity = "warning"
		*eventType = "health_alert"
	}

	// Reallocated sectors
	if normalizedData.Attributes["reallocated_sector_ct"].RawValue > config.ReallocatedSectorsThreshold {
		(*details)["ReallocatedSectors"] = fmt.Sprintf("%d (Warning: Exceeds threshold of %d)", normalizedData.Attributes["reallocated_sector_ct"].RawValue, config.ReallocatedSectorsThreshold)
		*severity = "warning"
		*eventType = "health_alert"
	}

	// Lifetime used for SSDs
	if ssdWear := normalizeSSDWear(normalizedData.Attributes); ssdWear != nil && *ssdWear > config.LifetimeUsedThreshold {
		(*details)["SSDLifeUsed"] = fmt.Sprintf("%d%% (Warning: Exceeds threshold of %d%%)", *ssdWear, config.LifetimeUsedThreshold)
		*severity = "critical"
		*eventType = "lifetime_alert"
	}
}

// generateMessage generates a summary message based on the details.
func generateMessage(details map[string]string) string {
	if _, found := details["GrownDefects"]; found {
		return "SMART data indicates potential drive issues (grown defects)."
	}
	if _, found := details["PendingSectors"]; found {
		return "SMART data indicates potential drive issues (pending sectors)."
	}
	if _, found := details["ReallocatedSectors"]; found {
		return "SMART data indicates potential drive issues (reallocated sectors)."
	}
	if _, found := details["SSDLifeUsed"]; found {
		return "SMART data indicates SSD nearing end of life."
	}
	return "SMART data collected successfully."
}

func PublishToNATS(metrics []NormalizedSmartData, nc *nats.Conn, subject string, cfg *DiskHealthMetricsConfig) error {
	for _, metric := range metrics {
		event := convertToNatsEvent(metric, cfg)

		eventJSON, err := json.Marshal(event)
		if err != nil {
			return err
		}

		if err := nc.Publish(subject, eventJSON); err != nil {
			return err
		}
	}

	return nil
}
