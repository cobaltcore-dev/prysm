// Copyright (c) 2024 Clyso GmbH
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program. If not, see <https://www.gnu.org/licenses/>.

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
