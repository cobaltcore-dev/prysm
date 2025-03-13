// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and prysm contributors
//
// SPDX-License-Identifier: Apache-2.0

package opslog

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	json "github.com/goccy/go-json"

	"github.com/fsnotify/fsnotify"
	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog/log"
)

type S3OperationLog struct {
	Bucket             string `json:"bucket"`
	Time               string `json:"time"`
	TimeLocal          string `json:"time_local"`
	RemoteAddr         string `json:"remote_addr"`
	User               string `json:"user"`
	Operation          string `json:"operation"`
	URI                string `json:"uri"`
	HTTPStatus         string `json:"http_status"`
	ErrorCode          string `json:"error_code"`
	BytesSent          int    `json:"bytes_sent"`
	BytesReceived      int    `json:"bytes_received"`
	ObjectSize         int    `json:"object_size"`
	TotalTime          int    `json:"total_time"`
	UserAgent          string `json:"user_agent"`
	Referrer           string `json:"referrer"`
	TransID            string `json:"trans_id"`
	AuthenticationType string `json:"authentication_type"`
	AccessKeyID        string `json:"access_key_id"`
	TempURL            bool   `json:"temp_url"`
}

// CleanupBucketName extracts the actual bucket name, removing any tenant/user prefixes.
func (log *S3OperationLog) CleanupBucketName() {
	if log.Bucket == "" {
		return
	}
	parts := strings.Split(log.Bucket, "/")
	log.Bucket = parts[len(parts)-1] // Keep only the last part
}

func extractUserAndTenant(user string) (string, string) {
	parts := strings.SplitN(user, "$", 2)
	if len(parts) == 2 {
		return parts[0], parts[1] // user, tenant
	}
	return user, "none" // user without tenant
}

func StartFileOpsLogger(cfg OpsLogConfig) {
	var nc *nats.Conn
	var err error

	// Configure and connect to NATS if enabled
	if cfg.UseNats {
		nc, err = nats.Connect(cfg.NatsURL)
		if err != nil {
			log.Error().Err(err).Str("nats_url", cfg.NatsURL).Msg("Error connecting to NATS server")
			return
		}
		defer nc.Close()
		log.Info().Str("nats_url", cfg.NatsURL).Msg("Connected to NATS server")
	}

	if cfg.Prometheus {
		StartPrometheusServer(cfg.PrometheusPort)
	}

	// Initialize metrics
	metrics := NewMetrics()
	ticker := time.NewTicker(1 * time.Minute) // Aggregation interval
	defer ticker.Stop()

	// Track last file modification time to avoid re-processing old logs
	var lastModTime time.Time

	// Create a new file system watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Error().Err(err).Msg("Error creating file watcher")
		return
	}
	defer watcher.Close()

	// Start goroutine to watch file changes
	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					log.Warn().Msg("Watcher events channel closed")
					return
				}
				if event.Op&fsnotify.Write == fsnotify.Write {
					time.Sleep(100 * time.Millisecond) // Allow time for full writes (a small delay before processing writes to avoid partial reads)
					processLogEntries(cfg, nc, watcher, metrics, &lastModTime)
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					log.Warn().Msg("Watcher errors channel closed")
					return
				}
				log.Error().Err(err).Msg("File watcher encountered an error")
			}
		}
	}()

	// Add the log file to be watched
	err = watcher.Add(cfg.LogFilePath)
	if err != nil {
		log.Error().Err(err).Str("file", cfg.LogFilePath).Msg("Error adding file to watcher")
		return
	}
	log.Info().Str("file", cfg.LogFilePath).Msg("Started watching file for changes")

	// Periodically report metrics
	for range ticker.C {
		if cfg.Prometheus {
			PublishToPrometheus(metrics, cfg)
		}

		// Send the aggregated metrics to NATS and reset
		if cfg.UseNats {
			jsonData, err := metrics.ToJSON()
			if err != nil || len(jsonData) == 0 {
				log.Error().Err(err).Msg("Skipping NATS publish: JSON encoding failed or empty!")
				continue
			}
			err = PublishToNATS(nc, jsonData, fmt.Sprintf("%s.metrics", cfg.NatsMetricsSubject))
			if err != nil {
				log.Error().Err(err).Msg("Error sending metrics to NATS")
			} else {
				log.Info().Msg("Metrics sent to NATS successfully")
			}
		}
		metrics.ResetPerWindowMetrics()
	}

	// Keep the program running
	select {}
}

func processLogEntries(cfg OpsLogConfig, nc *nats.Conn, watcher *fsnotify.Watcher, metrics *Metrics, lastModTime *time.Time) {
	fileInfo, err := os.Stat(cfg.LogFilePath)
	if err != nil {
		log.Error().Err(err).Str("file", cfg.LogFilePath).Msg("Error getting log file info")
		return
	}

	// Check if the file was actually modified since the last read
	if fileInfo.ModTime().Equal(*lastModTime) {
		// log.Trace().Str("file", cfg.LogFilePath).Msg("Skipping log processing - no new data")
		return
	}
	*lastModTime = fileInfo.ModTime() // Update last modification time

	file, err := os.Open(cfg.LogFilePath)
	if err != nil {
		log.Error().Err(err).Str("file", cfg.LogFilePath).Msg("Error opening log file")
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	buf := make([]byte, 64*1024)   // 64KB buffer
	scanner.Buffer(buf, 1024*1024) // 1MB max per line

	var logPool = sync.Pool{
		New: func() any { return new(S3OperationLog) },
	}

	for scanner.Scan() {
		line := scanner.Text()
		if len(line) < 2 || line[0] != '{' || line[len(line)-1] != '}' {
			continue
		}

		// Quick check: Ensure line starts with { and ends with }
		// if !strings.HasPrefix(line, "{") || !strings.HasSuffix(line, "}") {
		// 	log.Warn().Str("raw", line).Msg("Skipping non-JSON formatted line")
		// 	continue
		// }

		// Efficient JSON parsing using streaming decoder
		// decoder := json.NewDecoder(strings.NewReader(line))
		// decoder.DisallowUnknownFields() // Prevent unexpected fields

		logEntry := logPool.Get().(*S3OperationLog)
		err := json.Unmarshal([]byte(line), logEntry)
		if err != nil {
			log.Warn().Err(err).Str("raw", line).Msg("Skipping invalid JSON entry")
			logPool.Put(logEntry) // Return to pool
			continue
		}
		// if err := decoder.Decode(&logEntry); err != nil {
		// 	log.Warn().Err(err).Str("raw", line).Msg("Skipping invalid JSON entry")
		// 	continue
		// }
		// err := json.Unmarshal([]byte(line), &logEntry)
		// if err != nil {
		// 	log.Warn().Err(err).Str("raw", line).Msg("Skipping malformed or incomplete JSON entry")
		// 	continue
		// }

		// Ignore anonymous requests if configured
		if cfg.IgnoreAnonymousRequests && logEntry.User == "anonymous" {
			log.Trace().Str("user", logEntry.User).Msg("Skipping anonymous request")
			continue
		}

		// Normalize bucket name before processing
		logEntry.CleanupBucketName()

		// Update metrics with the log entry
		metrics.Update(*logEntry)
		logPool.Put(logEntry)

		// Print to stdout if enabled
		if cfg.LogToStdout {
			logEntryBytes, err := json.MarshalIndent(logEntry, "", "  ")
			if err != nil {
				log.Error().Err(err).Msg("Error marshalling log entry for stdout")
				continue
			}
			fmt.Println(string(logEntryBytes)) // Print log entry to stdout
		}

		// Publish raw log entry to NATS
		if cfg.UseNats {
			err = PublishToNATS(nc, logEntry, cfg.NatsSubject)
			if err != nil {
				log.Error().Err(err).Msg("Error publishing log entry to NATS")
			} else {
				log.Debug().Str("user", logEntry.User).Msg("Published log entry to NATS")
			}
		}
	}

	if err := scanner.Err(); err != nil {
		log.Error().Err(err).Msg("Error scanning log file")
	}

	// Rotate log file if needed
	rotateLogIfNeeded(cfg, watcher)
}

func StartSocketOpsLogger(cfg OpsLogConfig) {
	var nc *nats.Conn
	var err error

	// Configure and connect to NATS if enabled
	if cfg.UseNats {
		nc, err = nats.Connect(cfg.NatsURL)
		if err != nil {
			log.Error().Err(err).Str("nats_url", cfg.NatsURL).Msg("Error connecting to NATS server")
			return
		}
		defer nc.Close()
		log.Info().Str("nats_url", cfg.NatsURL).Msg("Connected to NATS server")
	}

	metrics := NewMetrics()
	ticker := time.NewTicker(1 * time.Minute) // Set up a ticker to trigger every 1 minute
	defer ticker.Stop()

	// Remove any existing socket file to avoid "address already in use" errors
	err = os.Remove(cfg.SocketPath)
	if err != nil && !os.IsNotExist(err) {
		log.Error().Err(err).Str("socket_path", cfg.SocketPath).Msg("Error removing existing Unix domain socket file")
		return
	}

	// Create a new Unix domain socket listener
	listener, err := net.Listen("unix", cfg.SocketPath)
	if err != nil {
		log.Error().Err(err).Str("socket_path", cfg.SocketPath).Msg("Error creating Unix domain socket")
		return
	}
	defer func() {
		err := listener.Close()
		if err != nil {
			log.Error().Err(err).Msg("Error closing Unix domain socket listener")
		}
	}()

	log.Info().Str("socket_path", cfg.SocketPath).Msg("Listening on Unix domain socket")

	// Goroutine to handle incoming connections
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				log.Error().Err(err).Msg("Error accepting connection on Unix domain socket")
				continue
			}
			go handleConnection(cfg, conn, nc, metrics) // Handle each connection in a separate goroutine
		}
	}()

	// Use a range loop over ticker.C to handle periodic metric reporting
	for range ticker.C {
		// Every minute, send the aggregated metrics to NATS and reset
		if cfg.UseNats {
			err := PublishToNATS(nc, metrics, cfg.NatsMetricsSubject)
			if err != nil {
				log.Error().Err(err).Msg("Error sending metrics to NATS")
			} else {
				log.Info().Msg("Metrics sent to NATS successfully")
			}
		}

		// Reset metrics for the next interval
		metrics = NewMetrics()
	}
}

func handleConnection(cfg OpsLogConfig, conn net.Conn, nc *nats.Conn, metrics *Metrics) {
	defer func() {
		err := conn.Close()
		if err != nil {
			log.Error().Err(err).Msg("Error closing connection")
		}
	}()

	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		var logEntry S3OperationLog
		err := json.Unmarshal(scanner.Bytes(), &logEntry)
		if err != nil {
			log.Error().Err(err).Msg("Error unmarshalling log entry")
			continue
		}

		// Send logEntry to NATS if configured
		if cfg.UseNats {
			err := PublishToNATS(nc, logEntry, cfg.NatsSubject)
			if err != nil {
				log.Error().Err(err).Msg("Error sending op event to NATS")
			} else {
				log.Info().Msg("Op event sent to NATS successfully")
			}
		}

		// Conditional logging to stdout if enabled
		if cfg.LogToStdout {
			logEntryBytes, err := json.MarshalIndent(logEntry, "", "  ")
			if err != nil {
				log.Error().Err(err).Msg("Error marshalling log entry for stdout")
				continue
			}
			fmt.Println(string(logEntryBytes)) // Print log entry to stdout
		}

		// Publish the individual log entry to NATS or print locally
		if cfg.UseNats {
			logEntryBytes, err := json.Marshal(logEntry)
			if err != nil {
				log.Error().Err(err).Msg("Error marshalling log entry for NATS")
				continue
			}

			err = nc.Publish(cfg.NatsSubject, logEntryBytes)
			if err != nil {
				log.Error().Err(err).Msg("Error publishing log entry to NATS")
			} else {
				log.Info().Msg("Log entry published to NATS successfully")
			}
		} else {
			logEntryBytes, err := json.MarshalIndent(logEntry, "", "  ")
			if err != nil {
				log.Error().Err(err).Msg("Error marshalling log entry for local logging")
				continue
			}
			log.Trace().Msg(string(logEntryBytes))
		}
	}

	if err := scanner.Err(); err != nil {
		log.Error().Err(err).Msg("Error reading from connection")
	}
}
