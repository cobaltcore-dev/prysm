// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and prysm contributors
//
// SPDX-License-Identifier: Apache-2.0

package diskhealthmetrics

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"github.com/rs/zerolog/log"
)

func checkNVMeCliInstalled() bool {
	_, err := exec.LookPath("nvme")
	return err == nil
}

// collectNVMeControllerData collects NVMe controller data using nvme id-ctrl
func collectNVMeControllerData(devicePath string) (*NVMeIDControllerOutput, error) {
	out, err := exec.Command("nvme", "id-ctrl", devicePath, "-o", "json").Output()
	if err != nil {
		return nil, fmt.Errorf("error running nvme id-ctrl: %v", err)
	}

	var controllerData NVMeIDControllerOutput
	if err := json.Unmarshal(out, &controllerData); err != nil {
		return nil, fmt.Errorf("error parsing nvme id-ctrl JSON: %v", err)
	}

	return &controllerData, nil
}

// collectNVMeErrorLog collects NVMe error log using nvme error-log
func collectNVMeErrorLog(devicePath string) (*NVMeErrorLogOutput, error) {
	out, err := exec.Command("nvme", "error-log", devicePath, "-o", "json").Output()
	if err != nil {
		return nil, fmt.Errorf("error running nvme error-log: %v", err)
	}

	var errorLog NVMeErrorLogOutput
	if err := json.Unmarshal(out, &errorLog); err != nil {
		return nil, fmt.Errorf("error parsing nvme error-log JSON: %v", err)
	}

	return &errorLog, nil
}

// enhanceNVMeData enhances existing SmartCtlOutput with additional NVMe-CLI data
// enhanceNVMeData enhances existing SmartCtlOutput with additional NVMe-CLI data
func enhanceNVMeData(smartData *SmartCtlOutput, nvmeController *NVMeIDControllerOutput, nvmeErrors *NVMeErrorLogOutput) {
	if nvmeController != nil {
		// Override/enhance device information with nvme-cli data
		if nvmeController.ModelNumber != "" {
			smartData.ModelName = strings.TrimSpace(nvmeController.ModelNumber)
		}

		if nvmeController.SerialNumber != "" {
			smartData.SerialNumber = strings.TrimSpace(nvmeController.SerialNumber)
		}

		if nvmeController.FirmwareRevision != "" {
			smartData.FirmwareVersion = strings.TrimSpace(nvmeController.FirmwareRevision)
		}

		// Update NVMe capacity if available and different
		if nvmeController.TotalCapacity > 0 {
			smartData.NVMeTotalCapacity = nvmeController.TotalCapacity
		}

		if nvmeController.UnallocatedCapacity > 0 {
			smartData.NVMeUnallocatedCapacity = nvmeController.UnallocatedCapacity
		}

		// Add vendor information from nvme-cli
		if nvmeController.VendorID > 0 {
			smartData.Vendor = fmt.Sprintf("VID:0x%04x", nvmeController.VendorID)
		}

		// Store SubsystemNQN - this will be used by your existing rebranding logic
		if nvmeController.SubsystemNQN != "" {
			smartData.Product = nvmeController.SubsystemNQN
			log.Debug().Str("subnqn", nvmeController.SubsystemNQN).Msg("SubsystemNQN added for rebranding detection")
		}

		// Store IEEE OUI information
		if nvmeController.IEEE != "" {
			smartData.LogicalUnitID = nvmeController.IEEE
		}
	}

	// Enhance error information
	if nvmeErrors != nil && len(nvmeErrors.Errors) > 0 {
		var totalErrors, mediaErrors, abortedCommands int64

		for _, err := range nvmeErrors.Errors {
			if err.ErrorCount > 0 {
				totalErrors += err.ErrorCount

				// Classify errors based on status field
				switch err.StatusField & 0x7FF {
				case 0x281: // Media and Data Integrity Error
					mediaErrors += err.ErrorCount
				case 0x7: // Aborted Command
					abortedCommands += err.ErrorCount
				}
			}
		}

		// Update NVMe health log with enhanced error information
		if smartData.NVMeSmartHealthInfoLog != nil {
			smartData.NVMeSmartHealthInfoLog.MediaErrors = mediaErrors
			smartData.NVMeSmartHealthInfoLog.NumErrLogEntries = int64(len(nvmeErrors.Errors))
		}

		log.Debug().Int64("total_errors", totalErrors).Int64("media_errors", mediaErrors).Msg("NVMe error analysis complete")
	}
}

// processNVMeSpecificAttributes processes NVMe-specific attributes from nvme-cli
func processNVMeSpecificAttributes(smartAttrs map[string]SmartAttribute, nvmeController *NVMeIDControllerOutput, nvmeErrors *NVMeErrorLogOutput) {
	// Process NVMe controller-specific attributes
	if nvmeController != nil {
		// Store SubsystemNQN as string (you might need to handle this differently since SmartAttribute expects int64)
		if nvmeController.SubsystemNQN != "" {
			log.Debug().Str("subnqn", nvmeController.SubsystemNQN).Msg("Processing NVMe subsystem NQN")
			// For now, we'll store the length as a metric since SmartAttribute uses int64
			updateAttributeFromValue(smartAttrs, "nvme_subsystem_nqn", int64(len(nvmeController.SubsystemNQN)), int64(len(nvmeController.SubsystemNQN)), -1, -1, "chars")
		}

		// Store IEEE OUI information
		if nvmeController.IEEE != "" {
			log.Debug().Str("ieee", nvmeController.IEEE).Msg("Processing NVMe IEEE OUI")
			// Convert hex string to int64 for storage
			if oui, err := strconv.ParseInt(strings.Replace(nvmeController.IEEE, "0x", "", -1), 16, 64); err == nil {
				updateAttributeFromValue(smartAttrs, "nvme_ieee_oui", oui, oui, -1, -1, "hex")
			}
		}

		// Store vendor and subsystem vendor IDs
		if nvmeController.VendorID > 0 {
			updateAttributeFromValue(smartAttrs, "nvme_vendor_id", nvmeController.VendorID, nvmeController.VendorID, -1, -1, "id")
		}

		if nvmeController.SubsystemVendorID > 0 {
			updateAttributeFromValue(smartAttrs, "nvme_subsystem_vendor_id", nvmeController.SubsystemVendorID, nvmeController.SubsystemVendorID, -1, -1, "id")
		}
	}

	// Process error log specific metrics
	if nvmeErrors != nil {
		errorCount := int64(len(nvmeErrors.Errors))
		updateAttributeFromValue(smartAttrs, "nvme_error_log_entries", errorCount, errorCount, -1, -1, "count")

		// Analyze error types
		var fabricWarnings, sparseErrors, changeNotifications int64
		var mediaErrors, abortedCommands, timeoutErrors int64

		for _, err := range nvmeErrors.Errors {
			if err.ErrorCount > 0 {
				// Classify based on status field and other indicators
				statusCode := err.StatusField & 0x7FF

				switch statusCode {
				case 0x281: // Media and Data Integrity Error
					mediaErrors += err.ErrorCount
				case 0x7: // Aborted Command
					abortedCommands += err.ErrorCount
				case 0x4: // Command Timeout
					timeoutErrors += err.ErrorCount
				}

				// Check for fabric-related errors (transport type specific)
				if err.TransportType > 0 && err.TransportTypeSpecificInfo > 0 {
					fabricWarnings += err.ErrorCount
				}

				// Sparse errors detection (based on LBA patterns or specific error codes)
				if err.LBA > 0 && (statusCode == 0x281 || statusCode == 0x282) {
					sparseErrors += err.ErrorCount
				}

				// Change notifications (vendor specific field analysis)
				if err.VendorSpecific > 0 {
					changeNotifications += err.ErrorCount
				}
			}
		}

		// Update specific error type attributes
		if fabricWarnings > 0 {
			updateAttributeFromValue(smartAttrs, "nvme_fabric_warnings", fabricWarnings, fabricWarnings, -1, -1, "count")
		}

		if sparseErrors > 0 {
			updateAttributeFromValue(smartAttrs, "nvme_sparse_errors", sparseErrors, sparseErrors, -1, -1, "count")
		}

		if changeNotifications > 0 {
			updateAttributeFromValue(smartAttrs, "nvme_change_notifications", changeNotifications, changeNotifications, -1, -1, "count")
		}

		// Additional detailed error metrics
		if mediaErrors > 0 {
			updateAttributeFromValue(smartAttrs, "nvme_media_errors", mediaErrors, mediaErrors, -1, -1, "count")
		}

		if abortedCommands > 0 {
			updateAttributeFromValue(smartAttrs, "nvme_aborted_commands", abortedCommands, abortedCommands, -1, -1, "count")
		}

		if timeoutErrors > 0 {
			updateAttributeFromValue(smartAttrs, "nvme_timeout_errors", timeoutErrors, timeoutErrors, -1, -1, "count")
		}

		log.Debug().Int64("fabric_warnings", fabricWarnings).Int64("sparse_errors", sparseErrors).Int64("change_notifications", changeNotifications).Msg("NVMe error classification complete")
	}
}
