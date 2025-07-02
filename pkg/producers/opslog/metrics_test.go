package opslog

import (
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSubtractMetrics(t *testing.T) {
	total := NewMetrics()
	prev := NewMetrics()

	// Setup total values across different metric types
	total.TotalRequests.Store(10)
	total.BytesSent.Store(2048)

	// Test various storage maps
	total.RequestsDetailed.Store("user1|bucket1|GET|200", newUint64(5))
	total.RequestsByMethodDetailed.Store("user1|bucket1|GET", newUint64(3))
	total.BytesSentDetailed.Store("user1|bucket1", newUint64(1024))
	total.ErrorsDetailed.Store("user1|bucket1|404", newUint64(2))

	// Setup previous values
	prev.TotalRequests.Store(7)
	prev.BytesSent.Store(1024)
	prev.RequestsDetailed.Store("user1|bucket1|GET|200", newUint64(2))
	prev.BytesSentDetailed.Store("user1|bucket1", newUint64(512))

	// Subtract
	delta := SubtractMetrics(total, prev)

	// Test atomic counters
	assert.Equal(t, uint64(3), delta.TotalRequests.Load())
	assert.Equal(t, uint64(1024), delta.BytesSent.Load())

	// Test detailed requests delta
	v1, ok := delta.RequestsDetailed.Load("user1|bucket1|GET|200")
	assert.True(t, ok, "Expected key user1|bucket1|GET|200 to exist in RequestsDetailed")
	assert.Equal(t, uint64(3), v1.(*atomic.Uint64).Load())

	// Test method details (new key, should equal total)
	v2, ok := delta.RequestsByMethodDetailed.Load("user1|bucket1|GET")
	assert.True(t, ok, "Expected key user1|bucket1|GET to exist in RequestsByMethodDetailed")
	assert.Equal(t, uint64(3), v2.(*atomic.Uint64).Load())

	// Test bytes delta
	v3, ok := delta.BytesSentDetailed.Load("user1|bucket1")
	assert.True(t, ok, "Expected key user1|bucket1 to exist in BytesSentDetailed")
	assert.Equal(t, uint64(512), v3.(*atomic.Uint64).Load())

	// Test errors (new key, should equal total)
	v4, ok := delta.ErrorsDetailed.Load("user1|bucket1|404")
	assert.True(t, ok, "Expected key user1|bucket1|404 to exist in ErrorsDetailed")
	assert.Equal(t, uint64(2), v4.(*atomic.Uint64).Load())
}

func TestCloneMetrics(t *testing.T) {
	original := NewMetrics()

	// Set some base values across different metric types
	original.TotalRequests.Store(42)
	original.BytesSent.Store(1024)
	original.Errors.Store(5)

	// Test different storage maps
	original.RequestsDetailed.Store("user1|bucket1|GET|200", newUint64(5))
	original.RequestsByMethodDetailed.Store("user1|bucket1|GET", newUint64(3))
	original.BytesSentDetailed.Store("user1|bucket1", newUint64(1024))
	original.BytesSentPerUser.Store("user1", newUint64(888))
	original.ErrorsDetailed.Store("user1|bucket1|404", newUint64(2))
	original.RequestsByIPDetailed.Store("user1|192.168.1.1", newUint64(7))

	// Clone it
	clone := original.Clone()

	// Test top-level atomic fields
	assert.Equal(t, uint64(42), clone.TotalRequests.Load())
	assert.Equal(t, uint64(1024), clone.BytesSent.Load())
	assert.Equal(t, uint64(5), clone.Errors.Load())

	// Test sync.Map values across different types
	v1, ok := clone.RequestsDetailed.Load("user1|bucket1|GET|200")
	assert.True(t, ok, "Expected key to exist in RequestsDetailed")
	assert.Equal(t, uint64(5), v1.(*atomic.Uint64).Load())

	v2, ok := clone.RequestsByMethodDetailed.Load("user1|bucket1|GET")
	assert.True(t, ok, "Expected key to exist in RequestsByMethodDetailed")
	assert.Equal(t, uint64(3), v2.(*atomic.Uint64).Load())

	v3, ok := clone.BytesSentDetailed.Load("user1|bucket1")
	assert.True(t, ok, "Expected key to exist in BytesSentDetailed")
	assert.Equal(t, uint64(1024), v3.(*atomic.Uint64).Load())

	v4, ok := clone.BytesSentPerUser.Load("user1")
	assert.True(t, ok, "Expected key to exist in BytesSentPerUser")
	assert.Equal(t, uint64(888), v4.(*atomic.Uint64).Load())

	v5, ok := clone.ErrorsDetailed.Load("user1|bucket1|404")
	assert.True(t, ok, "Expected key to exist in ErrorsDetailed")
	assert.Equal(t, uint64(2), v5.(*atomic.Uint64).Load())

	v6, ok := clone.RequestsByIPDetailed.Load("user1|192.168.1.1")
	assert.True(t, ok, "Expected key to exist in RequestsByIPDetailed")
	assert.Equal(t, uint64(7), v6.(*atomic.Uint64).Load())

	// Mutate original, ensure clone is untouched
	original.TotalRequests.Add(10)
	original.RequestsDetailed.Store("user1|bucket1|GET|200", newUint64(99))

	// Verify clone remains unchanged
	assert.Equal(t, uint64(42), clone.TotalRequests.Load(), "Clone TotalRequests should remain unchanged")

	v1After, _ := clone.RequestsDetailed.Load("user1|bucket1|GET|200")
	assert.Equal(t, uint64(5), v1After.(*atomic.Uint64).Load(), "Clone RequestsDetailed should remain unchanged")
}

func TestSubtractMetrics_ZeroDelta(t *testing.T) {
	total := NewMetrics()
	prev := NewMetrics()

	// Test zero delta across different metric types
	total.RequestsDetailed.Store("user1|bucket1|GET|200", newUint64(5))
	prev.RequestsDetailed.Store("user1|bucket1|GET|200", newUint64(5))

	total.BytesSentPerUser.Store("user1", newUint64(1024))
	prev.BytesSentPerUser.Store("user1", newUint64(1024))

	delta := SubtractMetrics(total, prev)

	// Zero deltas should not be stored
	_, ok1 := delta.RequestsDetailed.Load("user1|bucket1|GET|200")
	assert.False(t, ok1, "Zero delta should not be stored in RequestsDetailed")

	_, ok2 := delta.BytesSentPerUser.Load("user1")
	assert.False(t, ok2, "Zero delta should not be stored in BytesSentPerUser")
}

func TestSubtractMetrics_MissingInPrev(t *testing.T) {
	total := NewMetrics()
	prev := NewMetrics()

	// Test new keys that don't exist in previous
	total.RequestsDetailed.Store("new|key|GET|200", newUint64(7))
	total.ErrorsPerUser.Store("newuser|404", newUint64(3))

	delta := SubtractMetrics(total, prev)

	// New keys should appear with full value
	v1, ok1 := delta.RequestsDetailed.Load("new|key|GET|200")
	assert.True(t, ok1, "New key should exist in delta")
	assert.Equal(t, uint64(7), v1.(*atomic.Uint64).Load())

	v2, ok2 := delta.ErrorsPerUser.Load("newuser|404")
	assert.True(t, ok2, "New error key should exist in delta")
	assert.Equal(t, uint64(3), v2.(*atomic.Uint64).Load())
}

func TestLatencyObsPropagation(t *testing.T) {
	called := false
	callCount := 0
	var capturedArgs []string

	cb := func(u, tnt, bucket, method string, sec float64) {
		called = true
		callCount++
		capturedArgs = []string{u, tnt, bucket, method}
		assert.Equal(t, "u1", u)
		assert.Equal(t, "t1", tnt)
		assert.Equal(t, "b1", bucket)
		assert.Equal(t, "M", method)
		assert.InDelta(t, 0.123, sec, 1e-6)
	}

	// Test direct call
	m := NewMetrics(cb)
	m.LatencyObs("u1", "t1", "b1", "M", 0.123)
	assert.True(t, called, "LatencyObs should be called")
	assert.Equal(t, 1, callCount)
	assert.Equal(t, []string{"u1", "t1", "b1", "M"}, capturedArgs)

	// Test clone carries it forward
	clone := m.Clone()
	called = false
	callCount = 0
	clone.LatencyObs("u1", "t1", "b1", "M", 0.123)
	assert.True(t, called, "Cloned LatencyObs should be called")
	assert.Equal(t, 1, callCount)

	// Test subtract carries it forward
	total := NewMetrics(cb)
	prev := NewMetrics(cb)
	delta := SubtractMetrics(total, prev)
	called = false
	callCount = 0
	delta.LatencyObs("u1", "t1", "b1", "M", 0.123)
	assert.True(t, called, "Delta LatencyObs should be called")
	assert.Equal(t, 1, callCount)
}

func TestLatencyObsDefaultNoOp(t *testing.T) {
	// Test that NewMetrics without callback creates no-op function
	m := NewMetrics()

	// Should not panic
	assert.NotPanics(t, func() {
		m.LatencyObs("user", "tenant", "bucket", "method", 1.23)
	}, "Default LatencyObs should be no-op and not panic")
}

func TestMetricsUpdate_BasicFunctionality(t *testing.T) {
	config := &MetricsConfig{
		TrackRequestsDetailed:  true,
		TrackLatencyDetailed:   true,
		TrackBytesSentDetailed: true,
		TrackErrorsDetailed:    true,
	}

	latencyCallCount := 0
	latencyObs := func(user, tenant, bucket, method string, seconds float64) {
		latencyCallCount++
		assert.Equal(t, "user1", user)
		assert.Equal(t, "tenant1", tenant)
		assert.Equal(t, "bucket1", bucket)
		assert.Equal(t, "GET", method)
		assert.InDelta(t, 0.150, seconds, 1e-6) // 150ms converted to seconds
	}

	m := NewMetrics(latencyObs)

	logEntry := S3OperationLog{
		User:          "user1$tenant1",
		Bucket:        "bucket1",
		URI:           "GET /bucket1/object.txt HTTP/1.1",
		HTTPStatus:    "200",
		BytesSent:     1024,
		BytesReceived: 0,
		TotalTime:     150, // milliseconds
	}

	// Update metrics
	m.Update(logEntry, config)

	// Verify atomic counters
	assert.Equal(t, uint64(1), m.TotalRequests.Load())
	assert.Equal(t, uint64(1024), m.BytesSent.Load())
	assert.Equal(t, uint64(0), m.BytesReceived.Load())
	assert.Equal(t, uint64(0), m.Errors.Load()) // 200 is not an error

	// Verify detailed requests tracking
	v, ok := m.RequestsDetailed.Load("user1$tenant1|bucket1|GET|200")
	assert.True(t, ok, "Should track detailed request")
	assert.Equal(t, uint64(1), v.(*atomic.Uint64).Load())

	// Verify bytes tracking
	v2, ok2 := m.BytesSentDetailed.Load("user1$tenant1|bucket1")
	assert.True(t, ok2, "Should track detailed bytes sent")
	assert.Equal(t, uint64(1024), v2.(*atomic.Uint64).Load())

	// Verify latency observation was called
	assert.Equal(t, 1, latencyCallCount, "LatencyObs should be called once")
}

func TestMetricsUpdate_ErrorTracking(t *testing.T) {
	config := &MetricsConfig{
		TrackErrorsDetailed: true,
		TrackErrorsPerUser:  true,
	}

	m := NewMetrics()

	logEntry := S3OperationLog{
		User:       "user1$tenant1",
		Bucket:     "bucket1",
		URI:        "GET /bucket1/missing.txt HTTP/1.1",
		HTTPStatus: "404",
	}

	m.Update(logEntry, config)

	// Verify error counters
	assert.Equal(t, uint64(1), m.Errors.Load())

	// Verify detailed error tracking
	v1, ok1 := m.ErrorsDetailed.Load("user1$tenant1|bucket1|404")
	assert.True(t, ok1, "Should track detailed error")
	assert.Equal(t, uint64(1), v1.(*atomic.Uint64).Load())

	// Verify per-user error tracking
	v2, ok2 := m.ErrorsPerUser.Load("user1|404")
	assert.True(t, ok2, "Should track per-user error")
	assert.Equal(t, uint64(1), v2.(*atomic.Uint64).Load())
}

func TestMetricsUpdate_ConditionalTracking(t *testing.T) {
	// Test that disabled tracking doesn't create entries
	config := &MetricsConfig{
		TrackRequestsDetailed: false,
		TrackErrorsDetailed:   false,
		TrackLatencyDetailed:  false,
	}

	m := NewMetrics()

	logEntry := S3OperationLog{
		User:       "user1$tenant1",
		Bucket:     "bucket1",
		URI:        "GET /bucket1/object.txt HTTP/1.1",
		HTTPStatus: "404",
		TotalTime:  150,
	}

	m.Update(logEntry, config)

	// Basic counters should still work
	assert.Equal(t, uint64(1), m.TotalRequests.Load())
	assert.Equal(t, uint64(1), m.Errors.Load())

	// But detailed tracking should be empty
	_, ok1 := m.RequestsDetailed.Load("user1$tenant1|bucket1|GET|404")
	assert.False(t, ok1, "Should not track detailed requests when disabled")

	_, ok2 := m.ErrorsDetailed.Load("user1$tenant1|bucket1|404")
	assert.False(t, ok2, "Should not track detailed errors when disabled")
}

func newUint64(val uint64) *atomic.Uint64 {
	var u atomic.Uint64
	u.Store(val)
	return &u
}
