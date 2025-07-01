// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and prysm contributors
//
// SPDX-License-Identifier: Apache-2.0

package opslog

import (
	"encoding/json"
	"strings"
	"sync"
	"sync/atomic"
)

type Metrics struct {
	TotalRequests atomic.Uint64
	BytesSent     atomic.Uint64
	BytesReceived atomic.Uint64
	Errors        atomic.Uint64

	// LatencyByMethod sync.Map // map["user|bucket|method"]

	// LatencyObs records a single request‐latency observation into the
	// `requestsDurationHistogram`, which is registered once at startup.
	// Because the histogram lives for the entire process life and is never
	// re‐initialized or cleared, its buckets continuously accumulate across
	// scrape intervals—ensuring a true cumulative histogram that Prometheus
	// can derive accurate rates and quantiles from.
	LatencyObs func(user string, tenant string, bucket string, method string, seconds float64)

	RequestsByMethod      sync.Map // map[string]*atomic.Uint64
	RequestsByOperation   sync.Map // map[string]*atomic.Uint64
	RequestsPerStatusCode sync.Map // map[string]*atomic.Uint64
	RequestsByUser        sync.Map // map[string]*atomic.Uint64
	BytesSentPerUser      sync.Map // map[string]*atomic.Uint64
	BytesReceivedByUser   sync.Map // map[string]*atomic.Uint64
	ErrorsByUser          sync.Map // map[string]*atomic.Uint64

	RequestsByBucket               sync.Map // map[string]*atomic.Uint64
	BytesSentPerBucket             sync.Map // map[string]*atomic.Uint64
	BytesReceivedPerBucket         sync.Map // map[string]*atomic.Uint64
	RequestsByIP                   sync.Map // map[string]*atomic.Uint64
	RequestsByIPBucketMethodTenant sync.Map // map["ip|bucket|method|tenant"]*atomic.Uint64
	BytesSentByIP                  sync.Map // map[string]*atomic.Uint64
	BytesReceivedByIP              sync.Map // map[string]*atomic.Uint64
	ErrorsByUserAndBucket          sync.Map // map["user|bucket|http_status"]*atomic.Uint64
	ErrorsByIPAndBucket            sync.Map // map["ip|bucket|http_status"]*atomic.Uint64

}

func NewMetrics(obs ...func(user string, tenant string, bucket string, method string, seconds float64)) *Metrics {
	var cb func(user, tenant, bucket, method string, seconds float64)
	if len(obs) > 0 {
		cb = obs[0]
	} else {
		// default no‐op so nobody ever has a nil-pointer
		cb = func(_, _, _, _ string, _ float64) {}
	}
	return &Metrics{
		LatencyObs: cb,
	}
}

// Convert metrics to a JSON-friendly struct
func (m *Metrics) ToJSON(metricsConfig *MetricsConfig) ([]byte, error) {
	data := map[string]any{
		"total_requests": m.TotalRequests.Load(),
		"bytes_sent":     m.BytesSent.Load(),
		"bytes_received": m.BytesReceived.Load(),
		"errors":         m.Errors.Load(),
	}

	if metricsConfig.TrackRequestsByMethodDetailed {
		data["requests_by_method"] = loadSyncMap(&m.RequestsByMethod)
	}

	if metricsConfig.TrackRequestsByOperationDetailed {
		data["requests_by_operation"] = loadSyncMap(&m.RequestsByOperation)
	}

	if metricsConfig.TrackRequestsByStatusDetailed {
		data["requests_per_status"] = loadSyncMap(&m.RequestsPerStatusCode)
	}

	if metricsConfig.TrackRequestsPerBucket {
		data["requests_by_bucket"] = loadSyncMap(&m.RequestsByBucket)
	}

	if metricsConfig.TrackRequestsPerUser {
		data["requests_by_user"] = loadSyncMap(&m.RequestsByUser)
	}

	if metricsConfig.TrackRequestsByIPDetailed {
		data["requests_by_ip"] = loadSyncMap(&m.RequestsByIP)
	}

	if metricsConfig.TrackRequestsByIPBucketMethodTenant {
		data["requests_by_ip_bucket_method_tenant"] = loadSyncMap(&m.RequestsByIPBucketMethodTenant)
	}

	// Conditional fields (bytes tracking)
	if metricsConfig.TrackBytesSentPerUser {
		data["bytes_sent_per_user"] = loadSyncMap(&m.BytesSentPerUser)
	}

	if metricsConfig.TrackBytesReceivedPerUser {
		data["bytes_received_by_user"] = loadSyncMap(&m.BytesReceivedByUser)
	}

	if metricsConfig.TrackBytesSentPerBucket {
		data["bytes_sent_per_bucket"] = loadSyncMap(&m.BytesSentPerBucket)
	}

	if metricsConfig.TrackBytesReceivedPerBucket {
		data["bytes_received_per_bucket"] = loadSyncMap(&m.BytesReceivedPerBucket)
	}

	if metricsConfig.TrackBytesSentByIPDetailed {
		data["bytes_sent_by_ip"] = loadSyncMap(&m.BytesSentByIP)
	}

	if metricsConfig.TrackBytesReceivedByIPDetailed {
		data["bytes_received_by_ip"] = loadSyncMap(&m.BytesReceivedByIP)
	}

	// Conditional fields (errors tracking)
	if metricsConfig.TrackErrorsPerUser {
		data["errors_by_user"] = loadSyncMap(&m.ErrorsByUser)
		data["errors_by_user_and_bucket"] = loadSyncMap(&m.ErrorsByUserAndBucket)
	}

	if metricsConfig.TrackErrorsByIP {
		data["errors_by_ip_and_bucket"] = loadSyncMap(&m.ErrorsByIPAndBucket)
	}

	return json.Marshal(data)
}

// Update increments metrics based on a new log entry
func (m *Metrics) Update(logEntry S3OperationLog, metricsConfig *MetricsConfig) {
	m.TotalRequests.Add(1)
	m.BytesSent.Add(uint64(logEntry.BytesSent))
	m.BytesReceived.Add(uint64(logEntry.BytesReceived))

	// Extract HTTP method from logEntry.URI
	method := ExtractHTTPMethod(logEntry.URI)

	if metricsConfig.TrackRequestsByMethodDetailed {
		// Key format: "user|bucket|method"
		keyMethod := logEntry.User + "|" + logEntry.Bucket + "|" + method
		incrementSyncMap(&m.RequestsByMethod, keyMethod)
	}

	if metricsConfig.TrackRequestsByOperationDetailed {
		// Key format: "user|bucket|operation|method"
		keyOperation := logEntry.User + "|" + logEntry.Bucket + "|" + logEntry.Operation + "|" + method
		incrementSyncMap(&m.RequestsByOperation, keyOperation)
	}

	if metricsConfig.TrackRequestsByStatusDetailed {
		// Increment status code count
		incrementSyncMap(&m.RequestsPerStatusCode, logEntry.HTTPStatus)
	}

	if metricsConfig.TrackRequestsPerBucket {
		// Track per bucket (User | Bucket | Method | HTTP Status)
		keyBucket := logEntry.User + "|" + logEntry.Bucket + "|" + method + "|" + logEntry.HTTPStatus
		incrementSyncMap(&m.RequestsByBucket, keyBucket)
	}

	if metricsConfig.TrackRequestsPerUser {
		// Track per user (User | Bucket | Method | HTTP Status)
		keyUser := logEntry.User + "|" + logEntry.Bucket + "|" + method + "|" + logEntry.HTTPStatus
		incrementSyncMap(&m.RequestsByUser, keyUser)
	}

	if metricsConfig.TrackRequestsByIPDetailed {
		// Track by IP address
		keyUserIP := logEntry.User + "|" + logEntry.RemoteAddr
		incrementSyncMap(&m.RequestsByIP, keyUserIP)
	}

	if metricsConfig.TrackRequestsByIPBucketMethodTenant {
		// Track by IP address, Bucker, method and tenant
		key := logEntry.RemoteAddr + "|" + logEntry.Bucket + "|" + method + "|" + logEntry.User
		incrementSyncMap(&m.RequestsByIPBucketMethodTenant, key)
	}

	// //////////////

	// Bytes Tracking
	if metricsConfig.TrackBytesSentPerUser {
		incrementSyncMapValue(&m.BytesSentPerUser, logEntry.User, uint64(logEntry.BytesSent))
	}

	if metricsConfig.TrackBytesReceivedPerUser {
		incrementSyncMapValue(&m.BytesReceivedByUser, logEntry.User, uint64(logEntry.BytesReceived))
	}

	if metricsConfig.TrackBytesSentPerBucket {
		// Include user for tenant separation: "user|bucket"
		keyBucket := logEntry.User + "|" + logEntry.Bucket
		incrementSyncMapValue(&m.BytesSentPerBucket, keyBucket, uint64(logEntry.BytesSent))
	}

	if metricsConfig.TrackBytesReceivedPerBucket {
		// Include user for tenant separation: "user|bucket"
		keyBucket := logEntry.User + "|" + logEntry.Bucket
		incrementSyncMapValue(&m.BytesReceivedPerBucket, keyBucket, uint64(logEntry.BytesReceived))
	}

	if metricsConfig.TrackBytesSentByIPDetailed {
		keyUserIP := logEntry.User + "|" + logEntry.RemoteAddr
		incrementSyncMapValue(&m.BytesSentByIP, keyUserIP, uint64(logEntry.BytesSent))
	}

	if metricsConfig.TrackBytesReceivedByIPDetailed {
		keyUserIP := logEntry.User + "|" + logEntry.RemoteAddr
		incrementSyncMapValue(&m.BytesReceivedByIP, keyUserIP, uint64(logEntry.BytesReceived))
	}
	// //////////////

	// Error Tracking
	if logEntry.HTTPStatus[0] != '2' { // Non-2xx codes are errors
		if metricsConfig.TrackErrorsPerUser {
			incrementSyncMap(&m.ErrorsByUser, logEntry.User)
		}
		if metricsConfig.TrackErrorsPerUser {
			userBucketKey := logEntry.User + "|" + logEntry.Bucket + "|" + logEntry.HTTPStatus
			incrementSyncMap(&m.ErrorsByUserAndBucket, userBucketKey)
		}
		if metricsConfig.TrackErrorsByIP {
			ipBucketKey := logEntry.RemoteAddr + "|" + logEntry.User + "|" + logEntry.Bucket + "|" + logEntry.HTTPStatus
			incrementSyncMap(&m.ErrorsByIPAndBucket, ipBucketKey)
		}
		m.Errors.Add(1) // Always track total errors
	}

	// //////////////

	// Latency Tracking
	if logEntry.TotalTime > 0 {
		if metricsConfig.TrackLatencyDetailed ||
			metricsConfig.TrackLatencyPerMethod ||
			metricsConfig.TrackLatencyPerBucket ||
			metricsConfig.TrackLatencyPerTenant ||
			metricsConfig.TrackLatencyPerUser ||
			metricsConfig.TrackLatencyPerBucketAndMethod {

			latencySec := float64(logEntry.TotalTime) / 1000.0
			userStr, tenantStr := extractUserAndTenant(logEntry.User)
			m.LatencyObs(userStr, tenantStr, logEntry.Bucket, method, latencySec)
		}
	}
}

// Reset function
func (m *Metrics) Reset() {
	m.TotalRequests.Store(0)
	m.BytesSent.Store(0)
	m.BytesReceived.Store(0)
	m.Errors.Store(0)

	resetSyncMap(&m.RequestsByMethod)
	resetSyncMap(&m.RequestsByOperation)
	resetSyncMap(&m.RequestsPerStatusCode)
	resetSyncMap(&m.RequestsByUser)
	resetSyncMap(&m.BytesSentPerUser)
	resetSyncMap(&m.BytesReceivedByUser)
	resetSyncMap(&m.ErrorsByUser)
	resetSyncMap(&m.RequestsByBucket)
	resetSyncMap(&m.BytesSentPerBucket)
	resetSyncMap(&m.BytesReceivedPerBucket)
	resetSyncMap(&m.RequestsByIP)
	resetSyncMap(&m.RequestsByIPBucketMethodTenant)
	resetSyncMap(&m.BytesSentByIP)
	resetSyncMap(&m.BytesReceivedByIP)
	resetSyncMap(&m.ErrorsByUserAndBucket)
	resetSyncMap(&m.ErrorsByIPAndBucket)
}

// Helper function: Update max atomic value
func updateMaxAtomic(target *atomic.Uint64, value uint64) {
	for {
		curr := target.Load()
		if value > curr {
			target.Store(value)
		} else {
			break
		}
	}
}

// Helper function: Update max value in sync.Map
func updateMaxSyncMap(m *sync.Map, key string, value uint64) {
	val, _ := m.LoadOrStore(key, new(atomic.Uint64))
	atomicVal := val.(*atomic.Uint64)
	updateMaxAtomic(atomicVal, value)
}

// Helper function: Update min atomic value
func updateMinAtomic(target *atomic.Uint64, value uint64) {
	for {
		curr := target.Load()
		if curr == 0 || value < curr {
			target.Store(value)
		} else {
			break
		}
	}
}

// Helper function: Update min value in sync.Map
func updateMinSyncMap(m *sync.Map, key string, value uint64) {
	val, _ := m.LoadOrStore(key, new(atomic.Uint64))
	atomicVal := val.(*atomic.Uint64)
	updateMinAtomic(atomicVal, value)
}

// Helper function: Increment sync.Map atomic value
func incrementSyncMap(m *sync.Map, key string) {
	if val, ok := m.Load(key); ok {
		val.(*atomic.Uint64).Add(1)
		return
	}

	// Create new counter and ensure atomic operation
	newVal := new(atomic.Uint64)
	newVal.Add(1)
	actual, _ := m.LoadOrStore(key, newVal)
	if actual != newVal { // Another goroutine stored a different value first
		actual.(*atomic.Uint64).Add(1)
	}
}

// Helper function: Increment sync.Map numeric values
func incrementSyncMapValue(m *sync.Map, key string, value uint64) {
	if val, ok := m.Load(key); ok {
		val.(*atomic.Uint64).Add(value)
		return
	}

	// Create new counter and ensure atomic operation
	newVal := new(atomic.Uint64)
	newVal.Add(value)
	actual, _ := m.LoadOrStore(key, newVal)
	if actual != newVal { // Another goroutine stored a different value first
		actual.(*atomic.Uint64).Add(value)
	}
}

// Helper function: Reset sync.Map
func resetSyncMap(m *sync.Map) {
	m.Range(func(key, _ any) bool {
		m.Delete(key) // Fully removes keys
		return true
	})
}

// Helper function: Convert sync.Map to map[string]uint64
func loadSyncMap(m *sync.Map) map[string]uint64 {
	result := make(map[string]uint64)

	m.Range(func(key, value any) bool {
		if v, ok := value.(*atomic.Uint64); ok {
			result[key.(string)] = v.Load()
		} else {
			result[key.(string)] = 0 // Default to 0 if missing
		}
		return true
	})

	return result
}

// ExtractHTTPMethod extracts the HTTP method (GET, POST, PUT, DELETE) from the "uri" field
func ExtractHTTPMethod(uri string) string {
	// `strings.Fields()` splits by spaces, ensuring the first part is the method
	if len(uri) == 0 {
		return "UNKNOWN"
	}
	parts := strings.Fields(uri)
	if len(parts) == 0 {
		return "UNKNOWN"
	}
	if len(parts) > 0 {
		method := parts[0]
		switch method {
		case "GET", "PUT", "POST", "DELETE", "HEAD", "OPTIONS", "PATCH":
			return method
		}
	}
	return "UNKNOWN"
}

// var validMethodOps = map[string]map[string]bool{
// 	"GET":    {"list_buckets": true, "get_object": true},
// 	"PUT":    {"put_object": true},
// 	"DELETE": {"delete_object": true},
// 	"POST":   {"multi_part_upload": true},
// }

// // isMethodValidForOperation checks if the method is valid for a given operation
// func isMethodValidForOperation(method, operation string) bool {
// 	if ops, ok := validMethodOps[method]; ok {
// 		return ops[operation]
// 	}
// 	return false
// }

// Clone creates a deep copy of the Metrics
func (m *Metrics) Clone() *Metrics {
	clone := NewMetrics(m.LatencyObs)

	clone.TotalRequests.Store(m.TotalRequests.Load())
	clone.BytesSent.Store(m.BytesSent.Load())
	clone.BytesReceived.Store(m.BytesReceived.Load())
	clone.Errors.Store(m.Errors.Load())

	copySyncMap(&m.RequestsByMethod, &clone.RequestsByMethod)
	copySyncMap(&m.RequestsByOperation, &clone.RequestsByOperation)
	copySyncMap(&m.RequestsPerStatusCode, &clone.RequestsPerStatusCode)
	copySyncMap(&m.RequestsByUser, &clone.RequestsByUser)
	copySyncMap(&m.BytesSentPerUser, &clone.BytesSentPerUser)
	copySyncMap(&m.BytesReceivedByUser, &clone.BytesReceivedByUser)
	copySyncMap(&m.ErrorsByUser, &clone.ErrorsByUser)
	copySyncMap(&m.RequestsByBucket, &clone.RequestsByBucket)
	copySyncMap(&m.BytesSentPerBucket, &clone.BytesSentPerBucket)
	copySyncMap(&m.BytesReceivedPerBucket, &clone.BytesReceivedPerBucket)
	copySyncMap(&m.RequestsByIP, &clone.RequestsByIP)
	copySyncMap(&m.RequestsByIPBucketMethodTenant, &clone.RequestsByIPBucketMethodTenant)
	copySyncMap(&m.BytesSentByIP, &clone.BytesSentByIP)
	copySyncMap(&m.BytesReceivedByIP, &clone.BytesReceivedByIP)
	copySyncMap(&m.ErrorsByUserAndBucket, &clone.ErrorsByUserAndBucket)
	copySyncMap(&m.ErrorsByIPAndBucket, &clone.ErrorsByIPAndBucket)

	return clone
}

// copySyncMap copies keys and atomic values from one sync.Map to another
func copySyncMap(src, dst *sync.Map) {
	src.Range(func(key, val any) bool {
		if orig, ok := val.(*atomic.Uint64); ok {
			var copied atomic.Uint64
			copied.Store(orig.Load())
			dst.Store(key, &copied)
		}
		return true
	})
}

// SubtractMetrics calculates the delta between two metrics objects: total - previous
func SubtractMetrics(total, previous *Metrics) *Metrics {
	delta := NewMetrics(total.LatencyObs)

	// Handle top-level counters
	delta.TotalRequests.Store(diff(total.TotalRequests.Load(), previous.TotalRequests.Load()))
	delta.BytesSent.Store(diff(total.BytesSent.Load(), previous.BytesSent.Load()))
	delta.BytesReceived.Store(diff(total.BytesReceived.Load(), previous.BytesReceived.Load()))
	delta.Errors.Store(diff(total.Errors.Load(), previous.Errors.Load()))

	// Handle sync.Maps
	subtractSyncMap(&total.RequestsByUser, &previous.RequestsByUser, &delta.RequestsByUser)
	subtractSyncMap(&total.RequestsByMethod, &previous.RequestsByMethod, &delta.RequestsByMethod)
	subtractSyncMap(&total.RequestsByOperation, &previous.RequestsByOperation, &delta.RequestsByOperation)
	subtractSyncMap(&total.RequestsPerStatusCode, &previous.RequestsPerStatusCode, &delta.RequestsPerStatusCode)
	subtractSyncMap(&total.BytesSentPerUser, &previous.BytesSentPerUser, &delta.BytesSentPerUser)
	subtractSyncMap(&total.BytesReceivedByUser, &previous.BytesReceivedByUser, &delta.BytesReceivedByUser)
	subtractSyncMap(&total.ErrorsByUser, &previous.ErrorsByUser, &delta.ErrorsByUser)
	subtractSyncMap(&total.RequestsByBucket, &previous.RequestsByBucket, &delta.RequestsByBucket)
	subtractSyncMap(&total.BytesSentPerBucket, &previous.BytesSentPerBucket, &delta.BytesSentPerBucket)
	subtractSyncMap(&total.BytesReceivedPerBucket, &previous.BytesReceivedPerBucket, &delta.BytesReceivedPerBucket)
	subtractSyncMap(&total.RequestsByIP, &previous.RequestsByIP, &delta.RequestsByIP)
	subtractSyncMap(&total.RequestsByIPBucketMethodTenant, &previous.RequestsByIPBucketMethodTenant, &delta.RequestsByIPBucketMethodTenant)
	subtractSyncMap(&total.BytesSentByIP, &previous.BytesSentByIP, &delta.BytesSentByIP)
	subtractSyncMap(&total.BytesReceivedByIP, &previous.BytesReceivedByIP, &delta.BytesReceivedByIP)
	subtractSyncMap(&total.ErrorsByUserAndBucket, &previous.ErrorsByUserAndBucket, &delta.ErrorsByUserAndBucket)
	subtractSyncMap(&total.ErrorsByIPAndBucket, &previous.ErrorsByIPAndBucket, &delta.ErrorsByIPAndBucket)

	return delta
}

// subtractSyncMap calculates the difference and stores it in target
func subtractSyncMap(current, previous, target *sync.Map) {
	current.Range(func(key, curVal any) bool {
		cur := curVal.(*atomic.Uint64).Load()

		var prev uint64
		if prevVal, ok := previous.Load(key); ok {
			prev = prevVal.(*atomic.Uint64).Load()
		}

		delta := cur - prev
		if delta > 0 {
			var v atomic.Uint64
			v.Store(delta)
			target.Store(key, &v)
		}
		return true
	})
}

func diff(a, b uint64) uint64 {
	if a > b {
		return a - b
	}
	return 0
}
