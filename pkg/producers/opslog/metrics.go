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

package opslog

import "sync"

type Metrics struct {
	TotalRequests         int
	RequestsByOperation   map[string]int
	RequestsByStatusCode  map[string]int
	BytesSent             int
	BytesReceived         int
	Errors                int
	LatencySum            float64
	LatencyCount          int
	RequestsByUser        map[string]int
	BytesSentByUser       map[string]int
	BytesReceivedByUser   map[string]int
	RequestsByBucket      map[string]int
	BytesSentByBucket     map[string]int
	BytesReceivedByBucket map[string]int
	RequestsByIP          map[string]int
	BytesSentByIP         map[string]int
	BytesReceivedByIP     map[string]int
	MaxLatency            float64
	MinLatency            float64
	mu                    sync.Mutex
}

func NewMetrics() *Metrics {
	return &Metrics{
		RequestsByOperation:   make(map[string]int),
		RequestsByStatusCode:  make(map[string]int),
		RequestsByUser:        make(map[string]int),
		BytesSentByUser:       make(map[string]int),
		BytesReceivedByUser:   make(map[string]int),
		RequestsByBucket:      make(map[string]int),
		BytesSentByBucket:     make(map[string]int),
		BytesReceivedByBucket: make(map[string]int),
		RequestsByIP:          make(map[string]int),
		BytesSentByIP:         make(map[string]int),
		BytesReceivedByIP:     make(map[string]int),
		MaxLatency:            -1, // set to -1 to indicate no value yet
		MinLatency:            -1, // set to -1 to indicate no value yet
	}
}

func (m *Metrics) Update(logEntry S3OperationLog) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.TotalRequests++

	// Increment operation count
	m.RequestsByOperation[logEntry.Operation]++

	// Increment status code count
	m.RequestsByStatusCode[logEntry.HTTPStatus]++

	// Accumulate bytes sent and received
	m.BytesSent += logEntry.BytesSent
	m.BytesReceived += logEntry.BytesReceived

	// Track by user
	m.RequestsByUser[logEntry.User]++
	m.BytesSentByUser[logEntry.User] += logEntry.BytesSent
	m.BytesReceivedByUser[logEntry.User] += logEntry.BytesReceived

	// Track by bucket
	m.RequestsByBucket[logEntry.Bucket]++
	m.BytesSentByBucket[logEntry.Bucket] += logEntry.BytesSent
	m.BytesReceivedByBucket[logEntry.Bucket] += logEntry.BytesReceived

	// Track by IP address
	m.RequestsByIP[logEntry.RemoteAddr]++
	m.BytesSentByIP[logEntry.RemoteAddr] += logEntry.BytesSent
	m.BytesReceivedByIP[logEntry.RemoteAddr] += logEntry.BytesReceived

	// Track errors
	if logEntry.HTTPStatus[0] != '2' { // assuming non-2xx status codes are errors
		m.Errors++
	}

	// Accumulate latency
	if logEntry.TotalTime > 0 {
		latencySeconds := float64(logEntry.TotalTime) / 1000.0
		m.LatencySum += latencySeconds
		m.LatencyCount++

		if m.MaxLatency == -1 || latencySeconds > m.MaxLatency {
			m.MaxLatency = latencySeconds
		}

		if m.MinLatency == -1 || latencySeconds < m.MinLatency {
			m.MinLatency = latencySeconds
		}
	}
}
