// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and prysm contributors
//
// SPDX-License-Identifier: Apache-2.0

package opslog

import (
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/rs/zerolog/log"
)

func rotateLogIfNeeded(cfg OpsLogConfig, watcher *fsnotify.Watcher) {
	fileInfo, err := os.Stat(cfg.LogFilePath)
	if err != nil {
		log.Error().Err(err).Str("file", cfg.LogFilePath).Msg("Error getting log file info")
		return
	}

	fileSizeMB := float64(fileInfo.Size()) / (1024 * 1024) // Convert bytes to MB
	maxSizeMB := float64(cfg.MaxLogFileSize)

	shouldRotate := false

	// Check if the log file should be rotated due to size
	if maxSizeMB > 0 && fileSizeMB >= maxSizeMB {
		log.Warn().
			Str("file", cfg.LogFilePath).
			Float64("size_mb", fileSizeMB).
			Float64("max_size_mb", maxSizeMB).
			Msg("Rotating log due to size limit")
		shouldRotate = true
	}

	// Check if the log file should be rotated due to age
	logFileAgeHours := time.Since(fileInfo.ModTime()).Hours()
	if cfg.LogRetentionDays > 0 && logFileAgeHours >= float64(cfg.LogRetentionDays*24) {
		log.Warn().
			Str("file", cfg.LogFilePath).
			Float64("age_hours", logFileAgeHours).
			Msg("Rotating log due to age limit")
		shouldRotate = true
	}

	// Rotate only if necessary
	if shouldRotate {
		if err := rotateLogFile(cfg, watcher); err != nil {
			log.Error().Err(err).Str("file", cfg.LogFilePath).Msg("Error rotating log file")
		} else {
			log.Info().Str("file", cfg.LogFilePath).Msg("Log file rotated successfully")
		}
	}
}

func rotateLogFile(cfg OpsLogConfig, watcher *fsnotify.Watcher) error {
	logDir := filepath.Dir(cfg.LogFilePath)
	timestamp := time.Now().Format("20060102-150405")
	rotatedLogPath := filepath.Join(logDir, "radosgw.log."+timestamp)

	// Step 1: Copy the log file contents to a new rotated file
	srcFile, err := os.Open(cfg.LogFilePath)
	if err != nil {
		log.Error().Err(err).Msg("Error opening log file for rotation")
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(rotatedLogPath)
	if err != nil {
		log.Error().Err(err).Msg("Error creating rotated log file")
		return err
	}

	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		log.Error().Err(err).Msg("Error copying log file contents to rotated file")
		dstFile.Close()
		return err
	}
	dstFile.Close()

	// Step 2: Truncate the original log file to 0 bytes (like copytruncate)
	err = os.Truncate(cfg.LogFilePath, 0)
	if err != nil {
		log.Error().Err(err).Msg("Error truncating log file")
		return err
	}

	log.Info().Str("rotatedLogPath", rotatedLogPath).Msg("Rotated log file successfully using copytruncate method")

	// Step 3: Ensure the file watcher remains intact
	_ = watcher.Remove(cfg.LogFilePath)
	_ = watcher.Add(cfg.LogFilePath)

	// Cleanup old log files asynchronously
	go deleteOldLogs(cfg)

	return nil
}

func deleteOldLogs(cfg OpsLogConfig) {
	// Define the directory and pattern for rotated logs
	logDir := filepath.Dir(cfg.LogFilePath)
	logPattern := filepath.Join(logDir, "radosgw.log.*")
	// logPattern := filepath.Join(logDir, filepath.Base(cfg.LogFilePath)+".*")

	// Get the current time
	now := time.Now()

	// Get a list of matching log files
	files, err := filepath.Glob(logPattern)
	if err != nil {
		log.Error().Err(err).Msg("Error finding rotated log files")
		return
	}

	// Iterate over matched files
	for _, path := range files {
		info, err := os.Lstat(path) // Use Lstat to handle symbolic links
		if err != nil {
			log.Warn().Err(err).Str("file", path).Msg("Skipping file due to error accessing metadata")
			continue
		}

		// Skip directories
		if info.IsDir() {
			continue
		}

		// Check the file's modification time
		if now.Sub(info.ModTime()).Hours() > float64(cfg.LogRetentionDays*24) {
			// Attempt to delete old log file
			if err := os.Remove(path); err != nil {
				log.Warn().Err(err).Str("file", path).Msg("Failed to delete old log file (might be in use or permissions issue)")
			} else {
				log.Info().Str("file", path).Msg("Successfully deleted old log file")
			}
		}
	}
}
