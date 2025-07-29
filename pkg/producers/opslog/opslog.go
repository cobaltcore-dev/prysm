// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and prysm contributors
//
// SPDX-License-Identifier: Apache-2.0

package opslog

import (
	"bufio"
	"fmt"
	"io"
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

	// Configure and connect to NATS if enabled
	if cfg.UseNats {
		nc = connectToNATS(cfg)
		if nc == nil {
			return
		}
		defer nc.Close()
	}

	if cfg.Prometheus {
		StartPrometheusServer(cfg.PrometheusPort, &cfg)
	}

	// Initialize metrics
	metrics := NewMetrics(LatencyObs)
	interval := time.Duration(cfg.PrometheusIntervalSeconds) * time.Second
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	watcher := createLogWatcher(cfg)
	if watcher == nil {
		return
	}
	defer watcher.Close()

	startLogWatchLoop(cfg, nc, watcher, metrics)

	if cfg.TruncateLogOnStart && cfg.LogFilePath != "" {
		if err := rotateLogFile(cfg, watcher); err != nil {
			log.Error().Err(err).Str("file", cfg.LogFilePath).Msg("Error rotating log file")
		} else {
			log.Info().Str("file", cfg.LogFilePath).Msg("Log file rotated successfully")
		}
	}

	for range ticker.C {
		if cfg.Prometheus {
			PublishToPrometheus(metrics, cfg)
		}

		if cfg.UseNats {
			publishMetricsToNATS(cfg, nc, metrics)
		}
	}

	// Keep the program running
	select {}
}

func connectToNATS(cfg OpsLogConfig) *nats.Conn {
	nc, err := nats.Connect(cfg.NatsURL)
	if err != nil {
		log.Error().Err(err).Str("nats_url", cfg.NatsURL).Msg("Error connecting to NATS server")
		return nil
	}
	log.Info().Str("nats_url", cfg.NatsURL).Msg("Connected to NATS server")
	return nc
}

func createLogWatcher(cfg OpsLogConfig) *fsnotify.Watcher {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Error().Err(err).Msg("Error creating file watcher")
		return nil
	}

	err = watcher.Add(cfg.LogFilePath)
	if err != nil {
		log.Error().Err(err).Str("file", cfg.LogFilePath).Msg("Error adding file to watcher")
		watcher.Close()
		return nil
	}

	log.Info().Str("file", cfg.LogFilePath).Msg("Started watching file for changes")
	return watcher
}

func startLogWatchLoop(cfg OpsLogConfig, nc *nats.Conn, watcher *fsnotify.Watcher, metrics *Metrics) {
	// var lastModTime time.Time
	var lastOffset int64 = 0

	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					log.Warn().Msg("Watcher events channel closed")
					return
				}

				if event.Op&fsnotify.Write == fsnotify.Write {
					time.Sleep(100 * time.Millisecond)

					offset, err := processLogEntries(cfg, nc, watcher, metrics, lastOffset)
					if err != nil {
						log.Error().Err(err).Msg("Failed to process log entries")
						continue
					}
					lastOffset = offset
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
}

func publishMetricsToNATS(cfg OpsLogConfig, nc *nats.Conn, metrics *Metrics) {
	jsonData, err := metrics.ToJSON(&cfg.MetricsConfig)
	if err != nil || len(jsonData) == 0 {
		log.Error().Err(err).Msg("Skipping NATS publish: JSON encoding failed or empty!")
		return
	}
	err = PublishToNATS(nc, jsonData, fmt.Sprintf("%s.metrics", cfg.NatsMetricsSubject))
	if err != nil {
		log.Error().Err(err).Msg("Error sending metrics to NATS")
	} else {
		log.Info().Msg("Metrics sent to NATS successfully")
	}
}

func processLogEntries(cfg OpsLogConfig, nc *nats.Conn, watcher *fsnotify.Watcher, metrics *Metrics, lastOffset int64) (int64, error) {
	file, err := os.Open(cfg.LogFilePath)
	if err != nil {
		return lastOffset, fmt.Errorf("error opening log file: %w", err)
	}
	defer file.Close()

	fileInfo, err := os.Stat(cfg.LogFilePath)
	if err != nil {
		return lastOffset, fmt.Errorf("error stat'ing log file: %w", err)
	}
	currentSize := fileInfo.Size()

	if lastOffset > currentSize {
		log.Warn().
			Int64("lastOffset", lastOffset).
			Int64("currentSize", currentSize).
			Msg("Detected log file truncation. Resetting offset to 0.")
		lastOffset = 0
	}

	// Seek to last known position
	_, err = file.Seek(lastOffset, io.SeekStart)
	if err != nil {
		return lastOffset, fmt.Errorf("failed to seek log file: %w", err)
	}

	// reader := bufio.NewReader(file)
	reader := bufio.NewReaderSize(file, 64*1024)
	var newOffset = lastOffset

	logPool := sync.Pool{
		New: func() any { return new(S3OperationLog) },
	}

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			return newOffset, fmt.Errorf("error reading log file: %w", err)
		}

		// Update offset
		newOffset += int64(len(line))

		str := strings.TrimSpace(string(line))
		if len(str) < 2 || str[0] != '{' || str[len(str)-1] != '}' {
			continue
		}

		logEntry := logPool.Get().(*S3OperationLog)
		if err := json.Unmarshal([]byte(line), logEntry); err != nil {
			log.Warn().Err(err).Str("raw", str).Msg("Skipping invalid JSON entry")
			logPool.Put(logEntry)
			continue
		}

		// Ignore anonymous requests if configured
		if cfg.IgnoreAnonymousRequests && logEntry.User == "anonymous" {
			log.Trace().Str("user", logEntry.User).Msg("Skipping anonymous request")
			continue
		}

		// Normalize bucket name before processing
		logEntry.CleanupBucketName()

		// Update metrics with the log entry
		metrics.Update(*logEntry, &cfg.MetricsConfig)

		logPool.Put(logEntry)

		// Print to stdout if enabled
		if cfg.LogToStdout {
			var b []byte
			var err error
			if cfg.LogPrettyPrint {
				b, err = json.MarshalIndent(logEntry, "", "  ")
			} else {
				b, err = json.Marshal(logEntry)
			}
			if err == nil {
				fmt.Println(string(b))
			}
		}

		// Publish raw log entry to NATS
		if cfg.UseNats {
			if err := PublishToNATS(nc, logEntry, cfg.NatsSubject); err != nil {
				log.Error().Err(err).Msg("Error publishing log entry to NATS")
			}
		}
	}

	// Rotate log file if needed
	rotateLogIfNeeded(cfg, watcher)
	return newOffset, nil
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

	metrics := NewMetrics(latencyObs)
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
		metrics = NewMetrics(latencyObs)
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
			var b []byte
			var err error
			if cfg.LogPrettyPrint {
				b, err = json.MarshalIndent(logEntry, "", "  ")
			} else {
				b, err = json.Marshal(logEntry)
			}
			if err != nil {
				log.Error().Err(err).Msg("Error marshalling log entry for stdout")
				continue
			}
			fmt.Println(string(b)) // Print log entry to stdout
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
