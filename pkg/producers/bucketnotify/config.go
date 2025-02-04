// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and prysm contributors
//
// SPDX-License-Identifier: Apache-2.0

package bucketnotify

type BucketNotifyConfig struct {
	EndpointPort int
	NatsURL      string
	NatsSubject  string
	UseNats      bool
}
