// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and prysm contributors
//
// SPDX-License-Identifier: Apache-2.0

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

	var throughput, currentBytesSentRate, currentBytesReceivedRate, currentOpsRate, currentReadOpsRate, currentWriteOpsRate float64

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
		userLevel.Current.ReadOpsPerSec = 0.0
		userLevel.Current.WriteOpsPerSec = 0.0
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

		// Calculate read and write operations per second individually
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
		currentOpsRate = currentReadOpsRate + currentWriteOpsRate
	} else {
		// If deltaTime is zero or negative, reset the rates
		currentOpsRate = 0.0
		throughput = 0.0
		currentBytesSentRate = 0.0
		currentBytesReceivedRate = 0.0
		currentReadOpsRate = 0.0
		currentWriteOpsRate = 0.0
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
	userLevel.Current.ReadOpsPerSec = currentReadOpsRate
	userLevel.Current.WriteOpsPerSec = currentWriteOpsRate
}

func CalculateCurrentAPIUsagePerUser(userLevel *RadosGWUserMetrics, currentTime time.Time) {
	// Retrieve the previous usage if it exists
	prevUsage, exists := previousUserAPIUsage[userLevel.Meta.ID]

	// Initialize the current rates map if not already done
	if userLevel.Current.APIUsagePerSec == nil {
		userLevel.Current.APIUsagePerSec = make(map[string]float64)
	}

	// If this is the first collection, initialize and return
	if !exists {
		previousUserAPIUsage[userLevel.Meta.ID] = APIUsage{
			Categories:    copyMap(userLevel.APIUsagePerUser),
			LastCollected: currentTime,
		}
		for category := range userLevel.APIUsagePerUser {
			userLevel.Current.APIUsagePerSec[category] = 0.0
		}
		userLevel.Current.TotalAPIUsagePerSec = 0.0
		return
	}

	// Calculate the time difference in seconds
	deltaTime := currentTime.Sub(prevUsage.LastCollected).Seconds()
	if deltaTime <= 0 {
		return // Exit if deltaTime is zero or negative
	}

	totalAPIRatePerSec := 0.0

	// Calculate deltas per category and total usage rate
	for category, currentOps := range userLevel.APIUsagePerUser {
		// Retrieve previous operations for this category, or set to 0 if not found
		prevOps, found := prevUsage.Categories[category]
		opsDelta := currentOps
		if found && currentOps >= prevOps {
			opsDelta = currentOps - prevOps
		} else if found && currentOps < prevOps {
			// Handle counter reset case by treating it as the initial value
			opsDelta = currentOps
		}

		// Calculate per-second rate for this category
		apiRatePerSec := float64(opsDelta) / deltaTime
		userLevel.Current.APIUsagePerSec[category] = apiRatePerSec
		totalAPIRatePerSec += apiRatePerSec
	}

	// Set the total API usage per second in the current metrics
	userLevel.Current.TotalAPIUsagePerSec = totalAPIRatePerSec

	// Update the previous usage state with the current values for the next run
	previousUserAPIUsage[userLevel.Meta.ID] = APIUsage{
		Categories:    copyMap(userLevel.APIUsagePerUser),
		LastCollected: currentTime,
	}
}

// Helper function to copy a map (to prevent overriding issues)
func copyMap(original map[string]uint64) map[string]uint64 {
	copy := make(map[string]uint64)
	for k, v := range original {
		copy[k] = v
	}
	return copy
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
	currentReadOpsRate = float64(readOpsDelta) / deltaTime
	currentWriteOpsRate = float64(writeOpsDelta) / deltaTime

	// Calculate total operations (read + write) per second
	currentOpsRate = currentReadOpsRate + currentWriteOpsRate

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

	// Initialize the map if not already done
	if bucketMetrics.Current.APIUsage == nil {
		bucketMetrics.Current.APIUsage = make(map[string]float64)
	}

	totalAPIUsagePerSec := 0.0

	if !exists {
		// First-time collection: set current usage as previous and initialize rates to zero
		previousBucketAPIUsage[bucketName] = PreviousAPIUsage{
			Usage:         make(map[string]uint64),
			LastCollected: currentTime,
		}
		for category, ops := range currentAPIUsage {
			previousBucketAPIUsage[bucketName].Usage[category] = ops
			bucketMetrics.Current.APIUsage[category] = 0.0
		}
		bucketMetrics.Current.TotalAPIUsagePerSec = 0.0
		return
	}

	// Calculate the time difference in seconds
	deltaTime := currentTime.Sub(prevAPIUsage.LastCollected).Seconds()
	if deltaTime <= 0 {
		// If the time delta is zero or negative, skip the calculation
		return
	}

	// Calculate per-second API usage rates for each category
	for category, currentOps := range currentAPIUsage {
		prevOps, found := prevAPIUsage.Usage[category]
		var opsDelta uint64

		if found {
			if currentOps >= prevOps {
				opsDelta = currentOps - prevOps
			} else {
				opsDelta = currentOps // Assume a counter reset
			}
		} else {
			// New category, set initial delta as the current ops count
			opsDelta = currentOps
		}

		// Calculate the rate per second for this category
		currentOpsRate := float64(opsDelta) / deltaTime
		bucketMetrics.Current.APIUsage[category] = currentOpsRate

		// Add to total API usage per second
		totalAPIUsagePerSec += currentOpsRate
	}

	// Update the previous usage data with the latest values

	previousBucketAPIUsage[bucketName] = PreviousAPIUsage{
		Usage:         make(map[string]uint64),
		LastCollected: currentTime,
	}
	for category, ops := range currentAPIUsage {
		previousBucketAPIUsage[bucketName].Usage[category] = ops
	}

	// Set the total API usage per second in the bucket metrics
	bucketMetrics.Current.TotalAPIUsagePerSec = totalAPIUsagePerSec
}
