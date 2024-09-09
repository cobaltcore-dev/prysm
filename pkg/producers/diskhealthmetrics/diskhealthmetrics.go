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
	"time"

	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog/log"
)

func collectDiskHealthMetrics(cfg DiskHealthMetricsConfig) []NormalizedSmartData {
	var allMetrics []NormalizedSmartData

	for _, disk := range cfg.Disks {
		//FIXME rawData, err := collectSmartData(fmt.Sprintf("/dev/%s", disk))
		rawData, err := collectSmartData(disk)
		// rawData, err := collectSmartDataFromFile("./cmd/smart1/output-A-j.json")
		if err != nil {
			log.Error().Err(err).Str("disk", disk).Msg("error running smartctl")
			continue
		}

		// Normalize the device information
		deviceInfo := DeviceInfo{
			ModelFamily:     rawData.ModelFamily,
			DeviceModel:     rawData.DeviceModel, // rawData.ModelName
			SerialNumber:    rawData.SerialNumber,
			FirmwareVersion: rawData.FirmwareVersion,
			Vendor:          rawData.Vendor,
			Product:         rawData.Product,
			LunID:           rawData.LogicalUnitID,
			Capacity:        -1,
			DWPD:            0,
			RPM:             0,
			FormFactor:      "none",
			Media:           "unknown",
		}
		NormalizeDeviceInfo(&deviceInfo)

		// Normalize Smart Attributes
		smartAttrs := GetSmartAttributes()
		ProcessAndUpdateSmartAttributes(smartAttrs, rawData)
		CleanupSmartAttributes(smartAttrs)

		//FIXME: just for debug Print out the updated smartAttrs
		// for key, attr := range smartAttrs {
		// 	fmt.Printf("%s: %s (Unit: %s, Value: %d, Threshold: %d, Worst: %d, Raw: %d)\n", key, attr.Description, attr.Unit, attr.Value, attr.Threshold, attr.Worst, attr.RawValue)
		// }

		// Normalize the data
		normalizedData := normalizeSmartData(rawData, &deviceInfo, smartAttrs, cfg.NodeName, cfg.InstanceID)

		allMetrics = append(allMetrics, normalizedData)
	}

	return allMetrics
}

// normalizeSmartData normalizes the raw SMART data
func normalizeSmartData(smartData *SmartCtlOutput, deviceInfo *DeviceInfo, attributes map[string]SmartAttribute, nodeName, instanceID string) NormalizedSmartData {
	var temperatureCelsius *int64
	if smartData.Temperature.Current != 0 {
		temperatureCelsius = &smartData.Temperature.Current
	}

	// Calculate capacity from UserCapacity if not already set in DeviceInfo
	var capacityGB float64
	if deviceInfo.Capacity < 0 && smartData.UserCapacity != nil {
		capacityGB = float64(smartData.UserCapacity.Bytes) / (1024 * 1024 * 1024)
		deviceInfo.Capacity = capacityGB
	} else {
		capacityGB = deviceInfo.Capacity
	}

	var ssdLifeUsed *int64
	if smartData.NVMeSmartHealthInfoLog != nil {
		ssdLifeUsed = &smartData.NVMeSmartHealthInfoLog.PercentageUsed
	}

	return NormalizedSmartData{
		NodeName:           nodeName,
		InstanceID:         instanceID,
		Device:             smartData.Device.Name,
		DeviceInfo:         deviceInfo,
		CapacityGB:         capacityGB,
		TemperatureCelsius: temperatureCelsius,
		ReallocatedSectors: &findSmartAttributeByID(smartData.ATASMARTAttributes.Table, 5).Raw.Value,
		PendingSectors:     &findSmartAttributeByID(smartData.ATASMARTAttributes.Table, 197).Raw.Value,
		PowerOnHours:       &smartData.PowerOnTime.Hours,
		SSDLifeUsed:        ssdLifeUsed,
		ErrorCounts: map[string]int64{
			"UDMA_CRC_Error_Count": findSmartAttributeByID(smartData.ATASMARTAttributes.Table, 199).Raw.Value,
		},
		Attributes: attributes,
	}
}

func findSmartAttributeByID(attributes []SmartCtlATASMARTEntry, id int64) *SmartCtlATASMARTEntry {
	for _, attr := range attributes {
		if attr.ID == id {
			return &attr
		}
	}
	return nil
}

// func findSmartAttributeByName(attributes []SmartCtlATASMARTEntry, name string) *SmartCtlATASMARTEntry {
// 	for _, attr := range attributes {
// 		if attr.Name == name {
// 			return &attr
// 		}
// 	}
// 	return nil
// }

// func parseSMARTOutput(output []byte, attribute string) uint64 {
// 	lines := strings.Split(string(output), "\n")
// 	for _, line := range lines {
// 		if strings.Contains(line, attribute) {
// 			fields := strings.Fields(line)
// 			value, err := strconv.ParseUint(fields[9], 10, 64)
// 			if err != nil {
// 				log.Printf("Error parsing %s value: %v", attribute, err)
// 				return 0
// 			}
// 			return value
// 		}
// 	}
// 	return 0
// }

func StartMonitoring(cfg DiskHealthMetricsConfig) {
	if !checkSmartctlInstalled() {
		log.Fatal().Msg("smartctl is not installed. please install smartmontools package.")
	}

	var nc *nats.Conn
	var err error
	if cfg.UseNats {
		nc, err = nats.Connect(cfg.NatsURL)
		if err != nil {
			log.Fatal().Err(err).Msg("error connecting to nats")
		}
		defer nc.Close()
	}

	if cfg.Prometheus {
		StartPrometheusServer(cfg.PrometheusPort)
	}

	ticker := time.NewTicker(time.Duration(cfg.Interval) * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		metrics := collectDiskHealthMetrics(cfg)

		if cfg.Prometheus {
			PublishToPrometheus(metrics, cfg)
		}

		if cfg.UseNats {
			err = PublishToNATS(metrics, nc, cfg.NatsSubject, &cfg)
			if err != nil {
				log.Error().Err(err).Msg("error publishing metrics to nats")
			}
		} else {
			metricsJSON, err := json.Marshal(metrics)
			if err != nil {
				log.Error().Err(err).Msg("error marshalling metrics to json")
				continue
			}
			fmt.Println(string(metricsJSON))
		}
	}
}
