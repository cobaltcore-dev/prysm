// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and prysm contributors
//
// SPDX-License-Identifier: Apache-2.0

package opslog

import "strings"

type SLOperation string

const (
	SLOperationGet  SLOperation = "get"
	SLOperationList SLOperation = "list"
)

func classifyBucketSLOOperation(operation string) (SLOperation, bool) {
	switch strings.ToLower(operation) {
	case "get_obj", "head_obj":
		return SLOperationGet, true
	case "list_bucket", "list_buckets", "get_bucket_info":
		return SLOperationList, true
	default:
		return "", false
	}
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

func observeBucketSLI(logEntry S3OperationLog, tenant string) {
	sloOperation, ok := classifyBucketSLOOperation(logEntry.Operation)
	if !ok || logEntry.Bucket == "" {
		return
	}

	sliRequestsTotal.WithLabelValues(
		tenant,
		logEntry.Bucket,
		string(sloOperation),
		statusClass(logEntry.HTTPStatus),
	).Inc()

	if logEntry.TotalTime > 0 {
		sliRequestDuration.WithLabelValues(
			tenant,
			logEntry.Bucket,
			string(sloOperation),
		).Observe(float64(logEntry.TotalTime) / 1000.0)
	}
}
