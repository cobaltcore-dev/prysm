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
	"strings"
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
// Helper function to calculate percentage used
func calculatePercentageUsed(attrValue int64) int64 {
	return 100 - attrValue
}

func ProcessAndUpdateSmartAttributes(smartAttrs map[string]SmartAttribute, smartCtlOutput *SmartCtlOutput) {

	for _, entry := range smartCtlOutput.ATASMARTAttributes.Table {
		// Normalize the attribute name and resolve using alias map
		attrName := strings.ToLower(entry.Name)
		if resolvedName, found := aliasMap[attrName]; found {
			attrName = resolvedName
		}
		// If not found in aliasMap, the original attrName will be used
		if attr, found := smartAttrs[attrName]; found {
			attr.Value = entry.Value
			smartAttrs[attrName] = attr
		}

		// Check if the resolved name exists in smartAttrs
		if attr, found := smartAttrs[attrName]; found {
			// Handle special cases like Media_Wearout_Indicator
			switch attrName {
			case "media_wearout_indicator", "percent_life_remaining", "percent_lifetime_remain":
				percentageUsed := calculatePercentageUsed(entry.Value)
				attr.Value = percentageUsed
			default:
				// General handling: store value, worst, thresh, and raw data
				attr.Value = entry.Value
				attr.Worst = entry.Worst
				attr.Threshold = entry.Thresh
				attr.RawValue = entry.Raw.Value
			}
			smartAttrs[attrName] = attr
		}
	}

	NormalizeVendor(smartCtlOutput)

	// Process SCSI-specific attributes
	ProcessAndUpdateSCSISmartAttributes(smartAttrs, smartCtlOutput)
	// Process NVMe-specific attributes
	ProcessAndUpdateNVMeSmartAttributes(smartAttrs, smartCtlOutput)
}

// Process and update SCSI-specific SMART attributes
func ProcessAndUpdateSCSISmartAttributes(smartAttrs map[string]SmartAttribute, output *SmartCtlOutput) {
	// Update power-on hours
	if output.PowerOnTime.Hours > 0 {
		attrName := "power_on_hours"
		if resolvedName, found := aliasMap[attrName]; found {
			attrName = resolvedName
		}
		if attr, found := smartAttrs[attrName]; found {
			attr.Value = output.PowerOnTime.Hours
			smartAttrs[attrName] = attr
		}
	}

	if output.Temperature.Current > 0 {
		attrName := "temperature_celsius"
		if resolvedName, found := aliasMap[attrName]; found {
			attrName = resolvedName
		}
		if attr, found := smartAttrs[attrName]; found {
			attr.Value = output.Temperature.Current
			smartAttrs[attrName] = attr
		}
	}

	// Update power cycle count
	if output.SCSIStartStopCycleCounter != nil && output.SCSIStartStopCycleCounter.AccumulatedStartStopCycles > 0 {
		attrName := "power_cycle_count"
		if resolvedName, found := aliasMap[attrName]; found {
			attrName = resolvedName
		}
		if attr, found := smartAttrs[attrName]; found {
			attr.Value = output.SCSIStartStopCycleCounter.AccumulatedStartStopCycles
			smartAttrs[attrName] = attr
		}
	}

	// Update grown defects count
	if output.SCSIGrownDefectList > 0 {
		attrName := "grown_defects_count"
		if resolvedName, found := aliasMap[attrName]; found {
			attrName = resolvedName
		}
		if attr, found := smartAttrs[attrName]; found {
			attr.Value = output.SCSIGrownDefectList
			smartAttrs[attrName] = attr
		}
	}
}

// Process and update NVMe-specific SMART attributes
func ProcessAndUpdateNVMeSmartAttributes(smartAttrs map[string]SmartAttribute, output *SmartCtlOutput) {
	if output.NVMeSmartHealthInfoLog == nil {
		return
	}

	// Update power-on hours
	if output.NVMeSmartHealthInfoLog.PowerOnHours > 0 {
		attrName := "power_on_hours"
		if resolvedName, found := aliasMap[attrName]; found {
			attrName = resolvedName
		}
		if attr, found := smartAttrs[attrName]; found {
			attr.Value = output.NVMeSmartHealthInfoLog.PowerOnHours
			smartAttrs[attrName] = attr
		}
	}

	// Update temperature
	if output.NVMeSmartHealthInfoLog.Temperature > 0 {
		attrName := "temperature_celsius"
		if resolvedName, found := aliasMap[attrName]; found {
			attrName = resolvedName
		}
		if attr, found := smartAttrs[attrName]; found {
			attr.Value = output.NVMeSmartHealthInfoLog.Temperature
			smartAttrs[attrName] = attr
		}
	}

	// Update power cycles
	if output.NVMeSmartHealthInfoLog.PowerCycles > 0 {
		attrName := "power_cycle_count"
		if resolvedName, found := aliasMap[attrName]; found {
			attrName = resolvedName
		}
		if attr, found := smartAttrs[attrName]; found {
			attr.Value = output.NVMeSmartHealthInfoLog.PowerCycles
			smartAttrs[attrName] = attr
		}
	}

	// Update unsafe shutdowns
	if output.NVMeSmartHealthInfoLog.UnsafeShutdowns > 0 {
		attrName := "unsafe_shutdowns"
		if resolvedName, found := aliasMap[attrName]; found {
			attrName = resolvedName
		}
		if attr, found := smartAttrs[attrName]; found {
			attr.Value = output.NVMeSmartHealthInfoLog.UnsafeShutdowns
			smartAttrs[attrName] = attr
		}
	}

	// Update host read commands
	if output.NVMeSmartHealthInfoLog.HostReads > 0 {
		attrName := "host_read_commands"
		if resolvedName, found := aliasMap[attrName]; found {
			attrName = resolvedName
		}
		if attr, found := smartAttrs[attrName]; found {
			attr.Value = output.NVMeSmartHealthInfoLog.HostReads
			smartAttrs[attrName] = attr
		}
	}

	// Update host write commands
	if output.NVMeSmartHealthInfoLog.HostWrites > 0 {
		attrName := "host_write_commands"
		if resolvedName, found := aliasMap[attrName]; found {
			attrName = resolvedName
		}
		if attr, found := smartAttrs[attrName]; found {
			attr.Value = output.NVMeSmartHealthInfoLog.HostWrites
			smartAttrs[attrName] = attr
		}
	}

	// Update controller busy time
	if output.NVMeSmartHealthInfoLog.ControllerBusyTime > 0 {
		attrName := "controller_busy_time"
		if resolvedName, found := aliasMap[attrName]; found {
			attrName = resolvedName
		}
		if attr, found := smartAttrs[attrName]; found {
			attr.Value = output.NVMeSmartHealthInfoLog.ControllerBusyTime
			smartAttrs[attrName] = attr
		}
	}

	// Update error information log entries
	if output.NVMeSmartHealthInfoLog.NumErrLogEntries > 0 {
		attrName := "error_information_log_entries"
		if resolvedName, found := aliasMap[attrName]; found {
			attrName = resolvedName
		}
		if attr, found := smartAttrs[attrName]; found {
			attr.Value = output.NVMeSmartHealthInfoLog.NumErrLogEntries
			smartAttrs[attrName] = attr
		}
	}

	// Update percentage used
	if output.NVMeSmartHealthInfoLog.PercentageUsed > 0 {
		attrName := "percentage_used"
		if resolvedName, found := aliasMap[attrName]; found {
			attrName = resolvedName
		}
		if attr, found := smartAttrs[attrName]; found {
			attr.Value = output.NVMeSmartHealthInfoLog.PercentageUsed
			smartAttrs[attrName] = attr
		}
	}

	// Update available spare
	if output.NVMeSmartHealthInfoLog.AvailableSpare > 0 {
		attrName := "available_spare"
		if resolvedName, found := aliasMap[attrName]; found {
			attrName = resolvedName
		}
		if attr, found := smartAttrs[attrName]; found {
			attr.Value = output.NVMeSmartHealthInfoLog.AvailableSpare
			smartAttrs[attrName] = attr
		}
	}

	// Update available spare threshold
	if output.NVMeSmartHealthInfoLog.AvailableSpareThreshold > 0 {
		attrName := "available_spare_threshold"
		if resolvedName, found := aliasMap[attrName]; found {
			attrName = resolvedName
		}
		if attr, found := smartAttrs[attrName]; found {
			attr.Value = output.NVMeSmartHealthInfoLog.AvailableSpareThreshold
			smartAttrs[attrName] = attr
		}
	}

	// Update media and data integrity errors
	if output.NVMeSmartHealthInfoLog.MediaErrors > 0 {
		attrName := "media_and_data_integrity_errors"
		if resolvedName, found := aliasMap[attrName]; found {
			attrName = resolvedName
		}
		if attr, found := smartAttrs[attrName]; found {
			attr.Value = output.NVMeSmartHealthInfoLog.MediaErrors
			smartAttrs[attrName] = attr
		}
	}
}
