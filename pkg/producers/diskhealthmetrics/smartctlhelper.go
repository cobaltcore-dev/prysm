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
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/rs/zerolog/log"
)

func checkSmartctlInstalled() bool {
	_, err := exec.LookPath("smartctl")
	return err == nil
}

// discoverDevices discovers all devices capable of SMART monitoring
func discoverDevices() (*SmartCtlScanOutput, error) {
	// Execute the smartctl command to scan for devices
	out, err := exec.Command("smartctl", "--scan-open", "-j").Output()
	if err != nil {
		return nil, fmt.Errorf("error running smartctl --scan-open: %v", err)
	}

	// Parse the JSON output into the SmartCtlScanOutput struct
	var scanOutput SmartCtlScanOutput
	if err := json.Unmarshal(out, &scanOutput); err != nil {
		return nil, fmt.Errorf("error parsing JSON: %v", err)
	}

	return &scanOutput, nil
}

// collectSmartData collects SMART data for a specific device using smartctl --json --info --health --attributes --tolerance=verypermissive --nocheck=standby --format=brief --log=error
func collectSmartData(devicePath string) (*SmartCtlOutput, error) {
	// Execute the smartctl command to get extended JSON output
	out, err := exec.Command("smartctl", "--json", "--info", "--health", "--attributes", "--tolerance=verypermissive", "--nocheck=standby", "--format=brief", "--log=error", devicePath).Output()
	if err != nil {
		return nil, fmt.Errorf("error running smartctl: %v", err)
	}

	// Parse the JSON output into the SmartCtlOutput struct
	var smartData SmartCtlOutput
	if err := json.Unmarshal(out, &smartData); err != nil {
		return nil, fmt.Errorf("error parsing JSON: %v", err)
	}

	return &smartData, nil
}

// for tests only
func collectSmartDataFromFile(filePath string) (*SmartCtlOutput, error) {
	// Read the file content
	out, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("error reading file: %v", err)
	}

	// Parse the JSON output into the SmartCtlOutput struct
	var smartData SmartCtlOutput
	if err := json.Unmarshal(out, &smartData); err != nil {
		return nil, fmt.Errorf("error parsing JSON: %v", err)
	}

	return &smartData, nil
}

// ////
// ####
// ///

func ProcessAndUpdateSmartAttributes(smartAttrs map[string]SmartAttribute, smartCtlOutput *SmartCtlOutput) {
	// Process ATA-specific attributes
	if smartCtlOutput.ATASMARTAttributes != nil {
		ProcessAndUpdateATASmartAttributes(smartAttrs, smartCtlOutput)
	}

	// Process SCSI-specific attributes
	ProcessAndUpdateSCSISmartAttributes(smartAttrs, smartCtlOutput)

	// Process NVMe-specific attributes
	ProcessAndUpdateNVMeSmartAttributes(smartAttrs, smartCtlOutput)
}

// Process and update ATA-specific SMART attributes
func ProcessAndUpdateATASmartAttributes(smartAttrs map[string]SmartAttribute, smartCtlOutput *SmartCtlOutput) {
	for _, entry := range smartCtlOutput.ATASMARTAttributes.Table {
		// Normalize the attribute name and resolve using alias map
		attrName := strings.ToLower(entry.Name)
		if resolvedName, found := aliasMap[attrName]; found {
			attrName = resolvedName
		}

		// If the attribute exists in smartAttrs, update its values
		if attr, found := smartAttrs[attrName]; found {
			// Process special cases like percentage-based attributes
			switch attrName {
			case "media_wearout_indicator", "percent_life_remaining", "percent_lifetime_remain":
				// Special handling for percentage-based attributes
				percentageUsed := calculatePercentageUsed(entry.Value)
				attr.Value = percentageUsed
			default:
				// General attribute processing
				attr.Value = entry.Value
				attr.Worst = entry.Worst
				attr.Threshold = entry.Thresh
				attr.RawValue = entry.Raw.Value
			}
			smartAttrs[attrName] = attr
		} else {
			// Attribute not found in smartAttrs, log for debugging
			log.Warn().Str("attribute_name", attrName).Msg("Unrecognized ATA SMART attribute")
		}
	}
}

// Process and update SCSI-specific SMART attributes
func ProcessAndUpdateSCSISmartAttributes(smartAttrs map[string]SmartAttribute, output *SmartCtlOutput) {
	// Update power-on hours
	updateAttributeFromValue(smartAttrs, "power_on_hours", output.PowerOnTime.Hours, output.PowerOnTime.Hours, -1, -1, "hours")

	// Update temperature, adding a sanity check for 0°C or less
	if output.Temperature.Current > 0 {
		updateAttributeFromValue(smartAttrs, "temperature_celsius", output.Temperature.Current, output.Temperature.Current, -1, -1, "Celsius")
	} else {
		log.Warn().Msgf("Unexpected temperature value: %d°C for SCSI device", output.Temperature.Current)
	}

	// Update power cycle count, only if the data exists
	if output.SCSIStartStopCycleCounter != nil {
		updateAttributeFromValue(smartAttrs, "power_cycle_count", output.SCSIStartStopCycleCounter.AccumulatedStartStopCycles, output.SCSIStartStopCycleCounter.AccumulatedStartStopCycles, -1, -1, "count")
	}

	// Update grown defects count
	if output.SCSIGrownDefectList >= 0 {
		updateAttributeFromValue(smartAttrs, "grown_defects_count", output.SCSIGrownDefectList, output.SCSIGrownDefectList, -1, -1, "count")
	} else {
		log.Warn().Msgf("Invalid grown defects count: %d for SCSI device", output.SCSIGrownDefectList)
	}

	// Update SCSI error log counters
	if output.SCSIErrorCounterLog != nil {
		updateSCSIErrorLog(smartAttrs, output.SCSIErrorCounterLog)
	}
}

// Update SCSI error log attributes
func updateSCSIErrorLog(smartAttrs map[string]SmartAttribute, log *SmartCtlSCSIErrorCounterLog) {
	// Update read errors corrected
	updateAttributeFromValue(smartAttrs, "read_errors_corrected", log.Read.TotalErrorsCorrected, log.Read.TotalErrorsCorrected, -1, -1, "count")

	// Update write errors corrected
	updateAttributeFromValue(smartAttrs, "write_errors_corrected", log.Write.TotalErrorsCorrected, log.Write.TotalErrorsCorrected, -1, -1, "count")

	// Update verify errors corrected
	updateAttributeFromValue(smartAttrs, "verify_errors_corrected", log.Verify.TotalErrorsCorrected, log.Verify.TotalErrorsCorrected, -1, -1, "count")

	// Update read gigabytes processed
	updateAttributeFromValue(smartAttrs, "read_gigabytes_processed", parseGigabytes(log.Read.GigabytesProcessed), parseGigabytes(log.Read.GigabytesProcessed), -1, -1, "GB")

	// Update write gigabytes processed
	updateAttributeFromValue(smartAttrs, "write_gigabytes_processed", parseGigabytes(log.Write.GigabytesProcessed), parseGigabytes(log.Write.GigabytesProcessed), -1, -1, "GB")

	// Handle total uncorrected errors for read, write, and verify
	updateAttributeFromValue(smartAttrs, "total_uncorrected_read_errors", log.Read.TotalUncorrectedErrors, log.Read.TotalUncorrectedErrors, -1, -1, "count")
	updateAttributeFromValue(smartAttrs, "total_uncorrected_write_errors", log.Write.TotalUncorrectedErrors, log.Write.TotalUncorrectedErrors, -1, -1, "count")
	updateAttributeFromValue(smartAttrs, "total_uncorrected_verify_errors", log.Verify.TotalUncorrectedErrors, log.Verify.TotalUncorrectedErrors, -1, -1, "count")
}

// Process and update NVMe-specific SMART attributes
func ProcessAndUpdateNVMeSmartAttributes(smartAttrs map[string]SmartAttribute, output *SmartCtlOutput) {
	if output.NVMeSmartHealthInfoLog == nil {
		return
	}

	// Process power-on hours
	updateAttributeFromValue(smartAttrs, "power_on_hours", output.NVMeSmartHealthInfoLog.PowerOnHours, output.NVMeSmartHealthInfoLog.PowerOnHours, -1, -1, "hours")

	// Process temperature
	if output.NVMeSmartHealthInfoLog.Temperature > 0 {
		updateAttributeFromValue(smartAttrs, "temperature_celsius", output.NVMeSmartHealthInfoLog.Temperature, output.NVMeSmartHealthInfoLog.Temperature, -1, -1, "Celsius")
	} else {
		log.Warn().Msgf("Unexpected temperature value: %d°C for NVMe device", output.NVMeSmartHealthInfoLog.Temperature)
	}

	// Process power cycles
	updateAttributeFromValue(smartAttrs, "power_cycle_count", output.NVMeSmartHealthInfoLog.PowerCycles, output.NVMeSmartHealthInfoLog.PowerCycles, -1, -1, "count")

	// Process unsafe shutdowns
	updateAttributeFromValue(smartAttrs, "unsafe_shutdowns", output.NVMeSmartHealthInfoLog.UnsafeShutdowns, output.NVMeSmartHealthInfoLog.UnsafeShutdowns, -1, -1, "count")

	// Process host read commands
	updateAttributeFromValue(smartAttrs, "host_read_commands", output.NVMeSmartHealthInfoLog.HostReads, output.NVMeSmartHealthInfoLog.HostReads, -1, -1, "commands")

	// Process host write commands
	updateAttributeFromValue(smartAttrs, "host_write_commands", output.NVMeSmartHealthInfoLog.HostWrites, output.NVMeSmartHealthInfoLog.HostWrites, -1, -1, "commands")

	// Process controller busy time
	updateAttributeFromValue(smartAttrs, "controller_busy_time", output.NVMeSmartHealthInfoLog.ControllerBusyTime, output.NVMeSmartHealthInfoLog.ControllerBusyTime, -1, -1, "minutes")

	// Process error information log entries
	updateAttributeFromValue(smartAttrs, "error_information_log_entries", output.NVMeSmartHealthInfoLog.NumErrLogEntries, output.NVMeSmartHealthInfoLog.NumErrLogEntries, -1, -1, "count")

	// Process percentage used, adding a sanity check
	if output.NVMeSmartHealthInfoLog.PercentageUsed >= 0 && output.NVMeSmartHealthInfoLog.PercentageUsed <= 100 {
		updateAttributeFromValue(smartAttrs, "percentage_used", output.NVMeSmartHealthInfoLog.PercentageUsed, output.NVMeSmartHealthInfoLog.PercentageUsed, -1, -1, "percent")
	} else {
		log.Warn().Msgf("Unexpected percentage used value: %d for NVMe device", output.NVMeSmartHealthInfoLog.PercentageUsed)
	}

	// Process available spare
	updateAttributeFromValue(smartAttrs, "available_spare", output.NVMeSmartHealthInfoLog.AvailableSpare, output.NVMeSmartHealthInfoLog.AvailableSpare, -1, -1, "percent")

	// Process available spare threshold
	updateAttributeFromValue(smartAttrs, "available_spare_threshold", output.NVMeSmartHealthInfoLog.AvailableSpareThreshold, output.NVMeSmartHealthInfoLog.AvailableSpareThreshold, -1, -1, "percent")

	// Process media and data integrity errors
	updateAttributeFromValue(smartAttrs, "media_and_data_integrity_errors", output.NVMeSmartHealthInfoLog.MediaErrors, output.NVMeSmartHealthInfoLog.MediaErrors, -1, -1, "count")
}

// Helper function to update attributes by resolving alias and updating values
func updateAttributeFromValue(smartAttrs map[string]SmartAttribute, attrName string, value int64, rawValue int64, threshold int64, worst int64, unit string) {
	// Resolve the attribute name using the alias map
	if resolvedName, found := aliasMap[attrName]; found {
		attrName = resolvedName
	}

	// If the attribute exists in the map, update its fields
	if attr, found := smartAttrs[attrName]; found {
		attr.Value = value
		attr.RawValue = rawValue

		// Update threshold if provided
		if threshold != -1 {
			attr.Threshold = threshold
		}

		// Update worst value if provided
		if worst != -1 {
			attr.Worst = worst
		}

		// Update unit if provided and it's different from the current one
		if unit != "" && attr.Unit != unit {
			attr.Unit = unit
		}

		smartAttrs[attrName] = attr
	} else {
		// Log a warning if the attribute was not found in smartAttrs
		log.Warn().Str("attribute_name", attrName).Msg("Unrecognized SMART attribute")
	}
}

// Helper function to parse gigabytes from string
func parseGigabytes(value string) int64 {
	gb, _ := strconv.ParseFloat(value, 64)
	return int64(gb)
}

// Helper function to calculate percentage used
func calculatePercentageUsed(attrValue int64) int64 {
	return 100 - attrValue
}
