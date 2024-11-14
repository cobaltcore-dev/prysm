package radosgwusage

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCalculateCurrentUserMetrics(t *testing.T) {
	// Initialize test data
	userID := "user123"
	userMetrics := &RadosGWUserMetrics{
		Meta: RadosGWUserMetricsMeta{ID: userID},
		Totals: RadosGWUserMetricsTotals{
			BytesSentTotal:     1000,
			BytesReceivedTotal: 2000,
			ReadOpsTotal:       300,
			WriteOpsTotal:      400,
		},
		Current: RadosGWUserMetricsCurrent{},
	}

	// First call to establish the baseline
	initialTime := time.Now()
	CalculateCurrentUserMetrics(userMetrics, initialTime)

	// Assert initial values are set to zero for the first call
	assert.Equal(t, 0.0, userMetrics.Current.OpsPerSec)
	assert.Equal(t, 0.0, userMetrics.Current.ThroughputBytesPerSec)
	assert.Equal(t, 0.0, userMetrics.Current.DataBytesSentPerSec)
	assert.Equal(t, 0.0, userMetrics.Current.DataBytesReceivedPerSec)
	assert.Equal(t, 0.0, userMetrics.Current.ReadOpsPerSec)
	assert.Equal(t, 0.0, userMetrics.Current.WriteOpsPerSec)

	// Simulate updated metrics after 1 second
	time.Sleep(time.Second)
	userMetrics.Totals.BytesSentTotal = 3000
	userMetrics.Totals.BytesReceivedTotal = 5000
	userMetrics.Totals.ReadOpsTotal = 600
	userMetrics.Totals.WriteOpsTotal = 800

	// Second call to calculate deltas
	currentTime := initialTime.Add(time.Second)
	CalculateCurrentUserMetrics(userMetrics, currentTime)

	// Calculate expected values
	expectedBytesSentRate := 2000.0     // (3000 - 1000) / 1 second
	expectedBytesReceivedRate := 3000.0 // (5000 - 2000) / 1 second
	expectedThroughput := expectedBytesSentRate + expectedBytesReceivedRate
	expectedReadOpsRate := 300.0  // (600 - 300) / 1 second
	expectedWriteOpsRate := 400.0 // (800 - 400) / 1 second
	expectedOpsRate := expectedReadOpsRate + expectedWriteOpsRate

	// Assert calculated values match expected
	assert.Equal(t, expectedOpsRate, userMetrics.Current.OpsPerSec)
	assert.Equal(t, expectedThroughput, userMetrics.Current.ThroughputBytesPerSec)
	assert.Equal(t, expectedBytesSentRate, userMetrics.Current.DataBytesSentPerSec)
	assert.Equal(t, expectedBytesReceivedRate, userMetrics.Current.DataBytesReceivedPerSec)
	assert.Equal(t, expectedReadOpsRate, userMetrics.Current.ReadOpsPerSec)
	assert.Equal(t, expectedWriteOpsRate, userMetrics.Current.WriteOpsPerSec)
}

func TestCalculateCurrentAPIUsagePerUser(t *testing.T) {
	userLevel := RadosGWUserMetrics{
		Meta: struct {
			ID                  string
			DisplayName         string
			Email               string
			DefaultStorageClass string
		}{ID: "user123"},
		APIUsagePerUser: map[string]uint64{
			"delete_obj": 90,
			"get_obj":    300,
			"put_obj":    150,
		},
		Current: struct {
			OpsPerSec               float64
			ReadOpsPerSec           float64
			WriteOpsPerSec          float64
			DataBytesReceivedPerSec float64
			DataBytesSentPerSec     float64
			ThroughputBytesPerSec   float64
			APIUsagePerSec          map[string]float64
			TotalAPIUsagePerSec     float64
		}{APIUsagePerSec: make(map[string]float64)},
	}

	initialTime := time.Now()
	CalculateCurrentAPIUsagePerUser(&userLevel, initialTime)
	assert.Equal(t, 0.0, userLevel.Current.TotalAPIUsagePerSec)

	// Advance time and increment counts for the next collection
	secondTime := initialTime.Add(2 * time.Second)
	userLevel.APIUsagePerUser = map[string]uint64{
		"delete_obj": 180, // +90 in 2s -> 45 ops/s
		"get_obj":    400, // +100 in 2s -> 50 ops/s
		"put_obj":    200, // +50 in 2s -> 25 ops/s
	}
	CalculateCurrentAPIUsagePerUser(&userLevel, secondTime)
	assert.Equal(t, 120.0, userLevel.Current.TotalAPIUsagePerSec)
	assert.Equal(t, 45.0, userLevel.Current.APIUsagePerSec["delete_obj"])
	assert.Equal(t, 50.0, userLevel.Current.APIUsagePerSec["get_obj"])
	assert.Equal(t, 25.0, userLevel.Current.APIUsagePerSec["put_obj"])

	// Test counter reset by reducing counts
	thirdTime := secondTime.Add(2 * time.Second)
	userLevel.APIUsagePerUser = map[string]uint64{
		"delete_obj": 180, // Reset case, expect 0
		"get_obj":    500, // +100 in 2s -> 50 ops/s
		"put_obj":    250, // +50 in 2s -> 25 ops/s
	}
	CalculateCurrentAPIUsagePerUser(&userLevel, thirdTime)
	assert.Equal(t, 75.0, userLevel.Current.TotalAPIUsagePerSec)
	assert.Equal(t, 0.0, userLevel.Current.APIUsagePerSec["delete_obj"])
	assert.Equal(t, 50.0, userLevel.Current.APIUsagePerSec["get_obj"])
	assert.Equal(t, 25.0, userLevel.Current.APIUsagePerSec["put_obj"])
}

func TestCalculateCurrentBucketMetrics(t *testing.T) {
	// Setup initial bucket metrics and current time
	bucketMetrics := &RadosGWBucketMetrics{
		Meta: RadosGWBucketMetricsMeta{
			Name: "test-bucket",
		},
		Current: RadosGWBucketMetricsCurrent{
			OpsPerSec:             0.0,
			ReadOpsPerSec:         0.0,
			WriteOpsPerSec:        0.0,
			BytesSentPerSec:       0.0,
			BytesReceivedPerSec:   0.0,
			ThroughputBytesPerSec: 0.0,
		},
	}

	initialTime := time.Now()

	// Initial data point
	initialBytesSent := uint64(1000)
	initialBytesReceived := uint64(500)
	initialReadOps := uint64(10)
	initialWriteOps := uint64(5)

	// First collection - should initialize and set rates to 0
	CalculateCurrentBucketMetrics(bucketMetrics, initialBytesSent, initialBytesReceived, initialReadOps, initialWriteOps, initialTime)
	assert.Equal(t, 0.0, bucketMetrics.Current.OpsPerSec)
	assert.Equal(t, 0.0, bucketMetrics.Current.ReadOpsPerSec)
	assert.Equal(t, 0.0, bucketMetrics.Current.WriteOpsPerSec)
	assert.Equal(t, 0.0, bucketMetrics.Current.BytesSentPerSec)
	assert.Equal(t, 0.0, bucketMetrics.Current.BytesReceivedPerSec)
	assert.Equal(t, 0.0, bucketMetrics.Current.ThroughputBytesPerSec)

	// Second data point after 2 seconds with increased values
	secondTime := initialTime.Add(2 * time.Second)
	updatedBytesSent := uint64(1300)
	updatedBytesReceived := uint64(700)
	updatedReadOps := uint64(30)
	updatedWriteOps := uint64(10)

	CalculateCurrentBucketMetrics(bucketMetrics, updatedBytesSent, updatedBytesReceived, updatedReadOps, updatedWriteOps, secondTime)
	assert.Equal(t, 12.5, bucketMetrics.Current.OpsPerSec)              // (20 + 5) / 2s
	assert.Equal(t, 10.0, bucketMetrics.Current.ReadOpsPerSec)          // 20 / 2s
	assert.Equal(t, 2.5, bucketMetrics.Current.WriteOpsPerSec)          // 5 / 2s
	assert.Equal(t, 150.0, bucketMetrics.Current.BytesSentPerSec)       // 300 / 2s
	assert.Equal(t, 100.0, bucketMetrics.Current.BytesReceivedPerSec)   // 200 / 2s
	assert.Equal(t, 250.0, bucketMetrics.Current.ThroughputBytesPerSec) // (300 + 200) / 2s

	// Third data point after another 2 seconds with the same values (constant data)
	thirdTime := secondTime.Add(2 * time.Second)
	CalculateCurrentBucketMetrics(bucketMetrics, updatedBytesSent, updatedBytesReceived, updatedReadOps, updatedWriteOps, thirdTime)
	assert.Equal(t, 0.0, bucketMetrics.Current.OpsPerSec)
	assert.Equal(t, 0.0, bucketMetrics.Current.ReadOpsPerSec)
	assert.Equal(t, 0.0, bucketMetrics.Current.WriteOpsPerSec)
	assert.Equal(t, 0.0, bucketMetrics.Current.BytesSentPerSec)
	assert.Equal(t, 0.0, bucketMetrics.Current.BytesReceivedPerSec)
	assert.Equal(t, 0.0, bucketMetrics.Current.ThroughputBytesPerSec)

	// Fourth data point after another 2 seconds with further increased values
	fourthTime := thirdTime.Add(2 * time.Second)
	finalBytesSent := uint64(1600)
	finalBytesReceived := uint64(900)
	finalReadOps := uint64(50)
	finalWriteOps := uint64(15)

	CalculateCurrentBucketMetrics(bucketMetrics, finalBytesSent, finalBytesReceived, finalReadOps, finalWriteOps, fourthTime)
	assert.Equal(t, 12.5, bucketMetrics.Current.OpsPerSec)              // (20 + 5) / 2s
	assert.Equal(t, 10.0, bucketMetrics.Current.ReadOpsPerSec)          // 20 / 2s
	assert.Equal(t, 2.5, bucketMetrics.Current.WriteOpsPerSec)          // 5 / 2s
	assert.Equal(t, 150.0, bucketMetrics.Current.BytesSentPerSec)       // 300 / 2s
	assert.Equal(t, 100.0, bucketMetrics.Current.BytesReceivedPerSec)   // 200 / 2s
	assert.Equal(t, 250.0, bucketMetrics.Current.ThroughputBytesPerSec) // (300 + 200) / 2s
}

func TestCalculateCurrentBucketAPIUsage(t *testing.T) {
	// Define initial bucket metrics with current API usage values
	bucketMetrics := RadosGWBucketMetrics{
		Meta: RadosGWBucketMetricsMeta{
			Name: "test_bucket",
		},
		Current: RadosGWBucketMetricsCurrent{
			APIUsage: make(map[string]float64),
		},
	}

	// Initialize test timestamps
	startTime := time.Now()
	incrementTime := startTime.Add(2 * time.Second)

	// Set initial API usage values for the first call
	initialAPIUsage := map[string]uint64{
		"delete_obj": 100,
		"get_obj":    300,
		"put_obj":    150,
	}

	// First call to establish initial API usage
	CalculateCurrentBucketAPIUsage(&bucketMetrics, initialAPIUsage, startTime)

	// Verify that rates are zero for the initial collection
	assert.Equal(t, 0.0, bucketMetrics.Current.APIUsage["delete_obj"])
	assert.Equal(t, 0.0, bucketMetrics.Current.APIUsage["get_obj"])
	assert.Equal(t, 0.0, bucketMetrics.Current.APIUsage["put_obj"])
	assert.Equal(t, 0.0, bucketMetrics.Current.TotalAPIUsagePerSec)

	// Increment API usage values for the second call
	updatedAPIUsage := map[string]uint64{
		"delete_obj": 180, // +80 ops over 2s -> 40 ops/s
		"get_obj":    400, // +100 ops over 2s -> 50 ops/s
		"put_obj":    200, // +50 ops over 2s -> 25 ops/s
	}

	// Second call after incrementing usage
	CalculateCurrentBucketAPIUsage(&bucketMetrics, updatedAPIUsage, incrementTime)

	// Verify the calculated API usage rates per second
	assert.Equal(t, 40.0, bucketMetrics.Current.APIUsage["delete_obj"])
	assert.Equal(t, 50.0, bucketMetrics.Current.APIUsage["get_obj"])
	assert.Equal(t, 25.0, bucketMetrics.Current.APIUsage["put_obj"])
	assert.Equal(t, 115.0, bucketMetrics.Current.TotalAPIUsagePerSec)

	// Further increment API usage for a third check
	anotherIncrementTime := incrementTime.Add(2 * time.Second)
	finalAPIUsage := map[string]uint64{
		"delete_obj": 260, // +80 ops over 2s -> 40 ops/s
		"get_obj":    450, // +50 ops over 2s -> 25 ops/s
		"put_obj":    250, // +50 ops over 2s -> 25 ops/s
	}

	// Third call after further increments
	CalculateCurrentBucketAPIUsage(&bucketMetrics, finalAPIUsage, anotherIncrementTime)

	// Verify that calculated API usage rates per second reflect the changes
	assert.Equal(t, 40.0, bucketMetrics.Current.APIUsage["delete_obj"])
	assert.Equal(t, 25.0, bucketMetrics.Current.APIUsage["get_obj"])
	assert.Equal(t, 25.0, bucketMetrics.Current.APIUsage["put_obj"])
	assert.Equal(t, 90.0, bucketMetrics.Current.TotalAPIUsagePerSec)
}
