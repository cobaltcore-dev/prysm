// SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and prysm contributors
//
// SPDX-License-Identifier: Apache-2.0

package diskhealthmetrics

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// loadScenario reads one of the smartctl fixtures under testdata/scenarios.
func loadScenario(t *testing.T, scenario, device string) *SmartCtlOutput {
	t.Helper()

	path := filepath.Join("testdata", "scenarios", scenario, device+".json")
	data, err := collectSmartDataFromFile(path)
	require.NoError(t, err, "fixture %s must load", path)
	require.NotNil(t, data)
	return data
}

// processFixture runs the full attribute pipeline the producer uses, including
// the cleanup pass that drops attributes left at their -1 sentinel.
func processFixture(t *testing.T, scenario, device string) map[string]SmartAttribute {
	t.Helper()

	smartAttrs := GetSmartAttributes()
	ProcessAndUpdateSmartAttributes(smartAttrs, loadScenario(t, scenario, device))
	CleanupSmartAttributes(smartAttrs)
	return smartAttrs
}

// TestNVMeDataProcessedAttributes covers the gap behind the empty "Top 10 - Data
// Read/Write Processed" dashboard panels: NVMe drives report throughput as
// data_units_read/written in the NVMe health log, but only the SCSI error
// counter log ever populated read_/write_gigabytes_processed. On NVMe the two
// attributes stayed at their -1 sentinel and CleanupSmartAttributes deleted
// them, so the series never reached Prometheus at all.
func TestNVMeDataProcessedAttributes(t *testing.T) {
	testCases := []struct {
		scenario      string
		device        string
		expectedRead  int64
		expectedWrite int64
	}{
		// Expected values are data_units * 512000 / 1e9, truncated — the same
		// decimal-GB conversion smartctl prints for NVMe.
		{scenario: "healthy", device: "nvme0", expectedRead: 29853, expectedWrite: 23317},
		{scenario: "healthy", device: "nvme1", expectedRead: 29853, expectedWrite: 23317},
		{scenario: "mixed", device: "nvme0", expectedRead: 81053, expectedWrite: 125717},
		{scenario: "mixed", device: "nvme1", expectedRead: 14493, expectedWrite: 7957},
		{scenario: "failing", device: "nvme0", expectedRead: 439453, expectedWrite: 484117},
		{scenario: "failing", device: "nvme1", expectedRead: 644253, expectedWrite: 1252117},
	}

	for _, tc := range testCases {
		t.Run(tc.scenario+"/"+tc.device, func(t *testing.T) {
			smartAttrs := processFixture(t, tc.scenario, tc.device)

			readAttr, ok := smartAttrs["read_gigabytes_processed"]
			require.True(t, ok, "read_gigabytes_processed must survive cleanup for NVMe devices")
			assert.Equal(t, tc.expectedRead, readAttr.RawValue, "read GB (RawValue is what Prometheus exports)")
			assert.Equal(t, tc.expectedRead, readAttr.Value, "read GB")
			assert.Equal(t, "GB", readAttr.Unit, "must match the decimal GB the SCSI path reports")

			writeAttr, ok := smartAttrs["write_gigabytes_processed"]
			require.True(t, ok, "write_gigabytes_processed must survive cleanup for NVMe devices")
			assert.Equal(t, tc.expectedWrite, writeAttr.RawValue, "write GB (RawValue is what Prometheus exports)")
			assert.Equal(t, tc.expectedWrite, writeAttr.Value, "write GB")
			assert.Equal(t, "GB", writeAttr.Unit, "must match the decimal GB the SCSI path reports")
		})
	}
}

// TestSCSIDataProcessedAttributes pins the pre-existing SAS/SCSI behaviour so
// the NVMe addition cannot regress it. Values come straight from
// scsi_error_counter_log.{read,write}.gigabytes_processed, truncated to int64.
func TestSCSIDataProcessedAttributes(t *testing.T) {
	testCases := []struct {
		scenario      string
		expectedRead  int64
		expectedWrite int64
	}{
		{scenario: "healthy", expectedRead: 84858, expectedWrite: 23105},
		{scenario: "mixed", expectedRead: 12345, expectedWrite: 6789},
		{scenario: "failing", expectedRead: 484858, expectedWrite: 123105},
	}

	for _, tc := range testCases {
		t.Run(tc.scenario+"/sdb", func(t *testing.T) {
			smartAttrs := processFixture(t, tc.scenario, "sdb")

			readAttr, ok := smartAttrs["read_gigabytes_processed"]
			require.True(t, ok, "read_gigabytes_processed must be present for SCSI devices")
			assert.Equal(t, tc.expectedRead, readAttr.RawValue)
			assert.Equal(t, "GB", readAttr.Unit)

			writeAttr, ok := smartAttrs["write_gigabytes_processed"]
			require.True(t, ok, "write_gigabytes_processed must be present for SCSI devices")
			assert.Equal(t, tc.expectedWrite, writeAttr.RawValue)
			assert.Equal(t, "GB", writeAttr.Unit)
		})
	}
}

// TestNVMeDataUnitsToGigabytes pins the unit conversion itself. One NVMe data
// unit is 512000 bytes, so 1 GB is 1953.125 data units — values below that
// floor to zero, which is correct for a counter reported in whole GB.
func TestNVMeDataUnitsToGigabytes(t *testing.T) {
	testCases := []struct {
		name      string
		dataUnits int64
		expected  int64
	}{
		{name: "zero", dataUnits: 0, expected: 0},
		{name: "negative is clamped", dataUnits: -1, expected: 0},
		{name: "single unit floors to zero", dataUnits: 1, expected: 0},
		{name: "just under one GB", dataUnits: 1953, expected: 0},
		{name: "just over one GB", dataUnits: 1954, expected: 1},
		{name: "exactly one TB", dataUnits: 1953125, expected: 1000},
		{name: "healthy fixture read", dataUnits: 58306932, expected: 29853},
		{name: "healthy fixture write", dataUnits: 45541512, expected: 23317},
		{name: "petabyte scale", dataUnits: 1953125000, expected: 1000000},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, nvmeDataUnitsToGigabytes(tc.dataUnits))
		})
	}
}

// TestNVMeDataProcessedDoesNotLeakIntoATA guards the cleanup contract: a pure
// ATA fixture has neither a SCSI error counter log nor an NVMe health log, so
// both attributes must stay absent rather than being exported as 0.
func TestNVMeDataProcessedDoesNotLeakIntoATA(t *testing.T) {
	smartAttrs := processFixture(t, "healthy", "sda")

	assert.NotContains(t, smartAttrs, "read_gigabytes_processed",
		"ATA devices report no throughput counters; the attribute must be dropped, not zeroed")
	assert.NotContains(t, smartAttrs, "write_gigabytes_processed",
		"ATA devices report no throughput counters; the attribute must be dropped, not zeroed")
}
