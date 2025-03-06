// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and prysm contributors
//
// SPDX-License-Identifier: Apache-2.0

package opslog

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
)

type Metrics struct {
	TotalRequests         atomic.Uint64
	BytesSent             atomic.Uint64
	BytesReceived         atomic.Uint64
	Errors                atomic.Uint64
	LatencySum            atomic.Uint64
	LatencyCount          atomic.Uint64
	MaxLatency            atomic.Uint64
	MinLatency            atomic.Uint64
	RequestsByMethod      sync.Map // map[string]*atomic.Uint64
	RequestsByOperation   sync.Map // map[string]*atomic.Uint64
	RequestsByStatusCode  sync.Map // map[string]*atomic.Uint64
	RequestsByUser        sync.Map // map[string]*atomic.Uint64
	BytesSentByUser       sync.Map // map[string]*atomic.Uint64
	BytesReceivedByUser   sync.Map // map[string]*atomic.Uint64
	ErrorsByUser          sync.Map // map[string]*atomic.Uint64
	LatencyMaxByUser      sync.Map // user -> atomic.Uint64 (stores latency in milliseconds)
	LatencyMinByUser      sync.Map // user -> atomic.Uint64 (stores latency in milliseconds)
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
	return &Metrics{
		MaxLatency: atomic.Uint64{}, // Initially 0, updated dynamically
		MinLatency: atomic.Uint64{}, // Initially 0, updated dynamically
	}
}

// Convert metrics to a JSON-friendly struct
func (m *Metrics) ToJSON() ([]byte, error) {
	data := map[string]interface{}{
		"total_requests":            m.TotalRequests.Load(),
		"bytes_sent":                m.BytesSent.Load(),
		"bytes_received":            m.BytesReceived.Load(),
		"errors":                    m.Errors.Load(),
		"latency_sum":               m.LatencySum.Load(),
		"latency_count":             m.LatencyCount.Load(),
		"max_latency":               m.MaxLatency.Load(),
		"min_latency":               m.MinLatency.Load(),
		"requests_by_method":        loadSyncMap(&m.RequestsByMethod),
		"requests_by_op":            loadSyncMap(&m.RequestsByOperation),
		"requests_by_status":        loadSyncMap(&m.RequestsByStatusCode),
		"requests_by_user":          loadSyncMap(&m.RequestsByUser),
		"bytes_sent_by_user":        loadSyncMap(&m.BytesSentByUser),
		"bytes_received_by_user":    loadSyncMap(&m.BytesReceivedByUser),
		"errors_by_user":            loadSyncMap(&m.ErrorsByUser),
		"max_latency_by_user":       loadSyncMap(&m.LatencyMaxByUser),
		"min_latency_by_user":       loadSyncMap(&m.LatencyMinByUser),
		"requests_by_bucket":        loadSyncMap(&m.RequestsByBucket),
		"bytes_sent_by_bucket":      loadSyncMap(&m.BytesSentByBucket),
		"bytes_received_by_bucket":  loadSyncMap(&m.BytesReceivedByBucket),
		"requests_by_ip":            loadSyncMap(&m.RequestsByIP),
		"bytes_sent_by_ip":          loadSyncMap(&m.BytesSentByIP),
		"bytes_received_by_ip":      loadSyncMap(&m.BytesReceivedByIP),
		"errors_by_user_and_bucket": loadSyncMap(&m.ErrorsByUserAndBucket),
		"errors_by_ip_and_bucket":   loadSyncMap(&m.ErrorsByIPAndBucket),
	}

	// Ensure JSON encoding is safe
	jsonData, err := json.Marshal(data)
	if err != nil {
		fmt.Println("Error encoding metrics to JSON:", err) // Debugging output
		return nil, err
	}
	fmt.Print(string(jsonData))
	return jsonData, nil
}

// Update increments metrics based on a new log entry
func (m *Metrics) Update(logEntry S3OperationLog) {
	m.TotalRequests.Add(1)
	m.BytesSent.Add(uint64(logEntry.BytesSent))
	m.BytesReceived.Add(uint64(logEntry.BytesReceived))

	// Extract HTTP method from logEntry.URI
	method := ExtractHTTPMethod(logEntry.URI)

	// Ensure method and operation are consistent
	// if !isMethodValidForOperation(method, logEntry.Operation) {
	// 	fmt.Printf("Warning: Invalid method-operation pair: %s - %s\n", method, logEntry.Operation)
	// 	return // Ignore invalid pairs
	// }

	// Key format: "user|bucket|method"
	keyMethod := fmt.Sprintf("%s|%s|%s", logEntry.User, logEntry.Bucket, method)
	incrementSyncMap(&m.RequestsByMethod, keyMethod)

	// Key format: "user|bucket|operation|method"
	keyOperation := fmt.Sprintf("%s|%s|%s|%s", logEntry.User, logEntry.Bucket, logEntry.Operation, method)
	incrementSyncMap(&m.RequestsByOperation, keyOperation)

	// Increment status code count
	incrementSyncMap(&m.RequestsByStatusCode, logEntry.HTTPStatus)

	// Track by user
	incrementSyncMap(&m.RequestsByUser, logEntry.User)
	incrementSyncMapValue(&m.BytesSentByUser, logEntry.User, uint64(logEntry.BytesSent))
	incrementSyncMapValue(&m.BytesReceivedByUser, logEntry.User, uint64(logEntry.BytesReceived))

	// Track by bucket
	incrementSyncMap(&m.RequestsByBucket, logEntry.Bucket)
	incrementSyncMapValue(&m.BytesSentByBucket, logEntry.Bucket, uint64(logEntry.BytesSent))
	incrementSyncMapValue(&m.BytesReceivedByBucket, logEntry.Bucket, uint64(logEntry.BytesReceived))

	// Track by IP address
	keyUserIP := logEntry.User + "|" + logEntry.RemoteAddr

	incrementSyncMap(&m.RequestsByIP, keyUserIP)
	incrementSyncMapValue(&m.BytesSentByIP, keyUserIP, uint64(logEntry.BytesSent))
	incrementSyncMapValue(&m.BytesReceivedByIP, keyUserIP, uint64(logEntry.BytesReceived))

	// Track errors per user
	if logEntry.HTTPStatus[0] != '2' { // Non-2xx codes are errors
		incrementSyncMap(&m.ErrorsByUser, logEntry.User)
		m.Errors.Add(1)
	}

	// Track latency per user
	if logEntry.TotalTime > 0 {
		latencyMs := uint64(logEntry.TotalTime)

		// Track total latency for averaging
		m.LatencySum.Add(latencyMs)
		m.LatencyCount.Add(1)

		// Track per-user max latency
		val, _ := m.LatencyMaxByUser.LoadOrStore(logEntry.User, &atomic.Uint64{})
		atomicMax := val.(*atomic.Uint64)
		for {
			currMax := atomicMax.Load()
			if latencyMs > currMax {
				atomicMax.Store(latencyMs)
			} else {
				break
			}
		}

		// Track per-user min latency
		minVal, _ := m.LatencyMinByUser.LoadOrStore(logEntry.User, &atomic.Uint64{})
		atomicMin := minVal.(*atomic.Uint64)
		for {
			currMin := atomicMin.Load()
			if currMin == 0 || latencyMs < currMin {
				atomicMin.Store(latencyMs)
			} else {
				break
			}
		}
	}

	// Format keys
	userBucketKey := fmt.Sprintf("%s|%s|%s", logEntry.User, logEntry.Bucket, logEntry.HTTPStatus)
	ipBucketKey := fmt.Sprintf("%s|%s|%s", logEntry.RemoteAddr, logEntry.Bucket, logEntry.HTTPStatus)

	// Track errors by User + Bucket + HTTP Status
	incrementSyncMap(&m.ErrorsByUserAndBucket, userBucketKey)

	// Track errors by IP + Bucket + HTTP Status
	incrementSyncMap(&m.ErrorsByIPAndBucket, ipBucketKey)
}

// Reset function
func (m *Metrics) Reset() {
	m.TotalRequests.Store(0)
	m.BytesSent.Store(0)
	m.BytesReceived.Store(0)
	m.Errors.Store(0)
	m.LatencySum.Store(0)
	m.LatencyCount.Store(0)
	m.MaxLatency.Store(0)
	m.MinLatency.Store(0)

	resetSyncMap(&m.RequestsByMethod)
	resetSyncMap(&m.RequestsByOperation)
	resetSyncMap(&m.RequestsByStatusCode)
	resetSyncMap(&m.RequestsByUser)
	resetSyncMap(&m.BytesSentByUser)
	resetSyncMap(&m.BytesReceivedByUser)
	resetSyncMap(&m.ErrorsByUser)
	resetSyncMap(&m.LatencyMaxByUser)
	resetSyncMap(&m.LatencyMinByUser)
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
	m.LatencySum.Store(0)
	m.LatencyCount.Store(0)
	m.MaxLatency.Store(0)
	m.MinLatency.Store(0)

	resetSyncMap(&m.LatencyMaxByUser)
	resetSyncMap(&m.LatencyMinByUser)
}

// Helper function: Increment sync.Map atomic value
func incrementSyncMap(m *sync.Map, key string) {
	val, _ := m.LoadOrStore(key, new(atomic.Uint64))
	val.(*atomic.Uint64).Add(1)
}

// Helper function: Increment sync.Map numeric values
func incrementSyncMapValue(m *sync.Map, key string, value uint64) {
	val, _ := m.LoadOrStore(key, new(atomic.Uint64))
	val.(*atomic.Uint64).Add(value)
}

// Helper function: Reset sync.Map
func resetSyncMap(m *sync.Map) {
	m.Range(func(key, value interface{}) bool {
		m.Store(key, new(atomic.Uint64))
		return true
	})
}

// Helper function: Convert sync.Map to map[string]uint64
func loadSyncMap(m *sync.Map) map[string]uint64 {
	result := make(map[string]uint64)

	m.Range(func(key, value interface{}) bool {
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
	parts := strings.Fields(uri)
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
