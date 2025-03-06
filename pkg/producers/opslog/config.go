// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and prysm contributors
//
// SPDX-License-Identifier: Apache-2.0

package opslog

type OpsLogConfig struct {
	LogFilePath             string
	SocketPath              string
	NatsURL                 string
	NatsSubject             string
	NatsMetricsSubject      string
	UseNats                 bool
	LogToStdout             bool
	LogRetentionDays        int   // Number of days to keep old log files
	MaxLogFileSize          int64 // Maximum log file size in bytes before rotation
	Prometheus              bool
	PrometheusPort          int
	PodName                 string
	IgnoreAnonymousRequests bool
}
