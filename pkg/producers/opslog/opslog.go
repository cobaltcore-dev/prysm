// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and prysm contributors
//
// SPDX-License-Identifier: Apache-2.0

package opslog

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"time"

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

	// Initialize metrics
	metrics := NewMetrics()
	ticker := time.NewTicker(1 * time.Minute) // Set up a ticker to trigger every 1 minute
	defer ticker.Stop()

	// Create a new file system watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Error().Err(err).Msg("Error creating file watcher")
		return
	}
	defer watcher.Close()

	// Start a goroutine to handle file system events
	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					log.Warn().Msg("Watcher events channel closed")
					return
				}
				if event.Op&fsnotify.Write == fsnotify.Write {
					log.Info().Str("file", event.Name).Msg("File modified")
					processLogEntries(cfg, nc, watcher, metrics)
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
		// Send the aggregated metrics to NATS and reset
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

	// Keep the program running
	select {}
}

func processLogEntries(cfg OpsLogConfig, nc *nats.Conn, watcher *fsnotify.Watcher, metrics *Metrics) {
	file, err := os.Open(cfg.LogFilePath)
	if err != nil {
		log.Error().Err(err).Str("file", cfg.LogFilePath).Msg("Error opening log file")
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		var logEntry S3OperationLog
		err := json.Unmarshal(scanner.Bytes(), &logEntry)
		if err != nil {
			log.Error().Err(err).Msg("Error unmarshalling log entry")
			continue
		}

		// Update metrics with the log entry
		metrics.Update(logEntry)

		// Optionally print the log entry to stdout based on configuration
		if cfg.LogToStdout {
			logEntryBytes, err := json.MarshalIndent(logEntry, "", "  ")
			if err != nil {
				log.Error().Err(err).Msg("Error marshalling log entry for stdout")
				continue
			}
			fmt.Println(string(logEntryBytes)) // Print log entry to stdout
		}

		if cfg.UseNats {
			// Skip anonymous entries if necessary
			if logEntry.User != "anonymous" {
				err = PublishToNATS(nc, logEntry, cfg.NatsSubject)
				if err != nil {
					log.Error().Err(err).Msg("Error publishing log entry to NATS")
				} else {
					log.Debug().Str("user", logEntry.User).Msg("Published log entry to NATS")
				}
			} else {
				log.Debug().Str("user", logEntry.User).Msg("Skipped anonymous log entry")
			}
		} else {
			logEntryBytes, err := json.MarshalIndent(logEntry, "", "  ")
			if err != nil {
				log.Error().Err(err).Msg("Error marshalling log entry for local logging")
				continue
			}
			fmt.Println(string(logEntryBytes))
		}
	}

	if err := scanner.Err(); err != nil {
		log.Error().Err(err).Msg("Error scanning log file")
	}

	// // Truncate the log file after processing
	// err = os.Truncate(cfg.LogFilePath, 0)
	// if err != nil {
	// 	log.Error().Err(err).Str("file", cfg.LogFilePath).Msg("Error truncating log file")
	// } else {
	// 	log.Info().Str("file", cfg.LogFilePath).Msg("Log file truncated successfully")
	// }

	// Check if the log file should be rotated
	rotateLogIfNeeded(cfg, watcher)
}

func rotateLogIfNeeded(cfg OpsLogConfig, watcher *fsnotify.Watcher) {
	fileInfo, err := os.Stat(cfg.LogFilePath)
	if err != nil {
		log.Error().Err(err).Str("file", cfg.LogFilePath).Msg("Error getting log file info")
		return
	}

	// Check if the log file should be rotated based on size
	if fileInfo.Size() >= cfg.MaxLogFileSize*1024*1024 {
		err = rotateLogFile(cfg, watcher)
		if err != nil {
			log.Error().Err(err).Str("file", cfg.LogFilePath).Msg("Error rotating log file")
		} else {
			log.Info().Str("file", cfg.LogFilePath).Msg("Log file rotated due to size")
		}
		return
	}

	// Check if the log file should be rotated based on time
	logFileAgeHours := time.Since(fileInfo.ModTime()).Hours()
	if logFileAgeHours >= float64(cfg.LogRetentionDays*24) {
		err = rotateLogFile(cfg, watcher)
		if err != nil {
			log.Error().Err(err).Str("file", cfg.LogFilePath).Msg("Error rotating log file")
		} else {
			log.Info().Str("file", cfg.LogFilePath).Msg("Log file rotated due to age")
		}
	}
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
			err := PublishToNATS(nc, metrics, cfg.NatsSubject) //FIXME it should be different subject for metrics
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
