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
	BytesSent     uint64    // Total bytes sent at the previous collection time
	BytesReceived uint64    // Total bytes received at the previous collection time
	ReadOpsTotal  uint64    // Total read operations at the previous collection time
	WriteOpsTotal uint64    // Total write operations at the previous collection time
	DataSizeTotal uint64    // Total data size of the bucket at the previous collection time
	ObjectsTotal  uint64    // Total number of objects at the previous collection time
	LastCollected time.Time // Timestamp of the last collection for this bucket
}

// Map to store previous usage for each user
var previousUserUsage = make(map[string]PreviousUsage)

// Map to store previous usage for each bucket
var previousBucketUsage = make(map[string]PreviousUsage)

// Function to calculate and return current values (deltas) and throughput
func CalculateCurrentUserMetrics(userID string, currentBytesSent, currentBytesReceived, currentReadOps, currentWriteOps uint64, currentTime time.Time) CurrentUserUsageMetrics {
	prevUsage, exists := previousUserUsage[userID]

	var throughput, currentBytesSentRate, currentBytesReceivedRate, currentOpsRate float64

	// If this is the first collection, initialize the values
	if !exists {
		previousUserUsage[userID] = PreviousUsage{
			BytesSent:     currentBytesSent,
			BytesReceived: currentBytesReceived,
			ReadOpsTotal:  currentReadOps,
			WriteOpsTotal: currentWriteOps,
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

	// Calculate deltas (difference in bytes sent, received, read/write ops)
	readOpsDelta := currentReadOps - prevUsage.ReadOpsTotal
	writeOpsDelta := currentWriteOps - prevUsage.WriteOpsTotal
	bytesSentDelta := currentBytesSent - prevUsage.BytesSent
	bytesReceivedDelta := currentBytesReceived - prevUsage.BytesReceived

	// Calculate time difference in seconds
	deltaTime := currentTime.Sub(prevUsage.LastCollected).Seconds()

	if deltaTime > 0 {
		if bytesSentDelta > 0 || bytesReceivedDelta > 0 {
			// Calculate throughput in bytes per second (sent + received)
			throughput = float64(bytesSentDelta+bytesReceivedDelta) / deltaTime

			// Calculate bytes sent/received per second
			currentBytesSentRate = float64(bytesSentDelta) / deltaTime
			currentBytesReceivedRate = float64(bytesReceivedDelta) / deltaTime
		} else {
			// Reset values if no bytes were sent or received
			currentBytesSentRate = 0
			currentBytesReceivedRate = 0
			throughput = 0
		}

		// Calculate total operations (read + write) per second if there is activity
		if readOpsDelta > 0 || writeOpsDelta > 0 {
			currentOpsRate = float64(readOpsDelta+writeOpsDelta) / deltaTime
		} else {
			// Reset operations rate if there are no operations
			currentOpsRate = 0
		}
	}

	// Update previous usage data with current values
	previousUserUsage[userID] = PreviousUsage{
		BytesSent:     currentBytesSent,
		BytesReceived: currentBytesReceived,
		LastCollected: currentTime,
		ReadOpsTotal:  currentReadOps,
		WriteOpsTotal: currentWriteOps,
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

// Function to calculate and return current bucket-level metrics, including throughput and operations per second
func CalculateCurrentBucketMetrics(bucketName string, currentBytesSent, currentBytesReceived, currentReadOps, currentWriteOps uint64, currentTime time.Time) RadosGWBucketMetrics {
	prevUsage, exists := previousBucketUsage[bucketName]

	var throughput, currentBytesSentRate, currentBytesReceivedRate, currentOpsRate, currentReadOpsRate, currentWriteOpsRate float64

	// If this is the first collection, initialize the values
	if !exists {
		previousBucketUsage[bucketName] = PreviousUsage{
			BytesSent:     currentBytesSent,
			BytesReceived: currentBytesReceived,
			LastCollected: currentTime,
			ReadOpsTotal:  currentReadOps,
			WriteOpsTotal: currentWriteOps,
		}
		return RadosGWBucketMetrics{
			Meta: struct {
				Name      string
				Owner     string
				Zonegroup string
				Shards    *uint64
				CreatedAt *time.Time
			}{
				Name: bucketName,
			},
			Totals: struct {
				DataSize      uint64
				UtilizedSize  uint64
				Objects       uint64
				ReadOps       uint64
				WriteOps      uint64
				BytesSent     uint64
				BytesReceived uint64
				SuccessOps    uint64
				OpsTotal      uint64
				ErrorRate     float64
				Capacity      uint64
			}{
				BytesSent:     currentBytesSent,
				BytesReceived: currentBytesReceived,
				ReadOps:       currentReadOps,
				WriteOps:      currentWriteOps,
			},
			Current: struct {
				OpsPerSec             float64
				ReadOpsPerSec         float64
				WriteOpsPerSec        float64
				BytesSentPerSec       float64
				BytesReceivedPerSec   float64
				ThroughputBytesPerSec float64
			}{},
		}
	}

	// Calculate deltas (difference in bytes sent, received, read/write ops)
	readOpsDelta := currentReadOps - prevUsage.ReadOpsTotal
	writeOpsDelta := currentWriteOps - prevUsage.WriteOpsTotal
	bytesSentDelta := currentBytesSent - prevUsage.BytesSent
	bytesReceivedDelta := currentBytesReceived - prevUsage.BytesReceived

	// Calculate time difference in seconds
	deltaTime := currentTime.Sub(prevUsage.LastCollected).Seconds()

	if deltaTime > 0 {
		if bytesSentDelta > 0 || bytesReceivedDelta > 0 {
			// Calculate throughput in bytes per second (sent + received)
			throughput = float64(bytesSentDelta+bytesReceivedDelta) / deltaTime

			// Calculate bytes sent/received per second
			currentBytesSentRate = float64(bytesSentDelta) / deltaTime
			currentBytesReceivedRate = float64(bytesReceivedDelta) / deltaTime
		} else {
			// Reset values if no bytes were sent or received
			currentBytesSentRate = 0
			currentBytesReceivedRate = 0
			throughput = 0
		}

		// Calculate total operations (read + write) per second if there is activity
		if readOpsDelta > 0 || writeOpsDelta > 0 {
			currentOpsRate = float64(readOpsDelta+writeOpsDelta) / deltaTime
		} else {
			// Reset operations rate if there are no operations
			currentOpsRate = 0
		}
	} else {
		// Reset rates if deltaTime is zero or invalid
		currentReadOpsRate = 0
		currentWriteOpsRate = 0
		currentOpsRate = 0
		currentBytesSentRate = 0
		currentBytesReceivedRate = 0
		throughput = 0
	}

	// Update the previous usage data with the current values
	previousBucketUsage[bucketName] = PreviousUsage{
		BytesSent:     currentBytesSent,
		BytesReceived: currentBytesReceived,
		LastCollected: currentTime,
		ReadOpsTotal:  currentReadOps,
		WriteOpsTotal: currentWriteOps,
	}

	return RadosGWBucketMetrics{
		Meta: struct {
			Name      string
			Owner     string
			Zonegroup string
			Shards    *uint64
			CreatedAt *time.Time
		}{
			Name: bucketName,
		},
		Current: struct {
			OpsPerSec             float64
			ReadOpsPerSec         float64
			WriteOpsPerSec        float64
			BytesSentPerSec       float64
			BytesReceivedPerSec   float64
			ThroughputBytesPerSec float64
		}{
			OpsPerSec:             currentOpsRate,
			ReadOpsPerSec:         currentReadOpsRate,
			WriteOpsPerSec:        currentWriteOpsRate,
			BytesSentPerSec:       currentBytesSentRate,
			BytesReceivedPerSec:   currentBytesReceivedRate,
			ThroughputBytesPerSec: throughput,
		},
		Totals: struct {
			DataSize      uint64
			UtilizedSize  uint64
			Objects       uint64
			ReadOps       uint64
			WriteOps      uint64
			BytesSent     uint64
			BytesReceived uint64
			SuccessOps    uint64
			OpsTotal      uint64
			ErrorRate     float64
			Capacity      uint64
		}{
			BytesSent:     currentBytesSent,
			BytesReceived: currentBytesReceived,
			ReadOps:       currentReadOps,
			WriteOps:      currentWriteOps,
		},
	}
}
