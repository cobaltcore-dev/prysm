// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and prysm contributors
//
// SPDX-License-Identifier: Apache-2.0

package opslog

import "strings"

// SLIOperation classifies RGW operations into the ADR-defined operation categories.
type SLIOperation string

const (
	SLIOperationGet       SLIOperation = "get"
	SLIOperationPut       SLIOperation = "put"
	SLIOperationList      SLIOperation = "list"
	SLIOperationDelete    SLIOperation = "delete"
	SLIOperationHead      SLIOperation = "head"
	SLIOperationMultipart SLIOperation = "multipart"
)

// SLIProtocol represents the access protocol (S3 or Swift).
type SLIProtocol string

const (
	SLIProtocolS3    SLIProtocol = "s3"
	SLIProtocolSwift SLIProtocol = "swift"
)

// classifySLIOperation maps RGW operation names to SLI operation categories.
// Returns the operation category and whether it's a recognized SLI-relevant operation.
func classifySLIOperation(operation string) (SLIOperation, bool) {
	op := strings.ToLower(operation)

	// Strip swift_ prefix for classification (protocol detected separately)
	op = strings.TrimPrefix(op, "swift_")

	switch {
	// GET operations
	case op == "get_obj" || op == "get_object":
		return SLIOperationGet, true

	// HEAD operations
	case op == "head_obj" || op == "head_object" || op == "head_bucket":
		return SLIOperationHead, true

	// PUT operations
	case op == "put_obj" || op == "put_object" || op == "copy_obj" || op == "post_object":
		return SLIOperationPut, true

	// DELETE operations
	case op == "delete_obj" || op == "delete_object" || op == "delete_bucket" ||
		op == "delete_multi_obj" || op == "multi_object_delete":
		return SLIOperationDelete, true

	// LIST operations
	case op == "list_bucket" || op == "list_buckets" || op == "list_objects" ||
		op == "get_bucket_info" || op == "list_bucket_versions" ||
		op == "list_bucket_multiparts":
		return SLIOperationList, true

	// Multipart operations
	case op == "complete_multipart" || op == "init_multipart" ||
		op == "abort_multipart" || op == "upload_part" ||
		op == "list_multipart" || op == "list_bucket_multiparts":
		return SLIOperationMultipart, true

	default:
		return "", false
	}
}

// detectProtocol determines the access protocol from the RGW operation name.
// In Ceph RGW ops logs, Swift operations are prefixed with "swift_".
func detectProtocol(operation string) SLIProtocol {
	if strings.HasPrefix(strings.ToLower(operation), "swift_") {
		return SLIProtocolSwift
	}
	return SLIProtocolS3
}

func statusClass(status string) string {
	if len(status) == 3 &&
		status[0] >= '1' && status[0] <= '5' &&
		status[1] >= '0' && status[1] <= '9' &&
		status[2] >= '0' && status[2] <= '9' {
		return string(status[0]) + "xx"
	}
	return "unknown"
}

// observeSLI records per-tenant SLI request count and latency metrics.
// The SLI is keyed by tenant (not bucket), with protocol and operation labels.
// Anonymous requests (tenant="none") are excluded from the SLI.
func observeSLI(logEntry S3OperationLog, tenant string) {
	if globalSLICollector == nil {
		return
	}

	sliOperation, ok := classifySLIOperation(logEntry.Operation)
	if tenant == "none" || !ok {
		return
	}

	protocol := detectProtocol(logEntry.Operation)

	globalSLICollector.observeCounter(
		tenant,
		string(protocol),
		string(sliOperation),
		statusClass(logEntry.HTTPStatus),
	)

	// Observe latency if available
	if logEntry.TotalTime > 0 {
		latencySec := float64(logEntry.TotalTime) / 1000.0
		globalSLICollector.observeLatency(
			tenant,
			string(protocol),
			string(sliOperation),
			latencySec,
		)
	}
}
