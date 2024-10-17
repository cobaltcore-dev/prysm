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

import (
	"time"
)

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

type APIUsage struct {
	Categories    map[string]uint64 // Map of API categories and their corresponding operation counts
	LastCollected time.Time         // Timestamp of the last collection for API usage
}

type PreviousAPIUsage struct {
	Usage         map[string]uint64 // API category and previous usage count
	LastCollected time.Time         // Timestamp of the last collection
}

// Map to store previous usage for each user
var previousUserUsage = make(map[string]PreviousUsage)

// Map to store previous API usage for each user
var previousUserAPIUsage = make(map[string]APIUsage)

// Map to store previous usage for each bucket
var previousBucketUsage = make(map[string]PreviousUsage)

// Map to store previous API usage for each bucket
var previousBucketAPIUsage = make(map[string]PreviousAPIUsage)

// Function to calculate and return current values (deltas) and throughput
func CalculateCurrentUserMetrics(userLevel *RadosGWUserMetrics, currentTime time.Time) {
	prevUsage, exists := previousUserUsage[userLevel.Meta.ID]

	var throughput, currentBytesSentRate, currentBytesReceivedRate, currentOpsRate float64

	// If this is the first collection, initialize the values
	if !exists {
		previousUserUsage[userLevel.Meta.ID] = PreviousUsage{
			BytesSent:     userLevel.Totals.BytesSentTotal,
			BytesReceived: userLevel.Totals.BytesReceivedTotal,
			ReadOpsTotal:  userLevel.Totals.ReadOpsTotal,
			WriteOpsTotal: userLevel.Totals.WriteOpsTotal,
			LastCollected: currentTime,
		}
		// Initialize current metrics with zeros
		userLevel.Current.OpsPerSec = 0.0
		userLevel.Current.ThroughputBytesPerSec = 0.0
		userLevel.Current.DataBytesSentPerSec = 0.0
		userLevel.Current.DataBytesReceivedPerSec = 0.0
		return
	}

	// Calculate deltas (difference in bytes sent, received, read/write ops)
	readOpsDelta := userLevel.Totals.ReadOpsTotal - prevUsage.ReadOpsTotal
	writeOpsDelta := userLevel.Totals.WriteOpsTotal - prevUsage.WriteOpsTotal
	bytesSentDelta := userLevel.Totals.BytesSentTotal - prevUsage.BytesSent
	bytesReceivedDelta := userLevel.Totals.BytesReceivedTotal - prevUsage.BytesReceived

	// Ensure we don't get negative deltas (possible if counters are reset)
	if userLevel.Totals.ReadOpsTotal < prevUsage.ReadOpsTotal {
		readOpsDelta = 0
	}
	if userLevel.Totals.WriteOpsTotal < prevUsage.WriteOpsTotal {
		writeOpsDelta = 0
	}
	if userLevel.Totals.BytesSentTotal < prevUsage.BytesSent {
		bytesSentDelta = 0
	}
	if userLevel.Totals.BytesReceivedTotal < prevUsage.BytesReceived {
		bytesReceivedDelta = 0
	}

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
			currentBytesSentRate = 0.0
			currentBytesReceivedRate = 0.0
			throughput = 0.0
		}

		// Calculate total operations (read + write) per second if there is activity
		if readOpsDelta > 0 || writeOpsDelta > 0 {
			currentOpsRate = float64(readOpsDelta+writeOpsDelta) / deltaTime
		} else {
			// Reset operations rate if there are no operations
			currentOpsRate = 0.0
		}
	} else {
		// If deltaTime is zero or negative, reset the rates
		currentOpsRate = 0.0
		throughput = 0.0
		currentBytesSentRate = 0.0
		currentBytesReceivedRate = 0.0
	}

	// Update previous usage data with current values
	previousUserUsage[userLevel.Meta.ID] = PreviousUsage{
		BytesSent:     userLevel.Totals.BytesSentTotal,
		BytesReceived: userLevel.Totals.BytesReceivedTotal,
		LastCollected: currentTime,
		ReadOpsTotal:  userLevel.Totals.ReadOpsTotal,
		WriteOpsTotal: userLevel.Totals.WriteOpsTotal,
	}

	// Update the current metrics in the userLevel object
	userLevel.Current.OpsPerSec = currentOpsRate
	userLevel.Current.ThroughputBytesPerSec = throughput
	userLevel.Current.DataBytesSentPerSec = currentBytesSentRate
	userLevel.Current.DataBytesReceivedPerSec = currentBytesReceivedRate
}

func CalculateCurrentAPIUsagePerUser(userLevel *RadosGWUserMetrics, currentTime time.Time) {
	prevUsage, exists := previousUserAPIUsage[userLevel.Meta.ID]

	if !exists {
		// Initialize the previous usage if it doesn't exist
		previousUserAPIUsage[userLevel.Meta.ID] = APIUsage{
			Categories:    userLevel.APIUsagePerUser, // Set current usage as previous
			LastCollected: currentTime,
		}
		return
	}

	// Calculate the time difference in seconds
	deltaTime := currentTime.Sub(prevUsage.LastCollected).Seconds()

	totalAPIRatePerSec := 0.0

	if deltaTime > 0 {
		// Calculate API usage per second for each category
		for category, currentOps := range userLevel.APIUsagePerUser {
			prevOps, found := prevUsage.Categories[category]
			if found {
				opsDelta := currentOps - prevOps

				// Calculate per-second rate and update current usage
				apiRatePerSec := float64(opsDelta) / deltaTime
				userLevel.Current.APIUsagePerSec[category] = apiRatePerSec

				// Add to the total API usage per second
				totalAPIRatePerSec += apiRatePerSec
			}
		}
	}

	// Set the total API usage per second in the user metrics
	userLevel.Current.TotalAPIUsagePerSec = totalAPIRatePerSec

	// Update the previous usage with the current values
	previousUserAPIUsage[userLevel.Meta.ID] = APIUsage{
		Categories:    userLevel.APIUsagePerUser,
		LastCollected: currentTime,
	}
}

func CalculateCurrentBucketMetrics(bucketMetrics *RadosGWBucketMetrics, currentBytesSent, currentBytesReceived, currentReadOps, currentWriteOps uint64, currentTime time.Time) {
	bucketName := bucketMetrics.Meta.Name
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
		// Initialize current metrics with zeros
		bucketMetrics.Current.OpsPerSec = 0.0
		bucketMetrics.Current.ReadOpsPerSec = 0.0
		bucketMetrics.Current.WriteOpsPerSec = 0.0
		bucketMetrics.Current.BytesSentPerSec = 0.0
		bucketMetrics.Current.BytesReceivedPerSec = 0.0
		bucketMetrics.Current.ThroughputBytesPerSec = 0.0
		return
	}

	// Calculate time difference in seconds
	deltaTime := currentTime.Sub(prevUsage.LastCollected).Seconds()
	if deltaTime <= 0 {
		// If the time delta is zero or negative, skip the calculation
		return
	}

	// Calculate deltas (difference in bytes sent, received, read/write ops)
	readOpsDelta := currentReadOps - prevUsage.ReadOpsTotal
	writeOpsDelta := currentWriteOps - prevUsage.WriteOpsTotal
	bytesSentDelta := currentBytesSent - prevUsage.BytesSent
	bytesReceivedDelta := currentBytesReceived - prevUsage.BytesReceived

	// If the current value is less than the previous one (counter reset), reset the delta
	if currentReadOps < prevUsage.ReadOpsTotal {
		readOpsDelta = 0
	}
	if currentWriteOps < prevUsage.WriteOpsTotal {
		writeOpsDelta = 0
	}
	if currentBytesSent < prevUsage.BytesSent {
		bytesSentDelta = 0
	}
	if currentBytesReceived < prevUsage.BytesReceived {
		bytesReceivedDelta = 0
	}

	// Only proceed with valid deltas
	if bytesSentDelta > 0 || bytesReceivedDelta > 0 {
		// Calculate throughput in bytes per second (sent + received)
		throughput = float64(bytesSentDelta+bytesReceivedDelta) / deltaTime
		currentBytesSentRate = float64(bytesSentDelta) / deltaTime
		currentBytesReceivedRate = float64(bytesReceivedDelta) / deltaTime
	} else {
		throughput = 0.0
		currentBytesSentRate = 0.0
		currentBytesReceivedRate = 0.0
	}

	// Calculate individual read/write ops rates
	if readOpsDelta > 0 {
		currentReadOpsRate = float64(readOpsDelta) / deltaTime
	} else {
		currentReadOpsRate = 0.0
	}

	if writeOpsDelta > 0 {
		currentWriteOpsRate = float64(writeOpsDelta) / deltaTime
	} else {
		currentWriteOpsRate = 0.0
	}

	// Calculate total operations (read + write) per second
	if readOpsDelta > 0 || writeOpsDelta > 0 {
		currentOpsRate = float64(readOpsDelta+writeOpsDelta) / deltaTime
	} else {
		currentOpsRate = 0.0
	}

	// Update the previous usage data with the current values
	previousBucketUsage[bucketName] = PreviousUsage{
		BytesSent:     currentBytesSent,
		BytesReceived: currentBytesReceived,
		LastCollected: currentTime,
		ReadOpsTotal:  currentReadOps,
		WriteOpsTotal: currentWriteOps,
	}

	// Update the current metrics in the bucketMetrics object with float64 values
	bucketMetrics.Current.OpsPerSec = currentOpsRate
	bucketMetrics.Current.ReadOpsPerSec = currentReadOpsRate
	bucketMetrics.Current.WriteOpsPerSec = currentWriteOpsRate
	bucketMetrics.Current.BytesSentPerSec = currentBytesSentRate
	bucketMetrics.Current.BytesReceivedPerSec = currentBytesReceivedRate
	bucketMetrics.Current.ThroughputBytesPerSec = throughput
}

func CalculateCurrentBucketAPIUsage(bucketMetrics *RadosGWBucketMetrics, currentAPIUsage map[string]uint64, currentTime time.Time) {
	bucketName := bucketMetrics.Meta.Name
	prevAPIUsage, exists := previousBucketAPIUsage[bucketName]

	// Ensure the current API usage map is initialized
	if bucketMetrics.Current.APIUsage == nil {
		bucketMetrics.Current.APIUsage = make(map[string]float64)
	}

	totalAPIUsagePerSec := 0.0

	if !exists {
		// First collection, set previous usage and initialize current API usage to 0
		previousBucketAPIUsage[bucketName] = PreviousAPIUsage{
			Usage:         currentAPIUsage,
			LastCollected: currentTime,
		}
		// Set initial API usage rate to 0 for all categories
		for category := range currentAPIUsage {
			bucketMetrics.Current.APIUsage[category] = 0.0
		}
		bucketMetrics.Current.TotalAPIUsagePerSec = 0.0
		return
	}

	// Calculate time difference in seconds
	deltaTime := currentTime.Sub(prevAPIUsage.LastCollected).Seconds()
	if deltaTime <= 0 {
		// If the time delta is zero or negative, skip the calculation
		return
	}

	// Calculate the current rate (API operations per second) for each category
	for category, currentOps := range currentAPIUsage {
		prevOps, found := prevAPIUsage.Usage[category]

		// Handle counter reset or missing previous operations
		var opsDelta uint64
		if found {
			if currentOps >= prevOps {
				opsDelta = currentOps - prevOps
			} else {
				opsDelta = currentOps // Assume a counter reset
			}
		} else {
			// No previous data for this category, assume this is the first collection
			opsDelta = currentOps
		}

		// Calculate the current API usage rate (operations per second)
		currentOpsRate := float64(opsDelta) / deltaTime
		bucketMetrics.Current.APIUsage[category] = currentOpsRate

		// Accumulate the total API usage per second
		totalAPIUsagePerSec += currentOpsRate
	}

	// Update the previous API usage with the current values
	previousBucketAPIUsage[bucketName] = PreviousAPIUsage{
		Usage:         currentAPIUsage,
		LastCollected: currentTime,
	}

	bucketMetrics.Current.TotalAPIUsagePerSec = totalAPIUsagePerSec
}
