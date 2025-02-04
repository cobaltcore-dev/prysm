// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and prysm contributors
//
// SPDX-License-Identifier: Apache-2.0

package diskhealthmetrics

// Attribute represents a SMART attribute for Prometheus metrics
type NormalizedSmartAttribute struct {
	PromName string  // Normalized name for Prometheus
	Value    float64 // Attribute value
}

// DiskHealthMetrics represents the health metrics of a disk
type DiskHealthMetrics struct {
	DiskName   string                     `json:"disk_name"`   // Device name, e.g., "/dev/sda"
	NodeName   string                     `json:"node_name"`   // Name of the node where the disk is located
	InstanceID string                     `json:"instance_id"` // ID of the instance (useful in cloud environments)
	Attributes []NormalizedSmartAttribute `json:"attributes"`  // SMART attributes of the disk
}

// NormalizedSmartData represents normalized SMART data for consistency across devices
type NormalizedSmartData struct {
	NodeName           string                    `json:"node_name"`           // Name of the node where the drive is located
	InstanceID         string                    `json:"instance_id"`         // ID of the instance (useful in cloud environments)
	Device             string                    `json:"device"`              // Device name, e.g., "/dev/sda"
	DeviceInfo         *DeviceInfo               `json:"device_info"`         // Device information (e.g., vendor and model)
	CapacityGB         float64                   `json:"capacity_gb"`         // Capacity of the drive in gigabytes
	HealthStatus       *bool                     `json:"health_status"`       // Overall health status of the drive (true if healthy, false if failing, nil if unknown)
	TemperatureCelsius *int64                    `json:"temperature_celsius"` // Current temperature of the drive in Celsius
	ReallocatedSectors *int64                    `json:"reallocated_sectors"` // Number of reallocated sectors on the drive
	PendingSectors     *int64                    `json:"pending_sectors"`     // Number of pending sectors (unreadable sectors waiting to be reallocated)
	PowerOnHours       *int64                    `json:"power_on_hours"`      // Total number of hours the drive has been powered on
	SSDLifeUsed        *int64                    `json:"ssd_life_used"`       // Percentage of SSD life used (useful for SSD wear monitoring)
	ErrorCounts        map[string]int64          `json:"error_counts"`        // Dictionary of various error counts (e.g., command timeouts, CRC errors)
	Attributes         map[string]SmartAttribute `json:"attributes"`          // key-value pairs of SMART attributes with their values
}

// NatsEvent represents an event to be published to NATS
type NatsEvent struct {
	NodeName   string            `json:"node_name"`   // Name of the node where the drive is located
	InstanceID string            `json:"instance_id"` // ID of the instance (useful in cloud environments)
	Device     string            `json:"device"`      // Device identifier (e.g., /dev/sda)
	EventType  string            `json:"event_type"`  // e.g., 'health_alert', 'usage_alert'
	Severity   string            `json:"severity"`    // e.g., 'info', 'warning', 'critical'
	Message    string            `json:"message"`     // Description of the event
	Details    map[string]string `json:"details"`     // Additional details, such as SMART attributes
}

type DeviceInfo struct {
	ModelFamily     string  // ATA devices might have this, but it can be left blank for SCSI/NVMe.
	DeviceModel     string  // Device-specific model name (e.g., "Samsung SSD 970 EVO" for NVMe).
	SerialNumber    string  // Unique identifier for the device.
	FirmwareVersion string  // Firmware version if available (common for all protocols).
	Vendor          string  // The vendor/manufacturer of the device (e.g., "Seagate", "LENOVO").
	Product         string  // The product name or number (e.g., "WUS721010AL5204").
	LunID           string  // Logical Unit Identifier, mostly used for SCSI devices.
	Capacity        float64 // Capacity of the device in GB.
	DWPD            float64 // Drive Writes Per Day (usually relevant for SSDs, NVMe).
	RPM             int64   // Rotational speed in RPM (for HDDs, relevant for ATA/SCSI).
	FormFactor      string  // Physical form factor, like "sff" or "lff".
	Media           string  // Media type, such as "ssd", "hdd", "nvme".
	HealthStatus    bool
}

// SmartAttribute defines the structure for a SMART attribute's metadata.
type SmartAttribute struct {
	Description string
	Unit        string
	Threshold   int64
	Value       int64
	Worst       int64
	RawValue    int64
}

// GetSmartAttributes returns a map of SMART attributes with their metadata.
// Note: While SMART attribute IDs are commonly used, they are not universally
// standardized across different manufacturers and models. This function
// avoids relying solely on numeric IDs due to potential inconsistencies.
// Instead, it provides a structured mapping based on attribute names,
// allowing for more accurate and reliable interpretation of SMART data
// across various drives. For vendor-specific attributes, consider
// supplementing this with manufacturer-specific mappings.
func GetSmartAttributes() map[string]SmartAttribute {
	smartAttrs := map[string]SmartAttribute{
		"airflow_temperature_cel":         {"Airflow Temperature in Celsius", "Celsius", -1, -1, -1, -1},
		"command_timeout":                 {"Command Timeout", "ms", -1, -1, -1, -1},
		"current_pending_sector":          {"Current Pending Sector", "count", -1, -1, -1, -1},
		"end_to_end_error":                {"End-to-End Error", "count", -1, -1, -1, -1},
		"erase_fail_count":                {"Erase Fail Count", "count", -1, -1, -1, -1},
		"g_sense_error_rate":              {"G-sense Error Rate", "count", -1, -1, -1, -1},
		"hardware_ecc_recovered":          {"Hardware ECC Recovered", "count", -1, -1, -1, -1},
		"host_reads_mib":                  {"Host Reads in MiB", "MiB", -1, -1, -1, -1},
		"host_reads_32mib":                {"Host Reads in 32 MiB", "32 MiB", -1, -1, -1, -1},
		"host_writes_mib":                 {"Host Writes in MiB", "MiB", -1, -1, -1, -1},
		"host_writes_32mib":               {"Host Writes in 32 MiB", "32 MiB", -1, -1, -1, -1},
		"load_cycle_count":                {"Load Cycle Count", "count", -1, -1, -1, -1},
		"helium_level":                    {"Helium Level", "percent", -1, -1, -1, -1},
		"media_wearout_indicator":         {"Media Wearout Indicator", "percent", -1, -1, -1, -1},
		"multi_zone_error_rate":           {"Multi-Zone Error Rate", "count", -1, -1, -1, -1},
		"wear_leveling_count":             {"Wear Leveling Count", "count", -1, -1, -1, -1},
		"nand_writes_1gib":                {"NAND Writes in 1 GiB", "GiB", -1, -1, -1, -1},
		"offline_uncorrectable":           {"Offline Uncorrectable", "count", -1, -1, -1, -1},
		"percent_life_remaining":          {"Percent Life Remaining", "percent", -1, -1, -1, -1},
		"percent_lifetime_remain":         {"Percent Lifetime Remaining", "percent", -1, -1, -1, -1},
		"percentage_used":                 {"Percentage Used", "percent", -1, -1, -1, -1},
		"power_cycle_count":               {"Power Cycle Count", "count", -1, -1, -1, -1},
		"power_off_retract_count":         {"Power Off Retract Count", "count", -1, -1, -1, -1},
		"power_on_hours":                  {"Power-On Hours", "hours", -1, -1, -1, -1},
		"program_fail_count":              {"Program Fail Count", "count", -1, -1, -1, -1},
		"raw_read_error_rate":             {"Raw Read Error Rate", "count", -1, -1, -1, -1},
		"reallocated_event_count":         {"Reallocated Event Count", "count", -1, -1, -1, -1},
		"reallocated_sector_ct":           {"Reallocated Sector Count", "count", -1, -1, -1, -1},
		"reallocate_nand_blk_cnt":         {"Reallocate NAND Block Count", "count", -1, -1, -1, -1},
		"reported_uncorrect":              {"Reported Uncorrectable Errors", "count", -1, -1, -1, -1},
		"sata_downshift_count":            {"SATA Downshift Count", "count", -1, -1, -1, -1},
		"seek_error_rate":                 {"Seek Error Rate", "count", -1, -1, -1, -1},
		"spin_retry_count":                {"Spin Retry Count", "count", -1, -1, -1, -1},
		"spin_up_time":                    {"Spin-Up Time", "ms", -1, -1, -1, -1},
		"start_stop_count":                {"Start/Stop Count", "count", -1, -1, -1, -1},
		"temperature_case":                {"Case Temperature", "Celsius", -1, -1, -1, -1},
		"temperature_celsius":             {"Temperature in Celsius", "Celsius", -1, -1, -1, -1},
		"temperature_internal":            {"Internal Temperature", "Celsius", -1, -1, -1, -1},
		"total_lbas_read":                 {"Total LBAs Read", "sectors", -1, -1, -1, -1},
		"total_lbas_written":              {"Total LBAs Written", "sectors", -1, -1, -1, -1},
		"total_host_sector_write":         {"Total Host Sector Writes", "sectors", -1, -1, -1, -1},
		"udma_crc_error_count":            {"UDMA CRC Error Count", "count", -1, -1, -1, -1},
		"unsafe_shutdown_count":           {"Unsafe Shutdown Count", "count", -1, -1, -1, -1},
		"workld_host_reads_perc":          {"Workload Host Reads Percentage", "percent", -1, -1, -1, -1},
		"workld_media_wear_indic":         {"Workload Media Wear Indicator", "percent", -1, -1, -1, -1},
		"workload_minutes":                {"Workload Minutes", "minutes", -1, -1, -1, -1},
		"read_errors_corrected":           {"Read Errors Corrected", "count", -1, -1, -1, -1},
		"write_errors_corrected":          {"Write Errors Corrected", "count", -1, -1, -1, -1},
		"verify_errors_corrected":         {"Verify Errors Corrected", "count", -1, -1, -1, -1},
		"read_gigabytes_processed":        {"Read Gigabytes Processed", "GiB", -1, -1, -1, -1},
		"write_gigabytes_processed":       {"Write Gigabytes Processed", "GiB", -1, -1, -1, -1},
		"total_uncorrected_read_errors":   {"Total Uncorrected Read Errors", "count", -1, -1, -1, -1},
		"total_uncorrected_write_errors":  {"Total Uncorrected Write Errors", "count", -1, -1, -1, -1},
		"total_uncorrected_verify_errors": {"Total Uncorrected Verify Errors", "count", -1, -1, -1, -1},
		"grown_defects_count":             {"Grown Defects Count", "count", -1, -1, -1, -1},
	}

	return smartAttrs
}

// CleanupSmartAttributes removes entries where all fields (Threshold, Value, Worst, RawValue) are -1.
func CleanupSmartAttributes(smartAttrs map[string]SmartAttribute) {
	for key, attr := range smartAttrs {
		if attr.Threshold == -1 && attr.Value == -1 && attr.Worst == -1 && attr.RawValue == -1 {
			delete(smartAttrs, key)
		}
	}
}

// Alias map to handle different names for the same attribute
var aliasMap = map[string]string{
	"current_drive_temperature": "temperature_celsius",
}
