// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and prysm contributors
//
// SPDX-License-Identifier: Apache-2.0

package diskhealthmetrics

// NVMeIDControllerOutput represents the JSON output from nvme id-ctrl -o json
type NVMeIDControllerOutput struct {
	VendorID            int64  `json:"vid"`
	SubsystemVendorID   int64  `json:"ssvid"`
	ModelNumber         string `json:"mn"`
	SerialNumber        string `json:"sn"`
	FirmwareRevision    string `json:"fr"`
	SubsystemNQN        string `json:"subnqn"`
	IEEE                string `json:"ieee"`
	TotalCapacity       int64  `json:"tnvmcap"`
	UnallocatedCapacity int64  `json:"unvmcap"`
}

// NVMeErrorLogOutput represents the JSON output from nvme error-log -o json
type NVMeErrorLogOutput struct {
	Errors []NVMeErrorEntry `json:"errors"`
}

// NVMeErrorEntry represents a single error entry from the error log
type NVMeErrorEntry struct {
	ErrorCount                int64 `json:"error_count"`
	SubmissionQueueID         int64 `json:"sqid"`
	CommandID                 int64 `json:"cmdid"`
	StatusField               int64 `json:"status_field"`
	PhaseTag                  int64 `json:"phase_tag"`
	ParameterErrorLocation    int64 `json:"parm_error_location"`
	LBA                       int64 `json:"lba"`
	Namespace                 int64 `json:"nsid"`
	VendorSpecific            int64 `json:"vs"`
	TransportType             int64 `json:"trtype"`
	CommandSpecific           int64 `json:"cs"`
	TransportTypeSpecificInfo int64 `json:"trtype_spec_info"`
}
