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

	LatencyByMethod sync.Map // map["user|bucket|method"]

	RequestsByMethod     sync.Map // map[string]*atomic.Uint64
	RequestsByOperation  sync.Map // map[string]*atomic.Uint64
	RequestsByStatusCode sync.Map // map[string]*atomic.Uint64
	RequestsByUser       sync.Map // map[string]*atomic.Uint64
	BytesSentByUser      sync.Map // map[string]*atomic.Uint64
	BytesReceivedByUser  sync.Map // map[string]*atomic.Uint64
	ErrorsByUser         sync.Map // map[string]*atomic.Uint64

	RequestsByBucket      sync.Map // map[string]*atomic.Uint64
	BytesSentByBucket     sync.Map // map[string]*atomic.Uint64
	BytesReceivedByBucket sync.Map // map[string]*atomic.Uint64
	RequestsByIP          sync.Map // map[string]*atomic.Uint64
	BytesSentByIP         sync.Map // map[string]*atomic.Uint64
	BytesReceivedByIP     sync.Map // map[string]*atomic.Uint64
	ErrorsByUserAndBucket sync.Map // map["user|bucket|http_status"]*atomic.Uint64
	ErrorsByIPAndBucket   sync.Map // map["ip|bucket|http_status"]*atomic.Uint64
}

func NewMetrics() *Metrics {
	return &Metrics{}
}

// Convert metrics to a JSON-friendly struct
func (m *Metrics) ToJSON(metricsConfig *MetricsConfig) ([]byte, error) {
	data := map[string]any{
		"total_requests": m.TotalRequests.Load(),
		"bytes_sent":     m.BytesSent.Load(),
		"bytes_received": m.BytesReceived.Load(),
		"errors":         m.Errors.Load(),
	}

	if metricsConfig.TrackRequestsByMethod {
		data["requests_by_method"] = loadSyncMap(&m.RequestsByMethod)
	}

	if metricsConfig.TrackRequestsByOperation {
		data["requests_by_operation"] = loadSyncMap(&m.RequestsByOperation)
	}

	if metricsConfig.TrackRequestsByStatus {
		data["requests_by_status"] = loadSyncMap(&m.RequestsByStatusCode)
	}

	if metricsConfig.TrackRequestsByBucket {
		data["requests_by_bucket"] = loadSyncMap(&m.RequestsByBucket)
	}

	if metricsConfig.TrackRequestsByUser {
		data["requests_by_user"] = loadSyncMap(&m.RequestsByUser)
	}

	if metricsConfig.TrackRequestsByIP {
		data["requests_by_ip"] = loadSyncMap(&m.RequestsByIP)
	}

	// Conditional fields (bytes tracking)
	if metricsConfig.TrackBytesSentByUser {
		data["bytes_sent_by_user"] = loadSyncMap(&m.BytesSentByUser)
	}

	if metricsConfig.TrackBytesReceivedByUser {
		data["bytes_received_by_user"] = loadSyncMap(&m.BytesReceivedByUser)
	}

	if metricsConfig.TrackBytesSentByBucket {
		data["bytes_sent_by_bucket"] = loadSyncMap(&m.BytesSentByBucket)
	}

	if metricsConfig.TrackBytesReceivedByBucket {
		data["bytes_received_by_bucket"] = loadSyncMap(&m.BytesReceivedByBucket)
	}

	if metricsConfig.TrackBytesSentByIP {
		data["bytes_sent_by_ip"] = loadSyncMap(&m.BytesSentByIP)
	}

	if metricsConfig.TrackBytesReceivedByIP {
		data["bytes_received_by_ip"] = loadSyncMap(&m.BytesReceivedByIP)
	}

	// Conditional fields (errors tracking)
	if metricsConfig.TrackErrorsByUser {
		data["errors_by_user"] = loadSyncMap(&m.ErrorsByUser)
		data["errors_by_user_and_bucket"] = loadSyncMap(&m.ErrorsByUserAndBucket)
	}

	if metricsConfig.TrackErrorsByIP {
		data["errors_by_ip_and_bucket"] = loadSyncMap(&m.ErrorsByIPAndBucket)
	}

	// Latency Tracking
	if metricsConfig.TrackLatencyByMethod {
		data["latency_by_method"] = loadSyncMap(&m.LatencyByMethod)
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

	if metricsConfig.TrackRequestsByMethod {
		// Key format: "user|bucket|method"
		keyMethod := logEntry.User + "|" + logEntry.Bucket + "|" + method
		incrementSyncMap(&m.RequestsByMethod, keyMethod)
	}

	if metricsConfig.TrackRequestsByOperation {
		// Key format: "user|bucket|operation|method"
		keyOperation := logEntry.User + "|" + logEntry.Bucket + "|" + logEntry.Operation + "|" + method
		incrementSyncMap(&m.RequestsByOperation, keyOperation)
	}

	if metricsConfig.TrackRequestsByStatus {
		// Increment status code count
		incrementSyncMap(&m.RequestsByStatusCode, logEntry.HTTPStatus)
	}

	if metricsConfig.TrackRequestsByBucket {
		// Track per bucket (Bucket | Method | HTTP Status)
		keyBucket := logEntry.Bucket + "|" + method + "|" + logEntry.HTTPStatus
		incrementSyncMap(&m.RequestsByBucket, keyBucket)
	}

	if metricsConfig.TrackRequestsByUser {
		// Track per user (User | Bucket | Method | HTTP Status)
		keyUser := logEntry.User + "|" + logEntry.Bucket + "|" + method + "|" + logEntry.HTTPStatus
		incrementSyncMap(&m.RequestsByUser, keyUser)
	}

	if metricsConfig.TrackRequestsByIP {
		// Track by IP address
		keyUserIP := logEntry.User + "|" + logEntry.RemoteAddr
		incrementSyncMap(&m.RequestsByIP, keyUserIP)
	}

	// //////////////

	// Bytes Tracking
	if metricsConfig.TrackBytesSentByUser {
		incrementSyncMapValue(&m.BytesSentByUser, logEntry.User, uint64(logEntry.BytesSent))
	}

	if metricsConfig.TrackBytesReceivedByUser {
		incrementSyncMapValue(&m.BytesReceivedByUser, logEntry.User, uint64(logEntry.BytesReceived))
	}

	if metricsConfig.TrackBytesSentByBucket {
		incrementSyncMapValue(&m.BytesSentByBucket, logEntry.Bucket, uint64(logEntry.BytesSent))
	}

	if metricsConfig.TrackBytesReceivedByBucket {
		incrementSyncMapValue(&m.BytesReceivedByBucket, logEntry.Bucket, uint64(logEntry.BytesReceived))
	}

	if metricsConfig.TrackBytesSentByIP {
		keyUserIP := logEntry.User + "|" + logEntry.RemoteAddr
		incrementSyncMapValue(&m.BytesSentByIP, keyUserIP, uint64(logEntry.BytesSent))
	}

	if metricsConfig.TrackBytesReceivedByIP {
		keyUserIP := logEntry.User + "|" + logEntry.RemoteAddr
		incrementSyncMapValue(&m.BytesReceivedByIP, keyUserIP, uint64(logEntry.BytesReceived))
	}

	// //////////////

	// Error Tracking
	if logEntry.HTTPStatus[0] != '2' { // Non-2xx codes are errors
		if metricsConfig.TrackErrorsByUser {
			incrementSyncMap(&m.ErrorsByUser, logEntry.User)
		}
		if metricsConfig.TrackErrorsByUser {
			userBucketKey := logEntry.User + "|" + logEntry.Bucket + "|" + logEntry.HTTPStatus
			incrementSyncMap(&m.ErrorsByUserAndBucket, userBucketKey)
		}
		if metricsConfig.TrackErrorsByIP {
			ipBucketKey := logEntry.RemoteAddr + "|" + logEntry.Bucket + "|" + logEntry.HTTPStatus
			incrementSyncMap(&m.ErrorsByIPAndBucket, ipBucketKey)
		}
		m.Errors.Add(1) // Always track total errors
	}

	// //////////////

	// Latency Tracking
	if logEntry.TotalTime > 0 {
		latencyMs := uint64(logEntry.TotalTime)

		if metricsConfig.TrackLatencyByMethod {
			// Key format: "user|bucket|method"
			latencyKey := logEntry.User + "|" + logEntry.Bucket + "|" + method
			incrementSyncMapValue(&m.LatencyByMethod, latencyKey, latencyMs)
		}
	}

}

// Reset function
func (m *Metrics) Reset() {
	m.TotalRequests.Store(0)
	m.BytesSent.Store(0)
	m.BytesReceived.Store(0)
	m.Errors.Store(0)

	resetSyncMap(&m.LatencyByMethod)
	resetSyncMap(&m.RequestsByMethod)
	resetSyncMap(&m.RequestsByOperation)
	resetSyncMap(&m.RequestsByStatusCode)
	resetSyncMap(&m.RequestsByUser)
	resetSyncMap(&m.BytesSentByUser)
	resetSyncMap(&m.BytesReceivedByUser)
	resetSyncMap(&m.ErrorsByUser)
	resetSyncMap(&m.RequestsByBucket)
	resetSyncMap(&m.BytesSentByBucket)
	resetSyncMap(&m.BytesReceivedByBucket)
	resetSyncMap(&m.RequestsByIP)
	resetSyncMap(&m.BytesSentByIP)
	resetSyncMap(&m.BytesReceivedByIP)
	resetSyncMap(&m.ErrorsByUserAndBucket)
	resetSyncMap(&m.ErrorsByIPAndBucket)
}

func (m *Metrics) ResetPerWindowMetrics() {
	resetSyncMap(&m.LatencyByMethod)
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
