// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and prysm contributors
//
// SPDX-License-Identifier: Apache-2.0

package diskhealthmetrics

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog/log"
)

var osdIDCache = make(map[string]string)

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
		normalizedData := normalizeSmartData(rawData, deviceInfo, smartAttrs, cfg.NodeName, cfg.InstanceID, cfg.CephOSDBasePath)

		allMetrics = append(allMetrics, normalizedData)
	}

	return allMetrics
}

// normalizeSmartData normalizes the raw SMART data
func normalizeSmartData(smartData *SmartCtlOutput, deviceInfo *DeviceInfo, attributes map[string]SmartAttribute, nodeName, instanceID, basePath string) NormalizedSmartData {
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

	osdID, _ := getOSDIDForDisk(smartData.Device.Name, basePath) // Ignore error as it's handled within the function
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
		OSDID:      osdID, // This may be an empty string if OSD ID is not applicable or retrievable
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

// getOSDIDForDisk attempts to retrieve the OSD ID from a specified path. If the path does not exist,
// it logs a warning and returns an empty string, allowing the application to function in non-Ceph environments.
func getOSDIDForDisk(disk, basePath string) (string, error) {
	if osdID, found := osdIDCache[disk]; found {
		return osdID, nil
	}

	// Find the OSD directory that corresponds to this disk
	osdID, err := findOSDIDForDisk(disk, basePath)
	if err != nil {
		log.Warn().Err(err).Str("disk", disk).Str("base_path", basePath).Msg("OSD ID not found for disk, continuing without OSD ID")
		return "", nil
	}

	if osdID == "" {
		log.Debug().Str("disk", disk).Msg("No OSD ID found for disk")
		return "", nil
	}

	osdIDCache[disk] = osdID
	return osdID, nil
}

func findOSDIDForDisk(disk, basePath string) (string, error) {
	// Read all directories in the base path
	entries, err := os.ReadDir(basePath)
	if err != nil {
		return "", fmt.Errorf("failed to read base path %s: %w", basePath, err)
	}

	// Resolve the canonical path of the target disk
	targetDisk, err := filepath.EvalSymlinks(disk)
	if err != nil {
		targetDisk = disk // fallback to original if can't resolve
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		// Check if this directory matches the UUID_UUID pattern (contains underscore)
		if !strings.Contains(entry.Name(), "_") {
			continue
		}

		dirPath := filepath.Join(basePath, entry.Name())

		// Check if this directory contains a block file/symlink
		blockPath := filepath.Join(dirPath, "block")
		if _, err := os.Stat(blockPath); err != nil {
			continue // Skip if no block file/symlink
		}

		// Resolve the block symlink to get the actual device (equivalent to readlink -f)
		blockDevice, err := filepath.EvalSymlinks(blockPath)
		if err != nil {
			continue // Skip if can't resolve symlink
		}

		// Check if this matches our target disk
		if blockDevice == targetDisk || blockDevice == disk {
			// Read the whoami file to get the OSD ID
			whoamiPath := filepath.Join(dirPath, "whoami")
			osdIDBytes, err := os.ReadFile(whoamiPath)
			if err != nil {
				log.Warn().Err(err).Str("whoami_path", whoamiPath).Msg("Failed to read whoami file")
				continue
			}

			return strings.TrimSpace(string(osdIDBytes)), nil
		}
	}

	return "", nil // No matching OSD directory found
}
