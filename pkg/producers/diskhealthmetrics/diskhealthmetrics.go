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
		// rawData, err := collectSmartDataFromFile("../mat/devicehealth/nvme0.json")
		// rawData, err := collectSmartDataFromFile("../mat/devicehealth/sdl.json")
		if err != nil {
			log.Error().Err(err).Str("disk", disk).Msg("error running smartctl")
			continue
		}

		// Normalize the device information
		deviceInfo := &DeviceInfo{}
		FillDeviceInfoFromSmartData(deviceInfo, rawData)
		NormalizeVendor(deviceInfo)
		NormalizeDeviceInfo(deviceInfo)

		// Normalize Smart Attributes
		smartAttrs := GetSmartAttributes()
		ProcessAndUpdateSmartAttributes(smartAttrs, rawData)
		CleanupSmartAttributes(smartAttrs)

		//FIXME: just for debug Print out the updated smartAttrs
		// for key, attr := range smartAttrs {
		// 	fmt.Printf("%s: %s (Unit: %s, Value: %d, Threshold: %d, Worst: %d, Raw: %d)\n", key, attr.Description, attr.Unit, attr.Value, attr.Threshold, attr.Worst, attr.RawValue)
		// }

		// Normalize the data
		normalizedData := normalizeSmartData(rawData, deviceInfo, smartAttrs, cfg.NodeName, cfg.InstanceID)

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

	var reallocatedSectors, pendingSectors, powerOnHours *int64
	var udmaCrcErrorCount int64

	// Initialize device-specific attributes
	if smartData.Device.Protocol == "ATA" && smartData.ATASMARTAttributes != nil {
		reallocatedSectors = &findSmartAttributeByID(smartData.ATASMARTAttributes.Table, 5).Raw.Value
		pendingSectors = &findSmartAttributeByID(smartData.ATASMARTAttributes.Table, 197).Raw.Value
		udmaCrcErrorCount = findSmartAttributeByID(smartData.ATASMARTAttributes.Table, 199).Raw.Value
		powerOnHours = &smartData.PowerOnTime.Hours
	} else if smartData.Device.Protocol == "SCSI" && smartData.SCSIStartStopCycleCounter != nil {
		powerOnHours = &smartData.PowerOnTime.Hours
		reallocatedSectors = &smartData.SCSIGrownDefectList
	}

	return NormalizedSmartData{
		NodeName:           nodeName,
		InstanceID:         instanceID,
		Device:             smartData.Device.Name,
		DeviceInfo:         deviceInfo,
		CapacityGB:         capacityGB,
		TemperatureCelsius: temperatureCelsius,
		ReallocatedSectors: reallocatedSectors,
		PendingSectors:     pendingSectors,
		PowerOnHours:       powerOnHours,
		SSDLifeUsed:        ssdLifeUsed,
		ErrorCounts: map[string]int64{
			"UDMA_CRC_Error_Count": udmaCrcErrorCount,
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

	// Discover devices if wildcard (*) is used in the configuration.
	if len(cfg.Disks) == 1 && cfg.Disks[0] == "*" {
		devices, err := discoverDevices()
		if err != nil {
			log.Fatal().Err(err).Msg("Error discovering devices")
		}

		cfg.Disks = make([]string, len(devices.Devices))
		for i, device := range devices.Devices {
			cfg.Disks[i] = device.Name
		}
	}

	// Ensure that at least one device is found, log a fatal error otherwise.
	if len(cfg.Disks) == 0 {
		log.Fatal().Msg("No devices found for monitoring.")
	}

	// Log the list of devices to be monitored.
	log.Info().Strs("Devices", cfg.Disks).Msg("Devices for monitoring")

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
