// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and prysm contributors
//
// SPDX-License-Identifier: Apache-2.0

package diskhealthmetrics

type SMARTAttribute struct {
	ID          int    `json:"id"`
	Key         string `json:"key"`
	Name        string `json:"name"`
	Value       uint64 `json:"value"`
	Critical    bool   `json:"critical"`
	Description string `json:"-"`
	PromName    string `json:"-"`
	PromHelp    string `json:"-"`
}

// https://en.wikipedia.org/wiki/Self-Monitoring,_Analysis_and_Reporting_Technology
// https://www.hdsentinel.com/smart/smartattr.php
// https://www.hdsentinel.com/forum/viewtopic.php?t=10077
// This is the list of all known attributes supported by IDE and Serial ATA hard disks. Note: some manufacturers may use the attributes for different purposes also. Attributes not listed here are "vendor specific" attributes (their purpose is not known).
var SMARTAttributes = []SMARTAttribute{
	{ID: 1, Key: "Raw_Read_Error_Rate", Name: "Raw Read Error Rate", Critical: true, Description: "Errors occurred while reading raw data from a disk", PromName: "disk_raw_read_error_rate", PromHelp: "Rate of hardware read errors"},
	{ID: 2, Key: "Throughput_Performance", Name: "Throughput Performance", Critical: false, Description: "General throughput performance of the hard disk", PromName: "disk_throughput_performance", PromHelp: "General throughput performance of the hard disk"},
	{ID: 3, Key: "Spin_Up_Time", Name: "Spin Up Time", Critical: true, Description: "Time needed by spindle to spin-up to full RPM", PromName: "disk_spin_up_time", PromHelp: "Average time of spindle spin up"},
	{ID: 4, Key: "Start_Stop_Count", Name: "Start/Stop Count", Critical: false, Description: "Count of start/stop cycles of spindle", PromName: "disk_start_stop_count", PromHelp: "Count of start/stop cycles of spindle"},
	{ID: 5, Key: "Reallocated_Sector_Ct", Name: "Reallocated Sector Count", Critical: true, Description: "Count of sectors moved to the spare area", PromName: "disk_reallocated_sector_ct", PromHelp: "Number of reallocated sectors on the disk"},
	{ID: 6, Key: "Read_Channel_Margin", Name: "Read Channel Margin", Critical: false, Description: "Margin of a channel while reading data", PromName: "disk_read_channel_margin", PromHelp: "Margin of a channel while reading data"},
	{ID: 7, Key: "Seek_Error_Rate", Name: "Seek Error Rate", Critical: true, Description: "Rate of positioning errors of the read/write heads", PromName: "disk_seek_error_rate", PromHelp: "Rate of seek errors of the magnetic heads"},
	{ID: 8, Key: "Seek_Time_Performance", Name: "Seek Time Performance", Critical: true, Description: "Average time of seek operations of the heads", PromName: "disk_seek_time_performance", PromHelp: "Performance of seek operations"},
	{ID: 9, Key: "Power_On_Hours", Name: "Power-On Hours", Critical: false, Description: "Total time the drive is powered on", PromName: "disk_power_on_hours", PromHelp: "Total time the drive is powered on"},
	{ID: 10, Key: "Spin_Retry_Count", Name: "Spin Retry Count", Critical: true, Description: "Retry count of spin start attempts", PromName: "disk_spin_retry_count", PromHelp: "Count of spin-up retries"},
	{ID: 11, Key: "Calibration_Retry_Count", Name: "Calibration Retry Count", Critical: true, Description: "Number of attempts to calibrate a drive", PromName: "disk_calibration_retry_count", PromHelp: "Number of attempts to calibrate a drive"},
	{ID: 12, Key: "Power_Cycle_Count", Name: "Power Cycle Count", Critical: false, Description: "Number of complete power on/off cycles", PromName: "disk_power_cycle_count", PromHelp: "Number of complete power on/off cycles"},
	{ID: 13, Key: "Soft_Read_Error_Rate", Name: "Soft Read Error Rate", Critical: false, Description: "Number of software read errors", PromName: "disk_soft_read_error_rate", PromHelp: "Number of software read errors"},
	// {ID: 22, Key: "Helium_Level", Name: "Helium Level", Critical: false, Description: "Current helium level", PromName: "disk_helium_level", PromHelp: "Current helium level"},
	// {ID: 23, Key: "Integrity_Errors", Name: "Integrity Errors", Critical: false, Description: "Number of data integrity errors", PromName: "disk_integrity_errors", PromHelp: "Number of data integrity errors"},
	// {ID: 24, Key: "Initial_Bad_Block_Count", Name: "Initial Bad Block Count", Critical: false, Description: "Number of initial bad blocks", PromName: "disk_initial_bad_block_count", PromHelp: "Number of initial bad blocks"},
	// {ID: 170, Key: "Available_Reserved_Space", Name: "Available Reserved Space", Critical: false, Description: "Amount of reserved space available", PromName: "disk_available_reserved_space", PromHelp: "Amount of reserved space available"},
	// {ID: 171, Key: "Program_Fail_Count", Name: "Program Fail Count", Critical: false, Description: "Number of flash program operation failures", PromName: "disk_program_fail_count", PromHelp: "Number of flash program operation failures"},
	// {ID: 172, Key: "Erase_Fail_Count", Name: "Erase Fail Count", Critical: false, Description: "Number of flash erase operation failures", PromName: "disk_erase_fail_count", PromHelp: "Number of flash erase operation failures"},
	// {ID: 173, Key: "Wear_Leveling_Count", Name: "Wear Leveling Count", Critical: false, Description: "Wear leveling count", PromName: "disk_wear_leveling_count", PromHelp: "Wear leveling count"},
	// {ID: 174, Key: "Unexpected_Power_Loss_Count", Name: "Unexpected Power Loss Count", Critical: false, Description: "Number of unexpected power loss events", PromName: "disk_unexpected_power_loss_count", PromHelp: "Number of unexpected power loss events"},
	// {ID: 175, Key: "Power_Loss_Protection_Failure", Name: "Power Loss Protection Failure", Critical: false, Description: "Number of failures due to power loss protection", PromName: "disk_power_loss_protection_failure", PromHelp: "Number of failures due to power loss protection"},
	// {ID: 176, Key: "Erase_Fail_Count", Name: "Erase Fail Count", Critical: false, Description: "Number of flash erase operation failures", PromName: "disk_erase_fail_count", PromHelp: "Number of flash erase operation failures"},
	// {ID: 177, Key: "Wear_Range_Delta", Name: "Wear Range Delta", Critical: false, Description: "Delta between most and least worn flash blocks", PromName: "disk_wear_range_delta", PromHelp: "Delta between most and least worn flash blocks"},
	// {ID: 178, Key: "Used_Reserved_Block_Count_Total", Name: "Used Reserved Block Count Total", Critical: false, Description: "Number of reserved blocks used", PromName: "disk_used_reserved_block_count_total", PromHelp: "Number of reserved blocks used"},
	// {ID: 179, Key: "Used_Reserved_Block_Count", Name: "Used Reserved Block Count", Critical: false, Description: "Number of reserved blocks used", PromName: "disk_used_reserved_block_count", PromHelp: "Number of reserved blocks used"},
	// {ID: 180, Key: "Unused_Reserved_Block_Count_Total", Name: "Unused Reserved Block Count Total", Critical: false, Description: "Number of unused reserved blocks", PromName: "disk_unused_reserved_block_count_total", PromHelp: "Number of unused reserved blocks"},
	// {ID: 181, Key: "Program_Fail_Count", Name: "Program Fail Count", Critical: false, Description: "Number of flash program operation failures", PromName: "disk_program_fail_count", PromHelp: "Number of flash program operation failures"},
	// {ID: 182, Key: "Erase_Fail_Count", Name: "Erase Fail Count", Critical: false, Description: "Number of flash erase operation failures", PromName: "disk_erase_fail_count", PromHelp: "Number of flash erase operation failures"},
	// {ID: 183, Key: "Runtime_Bad_Block", Name: "Runtime Bad Block", Critical: false, Description: "Number of runtime bad blocks", PromName: "disk_runtime_bad_block", PromHelp: "Number of runtime bad blocks"},
	// {ID: 184, Key: "End-to-End_Error", Name: "End-to-End Error", Critical: true, Description: "Number of end-to-end errors", PromName: "disk_end_to_end_error", PromHelp: "Number of end-to-end errors"},
	// {ID: 185, Key: "Head_Stability", Name: "Head Stability", Critical: false, Description: "Stability of the head", PromName: "disk_head_stability", PromHelp: "Stability of the head"},
	// {ID: 186, Key: "Induced_Op-Vibration_Detection", Name: "Induced Op-Vibration Detection", Critical: false, Description: "Number of induced op-vibration detections", PromName: "disk_induced_op_vibration_detection", PromHelp: "Number of induced op-vibration detections"},
	// {ID: 187, Key: "Reported_Uncorrectable_Errors", Name: "Reported Uncorrectable Errors", Critical: true, Description: "Number of reported uncorrectable errors", PromName: "disk_reported_uncorrectable_errors", PromHelp: "Number of reported uncorrectable errors"},
	// {ID: 188, Key: "Command_Timeout", Name: "Command Timeout", Critical: true, Description: "Number of command timeouts", PromName: "disk_command_timeout", PromHelp: "Number of command timeouts"},
	// {ID: 189, Key: "High_Fly_Writes", Name: "High Fly Writes", Critical: false, Description: "Number of high fly writes", PromName: "disk_high_fly_writes", PromHelp: "Number of high fly writes"},
	{ID: 190, Key: "Airflow_Temperature_Cel", Name: "Airflow Temperature Celsius", Critical: false, Description: "Airflow temperature", PromName: "disk_airflow_temperature_celsius", PromHelp: "Airflow temperature in Celsius"},
	{ID: 191, Key: "G-Sense_Error_Rate", Name: "G-Sense Error Rate", Critical: false, Description: "Count of errors resulting from shock or vibration", PromName: "disk_g_sense_error_rate", PromHelp: "Number of mechanical errors resulting from shock or vibration"},
	{ID: 192, Key: "Power-Off_Retract_Count", Name: "Power-Off Retract Count", Critical: false, Description: "Count of power off cycles", PromName: "disk_power_off_retract_count", PromHelp: "Count of power off cycles"},
	{ID: 193, Key: "Load_Unload_Cycle_Count", Name: "Load/Unload Cycle Count", Critical: false, Description: "Number of load/unload cycles", PromName: "disk_load_unload_cycle_count", PromHelp: "Number of load/unload cycles"},
	{ID: 194, Key: "Temperature_Celsius", Name: "Temperature Celsius", Critical: false, Description: "Disk temperature in Celsius", PromName: "disk_temperature_celsius", PromHelp: "Disk temperature in Celsius"},
	{ID: 195, Key: "Hardware_ECC_Recovered", Name: "Hardware ECC Recovered", Critical: false, Description: "Count of corrected errors", PromName: "disk_hardware_ecc_recovered", PromHelp: "Count of corrected errors"},
	{ID: 196, Key: "Reallocation_Event_Count", Name: "Reallocation Event Count", Critical: true, Description: "Count of sector remap operations", PromName: "disk_reallocation_event_count", PromHelp: "Count of sector remap operations"},
	{ID: 197, Key: "Current_Pending_Sector", Name: "Current Pending Sector Count", Critical: true, Description: "Count of unstable sectors", PromName: "disk_current_pending_sector_count", PromHelp: "Count of unstable sectors"},
	{ID: 198, Key: "Offline_Uncorrectable", Name: "Offline Uncorrectable Sector Count", Critical: true, Description: "Count of uncorrectable errors when reading/writing", PromName: "disk_offline_uncorrectable_sector_count", PromHelp: "Count of uncorrectable errors when reading/writing"},
	{ID: 199, Key: "UDMA_CRC_Error_Count", Name: "UDMA CRC Error Count", Critical: false, Description: "Count of errors during data transfer between disk and host", PromName: "disk_udma_crc_error_count", PromHelp: "Count of errors during data transfer between disk and host"},
	{ID: 200, Key: "Write_Error_Rate", Name: "Write Error Rate", Critical: false, Description: "Errors occurred while writing raw data from a disk", PromName: "disk_write_error_rate", PromHelp: "Rate of errors occurred while writing raw data from a disk"},
	{ID: 201, Key: "Soft_Read_Error_Rate", Name: "Soft Read Error Rate", Critical: false, Description: "Number of software read errors", PromName: "disk_soft_read_error_rate", PromHelp: "Number of software read errors"},
	{ID: 202, Key: "Data_Address_Mark_Errors", Name: "Data Address Mark Errors", Critical: false, Description: "Number of data address mark errors", PromName: "disk_data_address_mark_errors", PromHelp: "Number of data address mark errors"},
	{ID: 203, Key: "Run_Out_Cancel", Name: "Run Out Cancel", Critical: false, Description: "Number of data correction errors", PromName: "disk_run_out_cancel", PromHelp: "Number of data correction errors"},
	{ID: 204, Key: "Soft_ECC_Correction", Name: "Soft ECC Correction", Critical: false, Description: "Number of corrected data errors", PromName: "disk_soft_ecc_correction", PromHelp: "Number of corrected data errors"},
	{ID: 205, Key: "Thermal_Asperity_Rate", Name: "Thermal Asperity Rate", Critical: false, Description: "Number of thermal problems", PromName: "disk_thermal_asperity_rate", PromHelp: "Number of thermal problems"},
	{ID: 206, Key: "Flying_Height", Name: "Flying Height", Critical: false, Description: "Head flying height", PromName: "disk_flying_height", PromHelp: "Head flying height"},
	{ID: 207, Key: "Spin_High_Current", Name: "Spin High Current", Critical: false, Description: "Current value during spin up", PromName: "disk_spin_high_current", PromHelp: "Current value during spin up"},
	{ID: 208, Key: "Spin_Buzz", Name: "Spin Buzz", Critical: false, Description: "Number of cycles needed to spin up", PromName: "disk_spin_buzz", PromHelp: "Number of cycles needed to spin up"},
	{ID: 209, Key: "Offline_Seek_Performance", Name: "Offline Seek Performance", Critical: false, Description: "Drive performance during offline operations", PromName: "disk_offline_seek_performance", PromHelp: "Drive performance during offline operations"},
	// {ID: 210, Key: "Perc_Rated_Life_Used", Name: "Percentage Rated Life Used", Critical: false, Description: "Percentage of the rated lifetime used", PromName: "disk_perc_rated_life_used", PromHelp: "Percentage of the rated lifetime used"},
	// {ID: 211, Key: "Unknown_Attribute", Name: "Unknown Attribute", Critical: false, Description: "Unknown attribute", PromName: "disk_unknown_attribute", PromHelp: "Unknown attribute"},
	// {ID: 212, Key: "Available_Reserved_Space", Name: "Available Reserved Space", Critical: false, Description: "Amount of reserved space available", PromName: "disk_available_reserved_space", PromHelp: "Amount of reserved space available"},
	{ID: 220, Key: "Disk_Shift", Name: "Disk Shift", Critical: false, Description: "Distance the disk has shifted relative to the spindle", PromName: "disk_shift", PromHelp: "Incorrect disk spin can be caused by mechanical shock or high temperature"},
	{ID: 221, Key: "G-Sense_Error_Rate", Name: "G-Sense Error Rate", Critical: false, Description: "Count of errors resulting from shock or vibration", PromName: "disk_g_sense_error_rate", PromHelp: "Number of mechanical errors resulting from shock or vibration"},
	{ID: 222, Key: "Loaded_Hours", Name: "Loaded Hours", Critical: false, Description: "Number of powered on hours", PromName: "disk_loaded_hours", PromHelp: "Number of powered on hours"},
	{ID: 223, Key: "Load_Unload_Retry_Count", Name: "Load/Unload Retry Count", Critical: false, Description: "Number of load/unload operations", PromName: "disk_load_unload_retry_count", PromHelp: "Number of load/unload operations"},
	{ID: 224, Key: "Load_Friction", Name: "Load Friction", Critical: false, Description: "Mechanical friction rate", PromName: "disk_load_friction", PromHelp: "Rate of friction between mechanical parts"},
	{ID: 226, Key: "Load-in_Time", Name: "Load-in Time", Critical: false, Description: "Total time the heads are loaded", PromName: "disk_load_in_time", PromHelp: "Total time the heads are loaded"},
	{ID: 227, Key: "Torque_Amplification_Count", Name: "Torque Amplification Count", Critical: false, Description: "Rate of torque increase", PromName: "disk_torque_amplification_count", PromHelp: "Rate of torque increase during spin up"},
	{ID: 228, Key: "Power-off_Retract_Count", Name: "Power-off Retract Count", Critical: false, Description: "Number of power off cycles", PromName: "disk_power_off_retract_count", PromHelp: "Number of times the head was retracted as a result of power loss"},
	{ID: 230, Key: "GMR_Head_Amplitude", Name: "GMR Head Amplitude", Critical: false, Description: "Head positioning amplitude", PromName: "disk_gmr_head_amplitude", PromHelp: "Head moving distances between operations"},
	{ID: 231, Key: "Temperature_Celsius", Name: "Temperature Celsius", Critical: false, Description: "Disk temperature", PromName: "disk_temperature_celsius", PromHelp: "Temperature inside the hard disk housing"},
	// {ID: 232, Key: "Available_Reserved_Space", Name: "Available Reserved Space", Critical: false, Description: "Amount of reserved space available", PromName: "disk_available_reserved_space", PromHelp: "Amount of reserved space available"},
	// {ID: 233, Key: "Media_Wearout_Indicator", Name: "Media Wearout Indicator", Critical: false, Description: "Indicates the wearout of the NAND flash", PromName: "disk_media_wearout_indicator", PromHelp: "Indicates the wearout of the NAND flash"},
	// {ID: 234, Key: "Good_Block_Count_And_Bad_Block_Count", Name: "Good Block Count and Bad Block Count", Critical: false, Description: "Counts of good and bad blocks", PromName: "disk_good_bad_block_count", PromHelp: "Counts of good and bad blocks"},
	// {ID: 235, Key: "Percentage_Of_Used_Reserve", Name: "Percentage of Used Reserve", Critical: false, Description: "Percentage of the used reserved space", PromName: "disk_percentage_used_reserve", PromHelp: "Percentage of the used reserved space"},
	{ID: 240, Key: "Head_Flying_Hours", Name: "Head Flying Hours", Critical: false, Description: "Number of head positioning hours", PromName: "disk_head_flying_hours", PromHelp: "Time spent during the positioning of the drive heads"},
	// {ID: 241, Key: "Total_LBAs_Written", Name: "Total LBAs Written", Critical: false, Description: "Total number of LBAs written", PromName: "disk_total_lbas_written", PromHelp: "Total number of LBAs written"},
	// {ID: 242, Key: "Total_LBAs_Read", Name: "Total LBAs Read", Critical: false, Description: "Total number of LBAs read", PromName: "disk_total_lbas_read", PromHelp: "Total number of LBAs read"},
	// {ID: 243, Key: "Total_Host_Sector_Writes", Name: "Total Host Sector Writes", Critical: false, Description: "Total number of host sector writes", PromName: "disk_total_host_sector_writes", PromHelp: "Total number of host sector writes"},
	// {ID: 244, Key: "Total_Host_Sector_Reads", Name: "Total Host Sector Reads", Critical: false, Description: "Total number of host sector reads", PromName: "disk_total_host_sector_reads", PromHelp: "Total number of host sector reads"},
	// {ID: 249, Key: "NAND_Writes_1GiB", Name: "NAND Writes (1GiB)", Critical: false, Description: "Number of writes to NAND (1GiB)", PromName: "disk_nand_writes_1gib", PromHelp: "Number of writes to NAND (1GiB)"},
	{ID: 250, Key: "Read_Error_Retry_Rate", Name: "Read Error Retry Rate", Critical: false, Description: "Number of retries during read operations", PromName: "disk_read_error_retry_rate", PromHelp: "Number of errors found during reading a sector from disk surface"},
	// {ID: 251, Key: "Minimum_Spares_Remaining", Name: "Minimum Spares Remaining", Critical: false, Description: "Minimum number of spare blocks remaining", PromName: "disk_minimum_spares_remaining", PromHelp: "Minimum number of spare blocks remaining"},
	// {ID: 252, Key: "Newly_Added_Bad_Flash_Block", Name: "Newly Added Bad Flash Block", Critical: false, Description: "Number of newly added bad flash blocks", PromName: "disk_newly_added_bad_flash_block", PromHelp: "Number of newly added bad flash blocks"},
	// {ID: 254, Key: "Free_Fall_Protection", Name: "Free Fall Protection", Critical: false, Description: "Free fall protection enabled", PromName: "disk_free_fall_protection", PromHelp: "Free fall protection enabled"},
}
