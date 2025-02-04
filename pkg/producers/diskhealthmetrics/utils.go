// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and prysm contributors
//
// SPDX-License-Identifier: Apache-2.0

package diskhealthmetrics

import (
	"os"
	"regexp"
	"strings"

	"github.com/rs/zerolog/log"
)

// IsVirtualized checks if the system is running on a virtualized environment
func IsVirtualized() bool {
	sysVendor, err := os.ReadFile("/sys/devices/virtual/dmi/id/sys_vendor")
	if err != nil {
		log.Error().Err(err).Msg("error reading sys_vendor")
		return false
	}
	sysVendorStr := strings.TrimSpace(string(sysVendor))

	virtTech := []string{"VMware", "VirtualBox", "QEMU", "Xen", "KVM", "Microsoft Hyper-V", "Parallels", "Oracle VM Server"}

	for _, tech := range virtTech {
		if strings.Contains(sysVendorStr, tech) {
			return true
		}
	}
	return false
}

func FindVendor(deviceModel, modelFamily string) string {
	var patterns = []struct {
		pattern *regexp.Regexp
		vendor  string
	}{
		{regexp.MustCompile(`(?i)^DL2400`), "Seagate"},
		{regexp.MustCompile(`(?i)TOSHIBA`), "Toshiba"},
		{regexp.MustCompile(`(?i)^MG0[345678]`), "Toshiba"},
		{regexp.MustCompile(`(?i)INTEL`), "Intel"},
		{regexp.MustCompile(`(?i)KIOXIA`), "Kioxia"},
		{regexp.MustCompile(`(?i)WESTERN`), "WesternDigital"},
		{regexp.MustCompile(`(?i)WDC`), "WesternDigital"},
		{regexp.MustCompile(`(?i)^WD100`), "WesternDigital"},
		{regexp.MustCompile(`(?i)SEAGATE`), "Seagate"},
		{regexp.MustCompile(`(?i)^ST[12][0123456789]`), "Seagate"},
		{regexp.MustCompile(`(?i)HGST`), "HGST"},
		{regexp.MustCompile(`(?i)^HU[HS]`), "HGST"},
		{regexp.MustCompile(`(?i)MICRON`), "Micron"},
		{regexp.MustCompile(`(?i)MTFDD`), "Micron"},
		{regexp.MustCompile(`(?i)SANDISK`), "SanDisk"},
		{regexp.MustCompile(`(?i)SAMSUNG`), "Samsung"},
		{regexp.MustCompile(`(?i)^MZ7`), "Samsung"},
	}

	for _, entry := range patterns {
		if entry.pattern.MatchString(deviceModel) || entry.pattern.MatchString(modelFamily) {
			return entry.vendor
		}
	}
	return ""
}
