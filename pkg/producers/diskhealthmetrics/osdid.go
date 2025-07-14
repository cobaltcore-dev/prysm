package diskhealthmetrics

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/rs/zerolog/log"
)

// Cache for OSD mappings: physical device -> OSD ID
var physicalDeviceToOSDCache = make(map[string]string)
var cacheInitialized = false

func getOSDIDForDisk(disk, basePath string) (string, error) {
	// Initialize the cache if not done yet
	if err := initOSDMappingCache(basePath); err != nil {
		return "", err
	}

	// Try direct lookup first
	if osdID, found := physicalDeviceToOSDCache[disk]; found {
		log.Debug().Str("disk", disk).Str("osd_id", osdID).Msg("Found OSD ID for disk")
		return osdID, nil
	}

	// Try with canonical path
	if canonical, err := filepath.EvalSymlinks(disk); err == nil {
		if osdID, found := physicalDeviceToOSDCache[canonical]; found {
			log.Debug().Str("disk", disk).Str("canonical", canonical).Str("osd_id", osdID).Msg("Found OSD ID for disk via canonical path")
			return osdID, nil
		}
	}

	log.Debug().Str("disk", disk).Msg("No OSD ID found for disk")
	return "", nil
}

// resolveDeviceMapperSlaves recursively resolves dm-* devices to physical devices
func resolveDeviceMapperSlaves(dev string) ([]string, error) {
	slavesPath := filepath.Join("/sys/block", dev, "slaves")

	// Check if slaves directory exists
	if _, err := os.Stat(slavesPath); os.IsNotExist(err) {
		// No slaves directory means this is a leaf device
		return []string{"/dev/" + dev}, nil
	}

	entries, err := os.ReadDir(slavesPath)
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
		resolvedSlaves, err := resolveDeviceMapperSlaves(slave)
		if err != nil {
			log.Warn().Err(err).Str("slave", slave).Msg("Failed to resolve slave")
			continue
		}
		devices = append(devices, resolvedSlaves...)
	}

	return devices, nil
}

// getDeviceMinorNumber extracts the minor number from a device path
func getDeviceMinorNumber(devicePath string) (int, error) {
	var stat syscall.Stat_t
	err := syscall.Stat(devicePath, &stat)
	if err != nil {
		return 0, err
	}

	minor := int(stat.Rdev & 0xff)
	return minor, nil
}

func initOSDMappingCache(basePath string) error {
	if cacheInitialized {
		return nil
	}

	log.Info().Str("base_path", basePath).Msg("Initializing OSD mapping cache")

	// Use filepath.Glob to match the UUID_UUID pattern directories
	pattern := filepath.Join(basePath, "*_*")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return fmt.Errorf("failed to glob pattern %s: %w", pattern, err)
	}

	for _, dirPath := range matches {
		// Check if this is actually a directory
		if stat, err := os.Stat(dirPath); err != nil || !stat.IsDir() {
			continue
		}

		// Check if this directory contains a block file/symlink
		blockPath := filepath.Join(dirPath, "block")
		if _, err := os.Stat(blockPath); err != nil {
			continue // Skip if no block file/symlink
		}

		// Resolve the block symlink to get the actual device
		blockDevice, err := filepath.EvalSymlinks(blockPath)
		if err != nil {
			continue // Skip if can't resolve symlink
		}

		// Read the whoami file to get the OSD ID
		whoamiPath := filepath.Join(dirPath, "whoami")
		osdIDBytes, err := os.ReadFile(whoamiPath)
		if err != nil {
			continue
		}

		osdID := strings.TrimSpace(string(osdIDBytes))

		log.Debug().Str("block_device", blockDevice).Str("osd_id", osdID).Msg("Processing OSD")

		// If this is a mapper device, resolve it to physical devices
		if strings.HasPrefix(blockDevice, "/dev/mapper/") {
			// Get the minor number to find the corresponding dm-* device
			minor, err := getDeviceMinorNumber(blockDevice)
			if err != nil {
				log.Warn().Err(err).Str("device", blockDevice).Msg("Failed to get device minor number")
				continue
			}

			// Resolve dm-<minor> to physical devices
			dmDevice := fmt.Sprintf("dm-%d", minor)
			physicalDevices, err := resolveDeviceMapperSlaves(dmDevice)
			if err != nil {
				log.Warn().Err(err).Str("dm_device", dmDevice).Msg("Failed to resolve device mapper slaves")
				continue
			}

			// Map each physical device to this OSD ID
			for _, physicalDevice := range physicalDevices {
				physicalDeviceToOSDCache[physicalDevice] = osdID
				log.Info().Str("physical_device", physicalDevice).Str("osd_id", osdID).Msg("Mapped physical device to OSD ID")
			}
		} else {
			// Direct mapping for non-mapper devices
			physicalDeviceToOSDCache[blockDevice] = osdID
			log.Info().Str("device", blockDevice).Str("osd_id", osdID).Msg("Mapped device to OSD ID")
		}
	}

	cacheInitialized = true
	log.Info().Int("mappings", len(physicalDeviceToOSDCache)).Msg("OSD mapping cache initialized")
	return nil
}
