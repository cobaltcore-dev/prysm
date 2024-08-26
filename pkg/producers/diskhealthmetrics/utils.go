// Copyright (c) 2024 Clyso GmbH
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program. If not, see <https://www.gnu.org/licenses/>.

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
