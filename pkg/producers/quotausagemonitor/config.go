// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and prysm contributors
//
// SPDX-License-Identifier: Apache-2.0

package quotausagemonitor

type QuotaUsageMonitorConfig struct {
	AdminURL          string
	AccessKey         string
	SecretKey         string
	NatsURL           string
	NatsSubject       string
	UseNats           bool
	Interval          int
	NodeName          string
	InstanceID        string
	QuotaUsagePercent float64
}
