// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and prysm contributors
//
// SPDX-License-Identifier: Apache-2.0

package diskhealthmetrics

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/rs/zerolog/log"
	"golang.org/x/sys/unix"
)

// Cache for OSD mappings: physical device -> OSD ID
var physicalDeviceToOSDCache = make(map[string]string)
var cacheInitialized = false

// normalizeDevicePath ensures we always use the same canonical path
func normalizeDevicePath(device string) string {
	// Try to get canonical path
	if canonical, err := filepath.EvalSymlinks(device); err == nil {
		return canonical
	}
	// If EvalSymlinks fails, use the original path
	return device
}

func getOSDIDForDisk(disk, basePath string) (string, error) {
	// Skip if no base path provided
	if basePath == "" {
		return "", nil
	}

	// For NVMe controller devices, discover namespace devices using glob
	actualDisks := []string{disk}
	if strings.HasPrefix(disk, "/dev/nvme") {
		// Check if this is a controller device (just /dev/nvme + number)
		suffix := strings.TrimPrefix(disk, "/dev/nvme")
		if _, err := strconv.Atoi(suffix); err == nil {
			// This is a controller device like /dev/nvme1, discover all namespaces
			globPattern := disk + "n*"
			if matches, err := filepath.Glob(globPattern); err == nil && len(matches) > 0 {
				actualDisks = matches
				for _, ns := range matches {
					log.Debug().Str("original", disk).Str("namespace", ns).Msg("Found namespace device")
				}
			}
		}
	}

	// Initialize the cache if not done yet
	if err := initOSDMappingCache(basePath); err != nil {
		log.Warn().Err(err).Msg("Failed to initialize OSD mapping cache")
		return "", nil
	}

	// Try to find OSD ID for any of the actual disks
	for _, actualDisk := range actualDisks {
		// Normalize the device path for consistent lookup
		normalizedDisk := normalizeDevicePath(actualDisk)

		// Try direct lookup with normalized path
		if osdID, found := physicalDeviceToOSDCache[normalizedDisk]; found {
			log.Debug().Str("disk", actualDisk).Str("osd_id", osdID).Msg("Found OSD ID for disk")
			return osdID, nil
		}

		// Also try with the original path in case normalization changed it
		if normalizedDisk != actualDisk {
			if osdID, found := physicalDeviceToOSDCache[actualDisk]; found {
				log.Debug().Str("disk", actualDisk).Str("osd_id", osdID).Msg("Found OSD ID for disk (original path)")
				return osdID, nil
			}
		}
	}

	log.Debug().Str("disk", disk).Msg("No OSD ID found for disk")
	return "", nil
}

// resolveDeviceMapperSlaves recursively resolves dm-* devices to physical devices
func resolveDeviceMapperSlaves(dev string) ([]string, error) {
	path := filepath.Join("/sys/block", dev, "slaves")

	// Check if slaves directory exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		// No slaves directory means this is a leaf device
		return []string{"/dev/" + dev}, nil
	}

	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}

	if len(entries) == 0 {
		// No slaves means this is a leaf device
		return []string{"/dev/" + dev}, nil
	}

	var devices []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		slave := entry.Name()

		// If the slave is also a dm device, recursively resolve it
		if strings.HasPrefix(slave, "dm-") {
			resolvedSlaves, err := resolveDeviceMapperSlaves(slave)
			if err != nil {
				log.Warn().Err(err).Str("slave", slave).Msg("Failed to resolve slave")
				continue
			}
			devices = append(devices, resolvedSlaves...)
		} else {
			// This is a physical device
			devices = append(devices, "/dev/"+slave)
		}
	}

	return devices, nil
}

// Get device mapper minor number using proper unix.Major/Minor functions
func getMapperDeviceMinor(mapperDevice string) (int, error) {
	var stat syscall.Stat_t
	if err := syscall.Stat(mapperDevice, &stat); err != nil {
		return 0, fmt.Errorf("failed to stat %s: %w", mapperDevice, err)
	}

	major := int(unix.Major(uint64(stat.Rdev)))
	minor := int(unix.Minor(uint64(stat.Rdev)))

	matches, err := filepath.Glob("/sys/block/dm-*")
	if err != nil {
		return 0, err
	}

	for _, dmPath := range matches {
		devFile := filepath.Join(dmPath, "dev")
		devBytes, err := os.ReadFile(devFile)
		if err != nil {
			continue
		}

		parts := strings.Split(strings.TrimSpace(string(devBytes)), ":")
		if len(parts) != 2 {
			continue
		}

		sysMajor, _ := strconv.Atoi(parts[0])
		sysMinor, _ := strconv.Atoi(parts[1])

		if sysMajor == major && sysMinor == minor {
			dmName := filepath.Base(dmPath)
			return strconv.Atoi(strings.TrimPrefix(dmName, "dm-"))
		}
	}

	return 0, fmt.Errorf("could not find dm device for %s", mapperDevice)
}

func initOSDMappingCache(basePath string) error {
	if cacheInitialized {
		return nil
	}

	// Check if basePath exists
	if _, err := os.Stat(basePath); os.IsNotExist(err) {
		log.Debug().Str("base_path", basePath).Msg("Ceph OSD base path does not exist, skipping OSD mapping")
		cacheInitialized = true
		return nil
	}

	log.Info().Str("base_path", basePath).Msg("Initializing OSD mapping cache")

	pattern := filepath.Join(basePath, "*_*")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return fmt.Errorf("failed to glob pattern %s: %w", pattern, err)
	}

	if len(matches) == 0 {
		log.Debug().Str("pattern", pattern).Msg("No OSD directories found")
		cacheInitialized = true
		return nil
	}

	mappingCount := 0
	for _, dirPath := range matches {
		if stat, err := os.Stat(dirPath); err != nil || !stat.IsDir() {
			continue
		}

		blockPath := filepath.Join(dirPath, "block")
		if _, err := os.Stat(blockPath); err != nil {
			continue
		}

		blockDevice, err := filepath.EvalSymlinks(blockPath)
		if err != nil {
			continue
		}

		whoamiPath := filepath.Join(dirPath, "whoami")
		osdIDBytes, err := os.ReadFile(whoamiPath)
		if err != nil {
			continue
		}

		osdID := strings.TrimSpace(string(osdIDBytes))

		// Handle device mapper devices
		if strings.HasPrefix(blockDevice, "/dev/mapper/") {
			// Get the dm-* number for this mapper device
			minor, err := getMapperDeviceMinor(blockDevice)
			if err != nil {
				log.Warn().Err(err).Str("device", blockDevice).Msg("Failed to get mapper device minor")
				continue
			}

			dmDevice := fmt.Sprintf("dm-%d", minor)

			// Resolve the device mapper chain to physical devices
			physicalDevices, err := resolveDeviceMapperSlaves(dmDevice)
			if err != nil {
				log.Warn().Err(err).Str("dm_device", dmDevice).Msg("Failed to resolve device mapper chain")
				continue
			}

			// Map each physical device to this OSD ID (with normalization)
			for _, physicalDevice := range physicalDevices {
				normalizedDevice := normalizeDevicePath(physicalDevice)

				// Store both original and normalized paths to be safe
				physicalDeviceToOSDCache[physicalDevice] = osdID
				if normalizedDevice != physicalDevice {
					physicalDeviceToOSDCache[normalizedDevice] = osdID
				}

				mappingCount++
				log.Debug().Str("physical_device", physicalDevice).Str("osd_id", osdID).Msg("Mapped physical device to OSD ID")
			}
		} else {
			// Direct device mapping (with normalization)
			normalizedDevice := normalizeDevicePath(blockDevice)

			// Store both original and normalized paths to be safe
			physicalDeviceToOSDCache[blockDevice] = osdID
			if normalizedDevice != blockDevice {
				physicalDeviceToOSDCache[normalizedDevice] = osdID
			}

			mappingCount++
			log.Debug().Str("device", blockDevice).Str("osd_id", osdID).Msg("Mapped direct device to OSD ID")
		}
	}

	cacheInitialized = true
	log.Info().Int("total_mappings", mappingCount).Msg("OSD mapping cache initialized")

	return nil
}
