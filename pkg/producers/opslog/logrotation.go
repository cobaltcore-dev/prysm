// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and prysm contributors
//
// SPDX-License-Identifier: Apache-2.0

package opslog

import (
	"os"
	"path/filepath"
	"syscall"
	"time"

	"io"

	"github.com/fsnotify/fsnotify"
	"github.com/rs/zerolog/log"
)

// rotateLogFile truncates the log file while preserving ModTime and ownership
func rotateLogFile(cfg OpsLogConfig, watcher *fsnotify.Watcher) error {
	logDir := filepath.Dir(cfg.LogFilePath)
	timestamp := time.Now().Format("20060102-150405")
	rotatedLogPath := filepath.Join(logDir, "radosgw.log."+timestamp)

	// Step 1: Retrieve original file ownership (UID, GID) and ModTime
	var originalStat syscall.Stat_t
	if err := syscall.Stat(cfg.LogFilePath, &originalStat); err != nil {
		log.Error().Err(err).Msg("Error getting log file metadata")
		return err
	}
	originalUID := int(originalStat.Uid)
	originalGID := int(originalStat.Gid)
	originalModTime := time.Unix(originalStat.Mtim.Sec, originalStat.Mtim.Nsec)

	// Step 2: Copy the current log file to a rotated version
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

	// Step 3: Truncate the original log file (clear content)
	err = os.Truncate(cfg.LogFilePath, 0)
	if err != nil {
		log.Error().Err(err).Msg("Error truncating log file")
		return err
	}

	// Step 4: Restore original UID/GID on the truncated file
	if err := os.Chown(cfg.LogFilePath, originalUID, originalGID); err != nil {
		log.Error().Err(err).Msg("Error restoring UID/GID on new log file")
		return err
	}

	// Step 5: Restore the original ModTime
	if err := os.Chtimes(cfg.LogFilePath, originalModTime, originalModTime); err != nil {
		log.Error().Err(err).Msg("Error restoring ModTime on log file")
		return err
	}

	log.Info().Str("rotatedLogPath", rotatedLogPath).Msg("Rotated log file successfully, preserving metadata")

	// Ensure the file watcher remains intact
	_ = watcher.Remove(cfg.LogFilePath)
	_ = watcher.Add(cfg.LogFilePath)

	// Cleanup old log files asynchronously
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
