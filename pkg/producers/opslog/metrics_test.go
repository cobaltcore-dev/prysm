package opslog

import (
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSubtractMetrics(t *testing.T) {
	total := NewMetrics()
	prev := NewMetrics()

	// Setup total values
	total.TotalRequests.Store(10)
	total.RequestsByUser.Store("user1|bucket1|GET|200", newUint64(5))
	total.RequestsByUser.Store("user2|bucket2|PUT|201", newUint64(3))

	// Setup previous values
	prev.TotalRequests.Store(7)
	prev.RequestsByUser.Store("user1|bucket1|GET|200", newUint64(2))

	// Subtract
	delta := SubtractMetrics(total, prev)

	// Expect user1 delta = 3
	v1, ok := delta.RequestsByUser.Load("user1|bucket1|GET|200")
	assert.True(t, ok, "Expected key user1|bucket1|GET|200 to exist")
	assert.Equal(t, uint64(3), v1.(*atomic.Uint64).Load())

	// Expect user2 delta = 3 (since not in previous)
	v2, ok := delta.RequestsByUser.Load("user2|bucket2|PUT|201")
	assert.True(t, ok, "Expected key user2|bucket2|PUT|201 to exist")
	assert.Equal(t, uint64(3), v2.(*atomic.Uint64).Load())
}

func TestCloneMetrics(t *testing.T) {
	original := NewMetrics()

	// Set some base values
	original.TotalRequests.Store(42)
	original.BytesSent.Store(1024)
	original.RequestsByUser.Store("user1|bucket1|GET|200", newUint64(5))
	original.BytesReceivedByUser.Store("user1", newUint64(888))

	// Clone it
	clone := original.Clone()

	// Top-level fields
	assert.Equal(t, uint64(42), clone.TotalRequests.Load())
	assert.Equal(t, uint64(1024), clone.BytesSent.Load())

	// SyncMap values
	v1, ok := clone.RequestsByUser.Load("user1|bucket1|GET|200")
	assert.True(t, ok, "Expected key to exist in RequestsByUser")
	assert.Equal(t, uint64(5), v1.(*atomic.Uint64).Load())

	v2, ok := clone.BytesReceivedByUser.Load("user1")
	assert.True(t, ok, "Expected key to exist in BytesReceivedByUser")
	assert.Equal(t, uint64(888), v2.(*atomic.Uint64).Load())

	// Mutate original, ensure clone is untouched
	original.TotalRequests.Add(10)
	original.RequestsByUser.Store("user1|bucket1|GET|200", newUint64(99))

	v1After, _ := clone.RequestsByUser.Load("user1|bucket1|GET|200")
	assert.Equal(t, uint64(5), v1After.(*atomic.Uint64).Load(), "Clone should remain unchanged")
}

func TestSubtractMetrics_ZeroDelta(t *testing.T) {
	total := NewMetrics()
	prev := NewMetrics()

	total.RequestsByUser.Store("x", newUint64(5))
	prev.RequestsByUser.Store("x", newUint64(5))

	delta := SubtractMetrics(total, prev)
	_, ok := delta.RequestsByUser.Load("x")
	assert.False(t, ok)
}

func TestSubtractMetrics_MissingInPrev(t *testing.T) {
	total := NewMetrics()
	prev := NewMetrics()

	total.RequestsByUser.Store("new|key", newUint64(7))

	delta := SubtractMetrics(total, prev)

	v, ok := delta.RequestsByUser.Load("new|key")
	assert.True(t, ok)
	assert.Equal(t, uint64(7), v.(*atomic.Uint64).Load())
}

func TestLatencyObsPropagation(t *testing.T) {
	called := false
	cb := func(u, tnt, bucket, method string, sec float64) {
		assert.Equal(t, "u1", u)
		assert.Equal(t, "t1", tnt)
		assert.Equal(t, "b1", bucket)
		assert.Equal(t, "M", method)
		assert.InDelta(t, 0.123, sec, 1e-6)
		called = true
	}

	m := NewMetrics(cb)
	// direct call
	m.LatencyObs("u1", "t1", "b1", "M", 0.123)
	assert.True(t, called)

	// clone carries it forward
	clone := m.Clone()
	called = false
	clone.LatencyObs("u1", "t1", "b1", "M", 0.123)
	assert.True(t, called)

	// subtract carries it forward
	total := NewMetrics(cb)
	prev := NewMetrics(cb)
	delta := SubtractMetrics(total, prev)
	called = false
	delta.LatencyObs("u1", "t1", "b1", "M", 0.123)
	assert.True(t, called)
}

func newUint64(val uint64) *atomic.Uint64 {
	var u atomic.Uint64
	u.Store(val)
	return &u
}
