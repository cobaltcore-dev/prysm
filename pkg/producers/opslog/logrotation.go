// Copyright 2024 Clyso GmbH
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package opslog

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/rs/zerolog/log"
)

func rotateLogFile(cfg OpsLogConfig, watcher *fsnotify.Watcher) error {
	// Ensure log directory exists
	logDir := filepath.Dir(cfg.LogFilePath)
	if _, err := os.Stat(logDir); os.IsNotExist(err) {
		err := os.MkdirAll(logDir, 0755)
		if err != nil {
			log.Error().Err(err).Msg("Error creating log directory")
			return fmt.Errorf("error creating log directory: %w", err)
		}
	}

	// Rotate the log file with a timestamp
	timestamp := time.Now().Format("20060102-150405")
	rotatedLogPath := fmt.Sprintf("%s.%s", cfg.LogFilePath, timestamp)

	// Rename the current log file
	err := os.Rename(cfg.LogFilePath, rotatedLogPath)
	if err != nil {
		log.Error().Err(err).Msg("Error rotating log file")
		return fmt.Errorf("error rotating log file: %w", err)
	}

	// Remove the old file from the watcher
	err = watcher.Remove(cfg.LogFilePath)
	if err != nil {
		log.Error().Err(err).Str("file", cfg.LogFilePath).Msg("Error removing old file from watcher")
	}

	// Create a new log file
	newFile, err := os.Create(cfg.LogFilePath)
	if err != nil {
		// Attempt to restore the rotated file if creating the new log file fails
		restoreErr := os.Rename(rotatedLogPath, cfg.LogFilePath)
		if restoreErr != nil {
			log.Error().Err(restoreErr).Msg("Error restoring old log file after failed creation of new log file")
			return fmt.Errorf("error creating new log file and restoring old file: %w, %v", err, restoreErr)
		}
		log.Error().Err(err).Msg("Error creating new log file")
		return fmt.Errorf("error creating new log file: %w", err)
	}
	newFile.Close()

	// Add the new file to the watcher
	err = watcher.Add(cfg.LogFilePath)
	if err != nil {
		log.Error().Err(err).Str("file", cfg.LogFilePath).Msg("Error adding new file to watcher")
		return fmt.Errorf("error adding new file to watcher: %w", err)
	}

	log.Info().Str("rotatedLogPath", rotatedLogPath).Msg("Rotated log file")

	// Optionally delete older rotated files in a goroutine
	go deleteOldLogs(cfg)

	return nil
}

func deleteOldLogs(cfg OpsLogConfig) {
	// Define the directory and pattern for rotated logs
	logDir := filepath.Dir(cfg.LogFilePath)
	logPattern := filepath.Base(cfg.LogFilePath) + ".*"

	// Get the current time
	now := time.Now()

	// Walk through the log directory and find files matching the pattern
	err := filepath.Walk(logDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			log.Error().Err(err).Str("file", path).Msg("Error accessing file")
			return nil
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Check if the file matches the rotated log pattern
		matched, err := filepath.Match(logPattern, info.Name())
		if err != nil {
			log.Error().Err(err).Str("file", info.Name()).Msg("Error matching file")
			return nil
		}

		if matched {
			// Check the file's modification time
			if now.Sub(info.ModTime()).Hours() > float64(cfg.LogRetentionDays*24) {
				// Delete the file if it is older than the retention period
				err := os.Remove(path)
				if err != nil {
					log.Error().Err(err).Str("file", path).Msg("Error deleting old log file")
				} else {
					log.Info().Str("file", path).Msg("Deleted old log file")
				}
			}
		}

		return nil
	})

	if err != nil {
		log.Error().Err(err).Msg("Error walking the log directory")
	}
}
