// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and prysm contributors
//
// SPDX-License-Identifier: Apache-2.0

package diskhealthmetrics

import (
	"fmt"
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// enhanceDeviceInfo adds additional context about OEM relationships
func enhanceDeviceInfo(deviceInfo *DeviceInfo) {
	if deviceInfo == nil {
		return
	}

	// Common OEM relationships and rebranding patterns
	oem := detectOEMRelationship(deviceInfo.Vendor, deviceInfo.DeviceModel, deviceInfo.Product)
	if oem != "" {
		// You could add this to a new field in DeviceInfo or include it in existing fields
		if deviceInfo.ModelFamily == "" {
			deviceInfo.ModelFamily = oem
		}
	}
}

// detectOEMRelationship detects common OEM relationships
func detectOEMRelationship(vendor, model, product string) string {
	vendor = strings.ToLower(vendor)
	model = strings.ToLower(model)
	product = strings.ToLower(product)

	// Create a title caser for English
	caser := cases.Title(language.English)

	// Common OEM patterns - check all three fields for comprehensive detection

	// Lenovo OEM patterns
	if strings.Contains(vendor, "lenovo") {
		if strings.Contains(model, "toshiba") || strings.Contains(product, "toshiba") {
			return "Lenovo (Toshiba OEM)"
		}
		if strings.Contains(model, "seagate") || strings.Contains(product, "seagate") {
			return "Lenovo (Seagate OEM)"
		}
		if strings.Contains(model, "hgst") || strings.Contains(product, "hgst") {
			return "Lenovo (HGST OEM)"
		}
	}

	// Dell OEM patterns
	if strings.Contains(vendor, "dell") {
		if strings.Contains(model, "seagate") || strings.Contains(product, "seagate") {
			return "Dell (Seagate OEM)"
		}
		if strings.Contains(model, "western digital") || strings.Contains(product, "western digital") || strings.Contains(product, "wd") {
			return "Dell (WD OEM)"
		}
		if strings.Contains(model, "toshiba") || strings.Contains(product, "toshiba") {
			return "Dell (Toshiba OEM)"
		}
	}

	// HP/HPE OEM patterns
	if strings.Contains(vendor, "hp") || strings.Contains(vendor, "hpe") {
		if strings.Contains(model, "western digital") || strings.Contains(product, "western digital") || strings.Contains(product, "wd") {
			return "HP (WD OEM)"
		}
		if strings.Contains(model, "seagate") || strings.Contains(product, "seagate") {
			return "HP (Seagate OEM)"
		}
		if strings.Contains(model, "toshiba") || strings.Contains(product, "toshiba") {
			return "HP (Toshiba OEM)"
		}
	}

	// Supermicro OEM patterns
	if strings.Contains(vendor, "supermicro") {
		if strings.Contains(model, "intel") || strings.Contains(product, "intel") {
			return "Supermicro (Intel OEM)"
		}
		if strings.Contains(model, "samsung") || strings.Contains(product, "samsung") {
			return "Supermicro (Samsung OEM)"
		}
	}

	// Generic patterns - sometimes the product field contains the actual manufacturer
	if strings.Contains(product, "seagate") && !strings.Contains(vendor, "seagate") {
		return fmt.Sprintf("%s (Seagate OEM)", caser.String(vendor))
	}
	if strings.Contains(product, "western digital") || strings.Contains(product, "wd") && !strings.Contains(vendor, "western digital") {
		return fmt.Sprintf("%s (WD OEM)", caser.String(vendor))
	}
	if strings.Contains(product, "toshiba") && !strings.Contains(vendor, "toshiba") {
		return fmt.Sprintf("%s (Toshiba OEM)", caser.String(vendor))
	}
	if strings.Contains(product, "hgst") && !strings.Contains(vendor, "hgst") {
		return fmt.Sprintf("%s (HGST OEM)", caser.String(vendor))
	}
	if strings.Contains(product, "samsung") && !strings.Contains(vendor, "samsung") {
		return fmt.Sprintf("%s (Samsung OEM)", caser.String(vendor))
	}
	if strings.Contains(product, "intel") && !strings.Contains(vendor, "intel") {
		return fmt.Sprintf("%s (Intel OEM)", caser.String(vendor))
	}

	return ""
}
