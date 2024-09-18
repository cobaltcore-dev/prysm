// Copyright 2024 Clyso GmbH
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package radosgwusage

import "time"

type CurrentUserUsageMetrics struct {
	TotalBytesSent       uint64
	TotalBytesReceived   uint64
	CurrentBytesSent     uint64
	CurrentBytesReceived uint64
	Throughput           float64
	CurrentOps           uint64
}

type PreviousUsage struct {
	BytesSent     uint64
	BytesReceived uint64
	OpsTotal      uint64
	LastCollected time.Time
}

// Map to store previous usage for each user
var previousUsage = make(map[string]PreviousUsage)

// Function to calculate and return current values (deltas) and throughput
func CalculateCurrentUserMetrics(userID string, currentBytesSent, currentBytesReceived, currentOps uint64, currentTime time.Time) CurrentUserUsageMetrics {
	prevUsage, exists := previousUsage[userID]

	var currentBytesSentDelta, currentBytesReceivedDelta, currentOpsDelta uint64
	var currentBytesSentRate, currentBytesReceivedRate float64
	var throughput float64
	var currentOpsRate float64

	// If this is the first collection, initialize the values
	if !exists {
		previousUsage[userID] = PreviousUsage{
			BytesSent:     currentBytesSent,
			BytesReceived: currentBytesReceived,
			OpsTotal:      currentOps,
			LastCollected: currentTime,
		}
		return CurrentUserUsageMetrics{
			TotalBytesSent:       currentBytesSent,
			TotalBytesReceived:   currentBytesReceived,
			CurrentBytesSent:     0, // No delta for the first collection
			CurrentBytesReceived: 0,
			Throughput:           0.0,
			CurrentOps:           0,
		}
	}

	// Calculate delta (difference in bytes sent and received)
	currentBytesSentDelta = currentBytesSent - prevUsage.BytesSent
	currentBytesReceivedDelta = currentBytesReceived - prevUsage.BytesReceived
	currentOpsDelta = currentOps - prevUsage.OpsTotal

	// Calculate time difference in seconds
	deltaTime := currentTime.Sub(prevUsage.LastCollected).Seconds()

	if deltaTime > 0 {
		if currentBytesSentDelta > 0 || currentBytesReceivedDelta > 0 {
			// Calculate throughput in bytes per second if there is activity
			throughput = float64(currentBytesSentDelta+currentBytesReceivedDelta) / deltaTime

			// Calculate current bytes sent/received rates (bytes per second) if there is activity
			currentBytesSentRate = float64(currentBytesSentDelta) / deltaTime
			currentBytesReceivedRate = float64(currentBytesReceivedDelta) / deltaTime
		} else {
			// Reset values if no bytes were sent or received
			currentBytesSentRate = 0
			currentBytesReceivedRate = 0
			throughput = 0
		}

		if currentOpsDelta > 0 {
			// Calculate current operations per second if there is activity
			currentOpsRate = float64(currentOpsDelta) / deltaTime
		} else {
			// Reset ops rate if there are no operations
			currentOpsRate = 0
		}
	}

	// Store the new totals and collection time
	previousUsage[userID] = PreviousUsage{
		BytesSent:     currentBytesSent,
		BytesReceived: currentBytesReceived,
		LastCollected: currentTime,
		OpsTotal:      currentOps,
	}

	return CurrentUserUsageMetrics{
		TotalBytesSent:       currentBytesSent,
		TotalBytesReceived:   currentBytesReceived,
		CurrentBytesSent:     uint64(currentBytesSentRate),
		CurrentBytesReceived: uint64(currentBytesReceivedRate),
		Throughput:           throughput,
		CurrentOps:           uint64(currentOpsRate),
	}
}
