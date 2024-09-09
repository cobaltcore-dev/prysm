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

// SmartCtlScanOutput represents the root structure of the JSON output from smartctl --scan-open -j
type SmartCtlScanOutput struct {
	JSONFormatVersion []int64          `json:"json_format_version"`
	Smartctl          SmartCtlDetails  `json:"smartctl"`
	Devices           []SmartCtlDevice `json:"devices"`
}

// SmartCtlOutput represents the JSON output from smartctl with all details
type SmartCtlOutput struct {
	ATAVersion                       *SmartCtlATAVersion             `json:"ata_version,omitempty"`
	ATASMARTAttributes               *SmartCtlATASMARTAttributes     `json:"ata_smart_attributes,omitempty"`
	ATASMARTErrorLog                 *SmartCtlATASMARTErrorLog       `json:"ata_smart_error_log,omitempty"`
	Device                           SmartCtlDevice                  `json:"device"`
	DeviceType                       *SmartCtlDeviceType             `json:"device_type,omitempty"`
	DeviceModel                      string                          `json:"device_model,omitempty"`
	FormFactor                       *SmartCtlFormFactor             `json:"form_factor,omitempty"`
	JSONFormatVersion                []int64                         `json:"json_format_version"`
	InSmartctlDatabase               bool                            `json:"in_smartctl_database"`
	InterfaceSpeed                   *SmartCtlInterfaceSpeed         `json:"interface_speed,omitempty"`
	LocalTime                        SmartCtlLocalTime               `json:"local_time"`
	LogicalBlockSize                 int64                           `json:"logical_block_size,omitempty"`
	LogicalUnitID                    string                          `json:"logical_unit_id,omitempty"`
	ModelNumber                      string                          `json:"model_number,omitempty"`
	PhysicalBlockSize                int64                           `json:"physical_block_size,omitempty"`
	PowerCycleCount                  int64                           `json:"power_cycle_count"`
	PowerOnTime                      SmartCtlPowerOnTime             `json:"power_on_time"`
	Product                          string                          `json:"product"`
	RotationRate                     int64                           `json:"rotation_rate,omitempty"`
	SATAVersion                      *SmartCtlSATAVersion            `json:"sata_version,omitempty"`
	SCSIErrorCounterLog              *SmartCtlSCSIErrorCounterLog    `json:"scsi_error_counter_log,omitempty"`
	SCSIGrownDefectList              int64                           `json:"scsi_grown_defect_list,omitempty"`
	SCSIModelName                    string                          `json:"scsi_model_name,omitempty"`
	SCSIProduct                      string                          `json:"scsi_product,omitempty"`
	SCSIProtectionIntervalBytesPerLB int64                           `json:"scsi_protection_interval_bytes_per_lb,omitempty"`
	SCSIProtectionType               int64                           `json:"scsi_protection_type,omitempty"`
	SCSIRevision                     string                          `json:"scsi_revision,omitempty"`
	SCSIStartStopCycleCounter        *SmartCtlSCSIStartStopCycle     `json:"scsi_start_stop_cycle_counter,omitempty"`
	SCSITransportProtocol            *SmartCtlSCSITransportProtocol  `json:"scsi_transport_protocol,omitempty"`
	SCSIVendor                       string                          `json:"scsi_vendor,omitempty"`
	SCSIVersion                      string                          `json:"scsi_version,omitempty"`
	NVMeControllerID                 int64                           `json:"nvme_controller_id,omitempty"`
	NVMeIEEEOuiIdentifier            int64                           `json:"nvme_ieee_oui_identifier,omitempty"`
	NVMeNamespaces                   []SmartCtlNVMENamespace         `json:"nvme_namespaces,omitempty"`
	NVMeNumberOfNamespaces           int64                           `json:"nvme_number_of_namespaces,omitempty"`
	NVMePCIVendor                    *SmartCtlNVMePCIVendor          `json:"nvme_pci_vendor,omitempty"`
	NVMeSmartHealthInfoLog           *SmartCtlNVMeSmartHealthInfoLog `json:"nvme_smart_health_information_log,omitempty"`
	NVMeTotalCapacity                int64                           `json:"nvme_total_capacity,omitempty"`
	NVMeUnallocatedCapacity          int64                           `json:"nvme_unallocated_capacity,omitempty"`
	NVMeVersion                      *SmartCtlNVMeVersion            `json:"nvme_version,omitempty"`
	ModelFamily                      string                          `json:"model_family,omitempty"`
	ModelName                        string                          `json:"model_name"`
	FirmwareVersion                  string                          `json:"firmware_version"`
	SerialNumber                     string                          `json:"serial_number"`
	SmartStatus                      SmartCtlSmartStatus             `json:"smart_status"`
	SmartSupport                     SmartCtlSmartSupport            `json:"smart_support"`
	Smartctl                         SmartCtlDetails                 `json:"smartctl"`
	Temperature                      SmartCtlTemperature             `json:"temperature"`
	TemperatureWarning               *SmartCtlTemperatureWarning     `json:"temperature_warning,omitempty"`
	Trim                             *SmartCtlTrimSupport            `json:"trim,omitempty"`
	UserCapacity                     *SmartCtlUserCapacity           `json:"user_capacity,omitempty"`
	Vendor                           string                          `json:"vendor"`
	WWN                              *SmartCtlWWN                    `json:"wwn,omitempty"`
}

// SmartCtlATAVersion represents the ATA version information
type SmartCtlATAVersion struct {
	MajorValue int64  `json:"major_value"`
	MinorValue int64  `json:"minor_value"`
	String     string `json:"string"`
}

// SmartCtlATASMARTAttributes represents the ATA SMART attributes
type SmartCtlATASMARTAttributes struct {
	Revision int64                   `json:"revision"`
	Table    []SmartCtlATASMARTEntry `json:"table"`
}

// SmartCtlATASMARTEntry represents a single ATA SMART attribute entry
type SmartCtlATASMARTEntry struct {
	ID         int64                 `json:"id"`
	Name       string                `json:"name"`
	Value      int64                 `json:"value"`
	Worst      int64                 `json:"worst"`
	Thresh     int64                 `json:"thresh"`
	WhenFailed string                `json:"when_failed,omitempty"`
	Flags      SmartCtlATASMARTFlags `json:"flags"`
	Raw        SmartCtlATASMARTRaw   `json:"raw"`
}

// SmartCtlATASMARTFlags represents the flags for a single ATA SMART attribute entry
type SmartCtlATASMARTFlags struct {
	Value         int64  `json:"value"`
	String        string `json:"string"`
	Prefailure    bool   `json:"prefailure"`
	UpdatedOnline bool   `json:"updated_online"`
	Performance   bool   `json:"performance"`
	ErrorRate     bool   `json:"error_rate"`
	EventCount    bool   `json:"event_count"`
	AutoKeep      bool   `json:"auto_keep"`
}

// SmartCtlATASMARTRaw represents the raw value for a single ATA SMART attribute entry
type SmartCtlATASMARTRaw struct {
	Value  int64  `json:"value"`
	String string `json:"string"`
}

// SmartCtlATASMARTErrorLog represents the ATA SMART error log
type SmartCtlATASMARTErrorLog struct {
	Summary SmartCtlATASMARTErrorLogSummary `json:"summary"`
}

// SmartCtlATASMARTErrorLogSummary represents the summary of the ATA SMART error log
type SmartCtlATASMARTErrorLogSummary struct {
	Count       int64                           `json:"count"`
	Revision    int64                           `json:"revision"`
	LoggedCount int64                           `json:"logged_count,omitempty"`
	Table       []SmartCtlATASMARTErrorLogEntry `json:"table,omitempty"`
}

// SmartCtlATASMARTErrorLogEntry represents a single entry in the ATA SMART error log
type SmartCtlATASMARTErrorLogEntry struct {
	ErrorNumber         int64                               `json:"error_number"`
	LifetimeHours       int64                               `json:"lifetime_hours"`
	ErrorDescription    string                              `json:"error_description"`
	CompletionRegisters SmartCtlATASMARTCompletionRegisters `json:"completion_registers"`
	PreviousCommands    []SmartCtlATASMARTPreviousCommand   `json:"previous_commands"`
}

// SmartCtlATASMARTCompletionRegisters represents the completion registers of an error log entry
type SmartCtlATASMARTCompletionRegisters struct {
	Count  int64 `json:"count"`
	Device int64 `json:"device"`
	Error  int64 `json:"error"`
	LBA    int64 `json:"lba"`
	Status int64 `json:"status"`
}

// SmartCtlATASMARTPreviousCommand represents a previous command of an error log entry
type SmartCtlATASMARTPreviousCommand struct {
	CommandName         string                    `json:"command_name"`
	PowerupMilliseconds int64                     `json:"powerup_milliseconds"`
	Registers           SmartCtlATASMARTRegisters `json:"registers"`
}

// SmartCtlATASMARTRegisters represents the registers of a previous command
type SmartCtlATASMARTRegisters struct {
	Command       int64 `json:"command"`
	Count         int64 `json:"count"`
	Device        int64 `json:"device"`
	DeviceControl int64 `json:"device_control"`
	Features      int64 `json:"features"`
	LBA           int64 `json:"lba"`
}

// SmartCtlDevice represents the device details
type SmartCtlDevice struct {
	InfoName string `json:"info_name"`
	Name     string `json:"name"`
	Protocol string `json:"protocol"`
	Type     string `json:"type"`
}

// SmartCtlTrimSupport represents the TRIM support information
type SmartCtlTrimSupport struct {
	Supported bool `json:"supported"`
}

// SmartCtlDeviceType represents the type of the device
type SmartCtlDeviceType struct {
	Name            string `json:"name"`
	SCSITerminology string `json:"scsi_terminology,omitempty"`
	SCSIValue       int64  `json:"scsi_value,omitempty"`
}

// SmartCtlFormFactor represents the physical form factor of the device
type SmartCtlFormFactor struct {
	Name      string `json:"name"`
	SCSIValue int64  `json:"scsi_value,omitempty"`
	ATAValue  int64  `json:"ata_value,omitempty"`
}

// SmartCtlInterfaceSpeed represents the interface speed information
type SmartCtlInterfaceSpeed struct {
	Max     SmartCtlSpeedInfo `json:"max"`
	Current SmartCtlSpeedInfo `json:"current,omitempty"`
}

// SmartCtlSpeedInfo represents detailed speed information
type SmartCtlSpeedInfo struct {
	BitsPerUnit    int64  `json:"bits_per_unit"`
	SATAValue      int64  `json:"sata_value"`
	String         string `json:"string"`
	UnitsPerSecond int64  `json:"units_per_second"`
}

// SmartCtlLocalTime represents the local time when the data was collected
type SmartCtlLocalTime struct {
	Asctime string `json:"asctime"`
	TimeT   int64  `json:"time_t"`
}

// SmartCtlPowerOnTime represents the power-on time of the device
type SmartCtlPowerOnTime struct {
	Hours   int64 `json:"hours"`
	Minutes int64 `json:"minutes,omitempty"`
}

// SmartCtlSATAVersion represents the SATA version information
type SmartCtlSATAVersion struct {
	String string `json:"string"`
	Value  int64  `json:"value"`
}

// SmartCtlSCSIErrorCounterLog represents the SCSI error counter log
type SmartCtlSCSIErrorCounterLog struct {
	Read   SmartCtlSCSIErrorDetails `json:"read"`
	Verify SmartCtlSCSIErrorDetails `json:"verify"`
	Write  SmartCtlSCSIErrorDetails `json:"write"`
}

// SmartCtlSCSIErrorDetails represents details of SCSI errors
type SmartCtlSCSIErrorDetails struct {
	CorrectionAlgorithmInvocations int64  `json:"correction_algorithm_invocations"`
	ErrorsCorrectedByECCDelayed    int64  `json:"errors_corrected_by_eccdelayed"`
	ErrorsCorrectedByECCFast       int64  `json:"errors_corrected_by_eccfast"`
	ErrorsCorrectedByReReads       int64  `json:"errors_corrected_by_rereads_rewrites"`
	GigabytesProcessed             string `json:"gigabytes_processed"`
	TotalErrorsCorrected           int64  `json:"total_errors_corrected"`
	TotalUncorrectedErrors         int64  `json:"total_uncorrected_errors"`
}

// SmartCtlSCSIStartStopCycle represents the start-stop cycle counter
type SmartCtlSCSIStartStopCycle struct {
	AccumulatedLoadUnloadCycles                int64  `json:"accumulated_load_unload_cycles"`
	AccumulatedStartStopCycles                 int64  `json:"accumulated_start_stop_cycles"`
	SpecifiedCycleCountOverDeviceLifetime      int64  `json:"specified_cycle_count_over_device_lifetime"`
	SpecifiedLoadUnloadCountOverDeviceLifetime int64  `json:"specified_load_unload_count_over_device_lifetime"`
	WeekOfManufacture                          string `json:"week_of_manufacture"`
	YearOfManufacture                          string `json:"year_of_manufacture"`
}

// SmartCtlSCSITransportProtocol represents the transport protocol of the SCSI device
type SmartCtlSCSITransportProtocol struct {
	Name  string `json:"name"`
	Value int64  `json:"value"`
}

// SmartCtlNVMePCIVendor represents the PCI vendor details of the NVMe device
type SmartCtlNVMePCIVendor struct {
	ID          int64 `json:"id"`
	SubsystemID int64 `json:"subsystem_id"`
}

// SmartCtlNVMeSmartHealthInfoLog represents the NVMe SMART health information log
type SmartCtlNVMeSmartHealthInfoLog struct {
	AvailableSpare          int64   `json:"available_spare"`
	AvailableSpareThreshold int64   `json:"available_spare_threshold"`
	ControllerBusyTime      int64   `json:"controller_busy_time"`
	CriticalCompTime        int64   `json:"critical_comp_time"`
	CriticalWarning         int64   `json:"critical_warning"`
	DataUnitsRead           int64   `json:"data_units_read"`
	DataUnitsWritten        int64   `json:"data_units_written"`
	HostReads               int64   `json:"host_reads"`
	HostWrites              int64   `json:"host_writes"`
	MediaErrors             int64   `json:"media_errors"`
	NumErrLogEntries        int64   `json:"num_err_log_entries"`
	PercentageUsed          int64   `json:"percentage_used"`
	PowerCycles             int64   `json:"power_cycles"`
	PowerOnHours            int64   `json:"power_on_hours"`
	Temperature             int64   `json:"temperature"`
	TemperatureSensors      []int64 `json:"temperature_sensors,omitempty"`
	UnsafeShutdowns         int64   `json:"unsafe_shutdowns"`
	WarningTempTime         int64   `json:"warning_temp_time"`
}

// SmartCtlNVMENamespace represents details of an NVMe namespace
type SmartCtlNVMENamespace struct {
	ID               int64                `json:"id"`
	Size             SmartCtlNVMeCapacity `json:"size"`
	Utilization      SmartCtlNVMeCapacity `json:"utilization"`
	Capacity         SmartCtlNVMeCapacity `json:"capacity"`
	FormattedLBASize int64                `json:"formatted_lba_size"`
	EUI64            *SmartCtlNVMeEUI64   `json:"eui64,omitempty"`
}

// SmartCtlNVMeCapacity represents the capacity details of an NVMe namespace
type SmartCtlNVMeCapacity struct {
	Blocks int64 `json:"blocks"`
	Bytes  int64 `json:"bytes"`
}

// SmartCtlNVMeEUI64 represents the EUI-64 identifier of an NVMe namespace
type SmartCtlNVMeEUI64 struct {
	ExtID int64 `json:"ext_id"`
	OUI   int64 `json:"oui"`
}

// SmartCtlNVMeVersion represents the NVMe version
type SmartCtlNVMeVersion struct {
	String string `json:"string"`
	Value  int64  `json:"value"`
}

// SmartCtlSmartStatus represents the SMART health status
type SmartCtlSmartStatus struct {
	NVMe   *SmartCtlNVMeStatus `json:"nvme,omitempty"`
	Passed bool                `json:"passed"`
}

// SmartCtlNVMeStatus represents the NVMe specific SMART status
type SmartCtlNVMeStatus struct {
	Value int64 `json:"value"`
}

// SmartCtlSmartSupport indicates whether SMART is supported and enabled
type SmartCtlSmartSupport struct {
	Available bool `json:"available"`
	Enabled   bool `json:"enabled"`
}

// SmartCtlDetails represents the details about the smartctl command used
type SmartCtlDetails struct {
	Argv                 []string   `json:"argv"`
	BuildInfo            string     `json:"build_info"`
	DriveDatabaseVersion StringInfo `json:"drive_database_version,omitempty"`
	ExitStatus           int64      `json:"exit_status"`
	PlatformInfo         string     `json:"platform_info"`
	SvnRevision          string     `json:"svn_revision"`
	Version              []int64    `json:"version"`
}

// StringInfo represents a string-based information in the JSON output
type StringInfo struct {
	String string `json:"string"`
}

// SmartCtlTemperature represents the temperature readings of the device
type SmartCtlTemperature struct {
	Current   int64 `json:"current"`
	DriveTrip int64 `json:"drive_trip,omitempty"`
}

// SmartCtlTemperatureWarning represents whether temperature warning is enabled
type SmartCtlTemperatureWarning struct {
	Enabled bool `json:"enabled"`
}

// SmartCtlUserCapacity represents the user capacity of the device
type SmartCtlUserCapacity struct {
	Blocks int64 `json:"blocks"`
	Bytes  int64 `json:"bytes"`
}

// SmartCtlWWN represents the World Wide Name information
type SmartCtlWWN struct {
	ID  int64 `json:"id"`
	NAA int64 `json:"naa"`
	OUI int64 `json:"oui"`
}
