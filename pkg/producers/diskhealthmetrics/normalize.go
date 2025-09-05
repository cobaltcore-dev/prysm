// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and prysm contributors
//
// SPDX-License-Identifier: Apache-2.0

package diskhealthmetrics

import (
	"fmt"
	"strings"

	"github.com/rs/zerolog/log"
)

// FillDeviceInfoFromSmartData populates the DeviceInfo struct based on the raw SMART data.
// This function handles the initial extraction of device details such as model, vendor,
// capacity, form factor, and other relevant attributes depending on the device protocol (ATA, SCSI, NVMe).
func FillDeviceInfoFromSmartData(deviceInfo *DeviceInfo, smartData *SmartCtlOutput) {
	// Fill in the device model and serial number, which are common across protocols.
	deviceInfo.DeviceModel = smartData.DeviceModel
	deviceInfo.SerialNumber = smartData.SerialNumber
	deviceInfo.FirmwareVersion = smartData.FirmwareVersion

	// Handle vendor and product details based on protocol.
	switch smartData.Device.Protocol {
	case "ATA":
		// ATA-specific processing
		deviceInfo.ModelFamily = smartData.ModelFamily
		deviceInfo.Vendor = "ATA" // ATA devices might not have a specific vendor field
		deviceInfo.Product = smartData.DeviceModel
		deviceInfo.Media = "hdd" // Default to hdd for ATA, unless otherwise determined

		// RPM and form factor, relevant for HDDs
		if smartData.RotationRate > 0 {
			deviceInfo.RPM = smartData.RotationRate
		}
		if smartData.FormFactor != nil {
			deviceInfo.FormFactor = smartData.FormFactor.Name
		}

	case "SCSI":
		// SCSI-specific processing
		deviceInfo.DeviceModel = smartData.SCSIModelName
		deviceInfo.Vendor = smartData.SCSIVendor
		deviceInfo.Product = smartData.SCSIProduct
		deviceInfo.LunID = smartData.LogicalUnitID
		// Set media type (if not explicitly provided by the output, assume it's an HDD)
		deviceInfo.Media = "hdd"
		if smartData.SmartSupport.Available && strings.Contains(strings.ToLower(smartData.Device.Type), "ssd") {
			deviceInfo.Media = "ssd"
		}

		// Capacity for SCSI
		if smartData.UserCapacity != nil {
			deviceInfo.Capacity = float64(smartData.UserCapacity.Bytes) / (1024 * 1024 * 1024)
		}

		// RPM and form factor, specific to SCSI drives
		if smartData.RotationRate > 0 {
			deviceInfo.RPM = smartData.RotationRate
		}
		if smartData.FormFactor != nil {
			deviceInfo.FormFactor = smartData.FormFactor.Name
		}

	case "NVMe":
		// NVMe-specific processing
		// Use the ID and SubsystemID to represent the vendor, or leave blank if unavailable
		if smartData.NVMePCIVendor != nil {
			deviceInfo.Vendor = fmt.Sprintf("Vendor ID: 0x%04X, Subsystem ID: 0x%04X", smartData.NVMePCIVendor.ID, smartData.NVMePCIVendor.SubsystemID)
			deviceInfo.VendorID = fmt.Sprintf("0x%04X", smartData.NVMePCIVendor.ID)
			deviceInfo.SubsystemVendorID = fmt.Sprintf("0x%04X", smartData.NVMePCIVendor.SubsystemID)
		}
		deviceInfo.Product = smartData.DeviceModel
		deviceInfo.Media = "nvme"

		// Capacity for NVMe
		if smartData.NVMeTotalCapacity > 0 {
			// Use NVMeTotalCapacity if available
			deviceInfo.Capacity = float64(smartData.NVMeTotalCapacity) / (1024 * 1024 * 1024) // Convert to GiB
		} else if smartData.UserCapacity != nil {
			// Fallback to UserCapacity if NVMeTotalCapacity is not available
			deviceInfo.Capacity = float64(smartData.UserCapacity.Bytes) / (1024 * 1024 * 1024) // Convert to GiB
		}

		// DWPD (Drive Writes Per Day) if available
		if smartData.NVMeSmartHealthInfoLog != nil {
			deviceInfo.DWPD = float64(smartData.NVMeSmartHealthInfoLog.PercentageUsed)
		}

		// NVMe devices typically don’t have RPM, so leave it as 0
		deviceInfo.RPM = 0
	}

	// Set health status based on smart status
	if smartData.SmartStatus.Passed {
		deviceInfo.HealthStatus = true
	} else {
		deviceInfo.HealthStatus = false
	}
}

// NormalizeDeviceInfo updates the DeviceInfo struct based on the device model.
// This function addresses the variability in vendor implementation and the inconsistencies
// found in smartmontools' drivedb.h. Given that the same drive can be labeled differently
// across various systems and databases, this exhaustive mapping ensures that the
// device information is normalized across the entire fleet. This normalization is crucial
// for accurate querying and consistent data representation, especially when dealing
// with large, heterogeneous storage environments.
//
// Note: The device model list includes entries from various manufacturers like Hitachi,
// HGST, IBM, and Western Digital, among others. The function standardizes attributes
// such as product name, capacity (in GB), vendor, media type, form factor, DWPD (Drive
// Writes Per Day), and RPM (for HDDs).
func NormalizeDeviceInfo(deviceInfo *DeviceInfo) {
	switch deviceInfo.DeviceModel {
	case "INTEL SSDSC2BX200G4R":
		deviceInfo.DeviceModel = "SSDSC2BX200G4R"
		deviceInfo.Product = "S3610"
		deviceInfo.Capacity = 200
		deviceInfo.Vendor = "Intel"
		deviceInfo.Media = "ssd"
		deviceInfo.FormFactor = "sff"
		deviceInfo.DWPD = 3.0

	case "*SSDSC2KG480G8R":
		deviceInfo.Product = "S4610"
		deviceInfo.Capacity = 480
		deviceInfo.Vendor = "Intel"
		deviceInfo.Media = "ssd"
		deviceInfo.FormFactor = "sff"
		deviceInfo.DWPD = 3.0

	case "SSDSC2KG240G8R":
		deviceInfo.Product = "S4610"
		deviceInfo.Capacity = 240
		deviceInfo.Vendor = "Intel"
		deviceInfo.Media = "ssd"
		deviceInfo.FormFactor = "sff"
		deviceInfo.DWPD = 3.0

	case "INTEL SSDSC2BB240G4":
		deviceInfo.DeviceModel = "SSDSC2BB240G4"
		deviceInfo.Product = "S3500"
		deviceInfo.Capacity = 240
		deviceInfo.Vendor = "Intel"
		deviceInfo.Media = "ssd"
		deviceInfo.FormFactor = "sff"
		deviceInfo.DWPD = 0.3

	case "*SSDSC2BB800G7":
		deviceInfo.DeviceModel = "SSDSC2BB800G7"
		deviceInfo.Product = "S3520"
		deviceInfo.Capacity = 800
		deviceInfo.Vendor = "Intel"
		deviceInfo.Media = "ssd"
		deviceInfo.FormFactor = "sff"
		deviceInfo.DWPD = 1.0

	case "Dell Express Flash NVMe P4610 1.6TB SFF":
		deviceInfo.DeviceModel = "P4610"
		deviceInfo.Product = "P4610-Dell"
		deviceInfo.Capacity = 1600
		deviceInfo.Vendor = "Intel"
		deviceInfo.Media = "ssd"
		deviceInfo.FormFactor = "u2"
		deviceInfo.DWPD = 3.0

	case "Dell Express Flash NVMe P4610 3.2TB SFF":
		deviceInfo.DeviceModel = "P4610"
		deviceInfo.Product = "P4610-Dell"
		deviceInfo.Capacity = 3200
		deviceInfo.Vendor = "Intel"
		deviceInfo.Media = "ssd"
		deviceInfo.FormFactor = "u2"
		deviceInfo.DWPD = 3.0

	case "Dell Express Flash NVMe P4600 3.2TB SFF":
		deviceInfo.DeviceModel = "P4600"
		deviceInfo.Product = "P4600-Dell"
		deviceInfo.Capacity = 3200
		deviceInfo.Vendor = "Intel"
		deviceInfo.Media = "ssd"
		deviceInfo.FormFactor = "u2"
		deviceInfo.DWPD = 3.0

	case "Dell Express Flash NVMe P4500 2.0TB*":
		deviceInfo.DeviceModel = "P4500"
		deviceInfo.Product = "P4500-Dell"
		deviceInfo.Capacity = 2000
		deviceInfo.Vendor = "Intel"
		deviceInfo.Media = "ssd"
		deviceInfo.FormFactor = "u2"
		deviceInfo.DWPD = 1.0

	case "Dell Ent NVMe P5600 MU U.2 3.2TB":
		deviceInfo.DeviceModel = "P5600"
		deviceInfo.Product = "P5600-Dell"
		deviceInfo.Capacity = 3200
		deviceInfo.Vendor = "Intel"
		deviceInfo.Media = "ssd"
		deviceInfo.FormFactor = "u2"
		deviceInfo.DWPD = 3.0

	case "Dell Ent NVMe P5600 MU U.2 1.6TB":
		deviceInfo.DeviceModel = "P5600"
		deviceInfo.Product = "P5600-Dell"
		deviceInfo.Capacity = 1600
		deviceInfo.Vendor = "Intel"
		deviceInfo.Media = "ssd"
		deviceInfo.FormFactor = "u2"
		deviceInfo.DWPD = 3.0

	case "*SSDSC2BB800G4":
		deviceInfo.DeviceModel = "S3500"
		deviceInfo.Product = "S3500"
		deviceInfo.Capacity = 800
		deviceInfo.Vendor = "Intel"
		deviceInfo.Media = "ssd"
		deviceInfo.FormFactor = "sff"
		deviceInfo.DWPD = 0.3

	case "INTEL SSDPE2KE016T8":
		deviceInfo.DeviceModel = "SSDPE2KE016T8"
		deviceInfo.Product = "P4610-Generic"
		deviceInfo.Capacity = 1600
		deviceInfo.Vendor = "Intel"
		deviceInfo.Media = "ssd"
		deviceInfo.FormFactor = "u2"
		deviceInfo.DWPD = 3.0

	case "INTEL SSDSC2KG019T8":
		deviceInfo.DeviceModel = "SSDSC2KG019T8"
		deviceInfo.Product = "S4610-Generic"
		deviceInfo.Capacity = 1600
		deviceInfo.Vendor = "Intel"
		deviceInfo.Media = "ssd"
		deviceInfo.FormFactor = "u2"
		deviceInfo.DWPD = 3.0

	case "INTEL SSDSC2BX800G4":
		deviceInfo.DeviceModel = "SSDSC2BX800G4"
		deviceInfo.Product = "S3610"
		deviceInfo.Capacity = 800
		deviceInfo.Vendor = "Intel"
		deviceInfo.Media = "ssd"
		deviceInfo.FormFactor = "sff"
		deviceInfo.DWPD = 3.0

	case "*SSDSC2BB160G4":
		deviceInfo.DeviceModel = "SSDSC2BB160G4"
		deviceInfo.Product = "S3500"
		deviceInfo.Capacity = 160
		deviceInfo.Vendor = "Intel"
		deviceInfo.Media = "ssd"
		deviceInfo.FormFactor = "sff"
		deviceInfo.DWPD = 0.3

	case "*SSDSC2BB240G6":
		deviceInfo.DeviceModel = "SSDSC2BB240G6"
		deviceInfo.Product = "S3510"
		deviceInfo.Capacity = 240
		deviceInfo.Vendor = "Intel"
		deviceInfo.Media = "ssd"
		deviceInfo.FormFactor = "sff"
		deviceInfo.DWPD = 0.3

	case "SSDSC2BB120G7R":
		deviceInfo.Product = "S3520"
		deviceInfo.Capacity = 120
		deviceInfo.Vendor = "Intel"
		deviceInfo.Media = "ssd"
		deviceInfo.FormFactor = "sff"
		deviceInfo.DWPD = 1.0

	case "SSDSC2KG240G7R":
		deviceInfo.Product = "S4600"
		deviceInfo.Capacity = 240
		deviceInfo.Vendor = "Intel"
		deviceInfo.Media = "ssd"
		deviceInfo.FormFactor = "sff"
		deviceInfo.DWPD = 1.0

	case "SSDSC2KG480GZR":
		deviceInfo.Product = "S4620"
		deviceInfo.Capacity = 480
		deviceInfo.Vendor = "Intel"
		deviceInfo.Media = "ssd"
		deviceInfo.FormFactor = "sff"
		deviceInfo.DWPD = 3.0

	case "SSDSC2KB240G8R":
		deviceInfo.Product = "S4510"
		deviceInfo.Capacity = 240
		deviceInfo.Vendor = "Intel"
		deviceInfo.Media = "ssd"
		deviceInfo.FormFactor = "sff"
		deviceInfo.DWPD = 2.0

	case "SSDSC2KB480G8R":
		deviceInfo.Product = "S4510"
		deviceInfo.Capacity = 480
		deviceInfo.Vendor = "Intel"
		deviceInfo.Media = "ssd"
		deviceInfo.FormFactor = "sff"
		deviceInfo.DWPD = 1.3

	case "INTEL SSDSA2CW120G3":
		deviceInfo.DeviceModel = "SSDSA2CW120G3"
		deviceInfo.Product = "320"
		deviceInfo.Capacity = 120
		deviceInfo.Vendor = "Intel"
		deviceInfo.Media = "ssd"
		deviceInfo.FormFactor = "sff"
		deviceInfo.DWPD = 1.0

	case "INTEL SSDSC2CW120A3":
		deviceInfo.DeviceModel = "SSDSC2CW120A3"
		deviceInfo.Product = "520"
		deviceInfo.Capacity = 120
		deviceInfo.Vendor = "Intel"
		deviceInfo.Media = "ssd"
		deviceInfo.FormFactor = "sff"
		deviceInfo.DWPD = 2.0

	case "INTEL SSDPE2KX020T7T":
		deviceInfo.DeviceModel = "SSDPE2KX020T7T"
		deviceInfo.Product = "S4500"
		deviceInfo.Capacity = 1920
		deviceInfo.Vendor = "Intel"
		deviceInfo.Media = "ssd"
		deviceInfo.FormFactor = "sff"
		deviceInfo.DWPD = 1.0

	case "INTEL SSDSC2KG240G8":
		deviceInfo.DeviceModel = "SSDSC2KG240G8"
		deviceInfo.Product = "S4610"
		deviceInfo.Capacity = 240
		deviceInfo.Vendor = "Intel"
		deviceInfo.Media = "ssd"
		deviceInfo.FormFactor = "sff"
		deviceInfo.DWPD = 3.0

	case "WDC WD8004FRYZ-01VAEB0":
		deviceInfo.DeviceModel = "WD8004FRYZ"
		deviceInfo.Product = "Gold"
		deviceInfo.Capacity = 8000
		deviceInfo.Vendor = "WesternDigital"
		deviceInfo.Media = "hdd"
		deviceInfo.FormFactor = "lff"
		deviceInfo.RPM = 7200

	case "HUS722T2TALA600":
		deviceInfo.Product = "Ultrastar7k2"
		deviceInfo.Capacity = 2000
		deviceInfo.Vendor = "WesternDigital"
		deviceInfo.Media = "hdd"
		deviceInfo.FormFactor = "lff"
		deviceInfo.RPM = 7200

	case "HUS728T8TAL5200":
		deviceInfo.Product = "UltrastarDC"
		deviceInfo.Capacity = 8000
		deviceInfo.Vendor = "WesternDigital"
		deviceInfo.Media = "hdd"
		deviceInfo.FormFactor = "lff"
		deviceInfo.RPM = 7200

	case "WDC WD2005FBYZ-01YCBB2":
		deviceInfo.DeviceModel = "WD2005FBYZ-01YCBB2"
		deviceInfo.Product = "Gold"
		deviceInfo.Capacity = 2000
		deviceInfo.Vendor = "WesternDigital"
		deviceInfo.Media = "hdd"
		deviceInfo.FormFactor = "lff"
		deviceInfo.RPM = 7200

	case "*HUS726040ALA614":
		deviceInfo.DeviceModel = "HUS726040ALA614"
		deviceInfo.Product = "Ultrastar7K6000"
		deviceInfo.Capacity = 4000
		deviceInfo.Vendor = "WesternDigital"
		deviceInfo.Media = "hdd"
		deviceInfo.FormFactor = "lff"
		deviceInfo.RPM = 7200

	case "WD1004FBYZ":
		deviceInfo.Product = "Re"
		deviceInfo.Capacity = 1000
		deviceInfo.Vendor = "WesternDigital"
		deviceInfo.Media = "hdd"
		deviceInfo.FormFactor = "lff"
		deviceInfo.RPM = 7200

	case "WDC WD10JFCX-68N6GN0":
		deviceInfo.DeviceModel = "WD10JFCX-68N6GN0"
		deviceInfo.Product = "RedPlus"
		deviceInfo.Capacity = 1000
		deviceInfo.Vendor = "WesternDigital"
		deviceInfo.Media = "hdd"
		deviceInfo.FormFactor = "sff"
		deviceInfo.RPM = 5400

	case "WDC WD8002FRYZ-01FF2B0":
		deviceInfo.Product = "Gold"
		deviceInfo.Capacity = 8000
		deviceInfo.Vendor = "WesternDigital"
		deviceInfo.Media = "hdd"
		deviceInfo.FormFactor = "lff"
		deviceInfo.RPM = 7200

	case "WDC WD121KRYZ-01W0RB0":
		deviceInfo.DeviceModel = "WD121KRYZ-01W0RB0"
		deviceInfo.Product = "Gold"
		deviceInfo.Capacity = 12000
		deviceInfo.Vendor = "WesternDigital"
		deviceInfo.Media = "hdd"
		deviceInfo.FormFactor = "lff"
		deviceInfo.RPM = 7200

	case "WDC WD101KRYZ-01JPDB1":
		deviceInfo.Product = "Gold"
		deviceInfo.Capacity = 10000
		deviceInfo.Vendor = "WesternDigital"
		deviceInfo.Media = "hdd"
		deviceInfo.FormFactor = "lff"
		deviceInfo.RPM = 7200

	case "WDC WD8003FRYZ-01JPDB1":
		deviceInfo.Product = "Gold"
		deviceInfo.Capacity = 8000
		deviceInfo.Vendor = "WesternDigital"
		deviceInfo.Media = "hdd"
		deviceInfo.FormFactor = "lff"
		deviceInfo.RPM = 7200

	case "WDC WD102KRYZ-01A5AB0":
		deviceInfo.DeviceModel = "WD102KRYZ-01A5AB0"
		deviceInfo.Product = "Gold"
		deviceInfo.Capacity = 10000
		deviceInfo.Vendor = "WesternDigital"
		deviceInfo.Media = "hdd"
		deviceInfo.FormFactor = "lff"
		deviceInfo.RPM = 7200

	case "WDC WD40EFRX-68WT0N0":
		deviceInfo.DeviceModel = "WD40EFRX-68WT0N0"
		deviceInfo.Product = "Red"
		deviceInfo.Capacity = 4000
		deviceInfo.Vendor = "WesternDigital"
		deviceInfo.Media = "hdd"
		deviceInfo.FormFactor = "lff"
		deviceInfo.RPM = 5400

	case "WDC WD60EFRX-68L0BN1":
		deviceInfo.DeviceModel = "WD60EFRX-68L0BN1"
		deviceInfo.Product = "RedPlus"
		deviceInfo.Capacity = 6000
		deviceInfo.Vendor = "WesternDigital"
		deviceInfo.Media = "hdd"
		deviceInfo.FormFactor = "lff"
		deviceInfo.RPM = 5400

	case "WDC WD60EFRX-68MYMN1":
		deviceInfo.DeviceModel = "WD60EFRX-68MYMN1"
		deviceInfo.Product = "RedPlus"
		deviceInfo.Capacity = 6000
		deviceInfo.Vendor = "WesternDigital"
		deviceInfo.Media = "hdd"
		deviceInfo.FormFactor = "lff"
		deviceInfo.RPM = 5400

	case "HUS722T1TALA600":
		deviceInfo.Product = "Ultrastar7k2"
		deviceInfo.Capacity = 6000
		deviceInfo.Vendor = "WesternDigital"
		deviceInfo.Media = "hdd"
		deviceInfo.FormFactor = "lff"
		deviceInfo.RPM = 7200

	case "HUH721010AL5200":
		deviceInfo.Product = "UltrastarHe10"
		deviceInfo.Capacity = 10000
		deviceInfo.Vendor = "WesternDigital"
		deviceInfo.Media = "hdd"
		deviceInfo.FormFactor = "lff"
		deviceInfo.RPM = 7200

	case "HGST HUS722T1TALA604":
		deviceInfo.DeviceModel = "HUS722T1TALA604"
		deviceInfo.Product = "Ultrastar7K2"
		deviceInfo.Capacity = 1000
		deviceInfo.Vendor = "WesternDigital"
		deviceInfo.Media = "hdd"
		deviceInfo.FormFactor = "lff"
		deviceInfo.RPM = 7200

	case "HGST HUS726060ALE610":
		deviceInfo.DeviceModel = "HUS726060ALE610"
		deviceInfo.Product = "Ultrastar7k6"
		deviceInfo.Capacity = 6000
		deviceInfo.Vendor = "WesternDigital"
		deviceInfo.Media = "hdd"
		deviceInfo.FormFactor = "lff"
		deviceInfo.RPM = 7200

	case "*HUS726T4TALA6L0":
		deviceInfo.DeviceModel = "HUS726T4TALA6L0"
		deviceInfo.Product = "UltrastarHC310"
		deviceInfo.Capacity = 4000
		deviceInfo.Vendor = "WesternDigital"
		deviceInfo.Media = "hdd"
		deviceInfo.FormFactor = "lff"
		deviceInfo.RPM = 7200

	case "WDC WD5000BHTZ-04JCPV1":
		deviceInfo.DeviceModel = "WD5000BHTZ-04JCPV1"
		deviceInfo.Product = "VelociRaptor"
		deviceInfo.Capacity = 500
		deviceInfo.Vendor = "WesternDigital"
		deviceInfo.Media = "hdd"
		deviceInfo.FormFactor = "sff"
		deviceInfo.RPM = 10000

	case "WDC WD20EFRX-68EUZN0":
		deviceInfo.DeviceModel = "WD20EFRX-68EUZN0"
		deviceInfo.Product = "RedPlus"
		deviceInfo.Capacity = 2000
		deviceInfo.Vendor = "WesternDigital"
		deviceInfo.Media = "hdd"
		deviceInfo.FormFactor = "lff"
		deviceInfo.RPM = 5400

	case "WDC WD5000BHTZ-04JCPV0":
		deviceInfo.DeviceModel = "WD5000BHTZ-04JCPV0"
		deviceInfo.Product = "VelociRaptor"
		deviceInfo.Capacity = 500
		deviceInfo.Vendor = "WesternDigital"
		deviceInfo.Media = "hdd"
		deviceInfo.FormFactor = "sff"
		deviceInfo.RPM = 10000

	case "ST8000NM014A":
		deviceInfo.DeviceModel = "ST8000NM014A"
		deviceInfo.Product = "Exos7E10"
		deviceInfo.Capacity = 8000
		deviceInfo.Vendor = "Seagate"
		deviceInfo.Media = "hdd"
		deviceInfo.FormFactor = "lff"
		deviceInfo.RPM = 7200

	case "ST1000NM0055-1V410C":
		deviceInfo.Product = "Exos7E8"
		deviceInfo.Capacity = 1000
		deviceInfo.Vendor = "Seagate"
		deviceInfo.Media = "hdd"
		deviceInfo.FormFactor = "lff"
		deviceInfo.RPM = 7200

	case "ST300MP0026":
		deviceInfo.Product = "EntPerf"
		deviceInfo.Capacity = 300
		deviceInfo.Vendor = "Seagate"
		deviceInfo.Media = "hdd"
		deviceInfo.FormFactor = "sff"
		deviceInfo.RPM = 15000

	case "ST2000NM0155":
		deviceInfo.Product = "Exos7E8"
		deviceInfo.Capacity = 2000
		deviceInfo.Vendor = "Seagate"
		deviceInfo.Media = "hdd"
		deviceInfo.FormFactor = "lff"
		deviceInfo.RPM = 7200

	case "ST1000NM0033-9ZM173":
		deviceInfo.Product = "ConstellationES.3"
		deviceInfo.Capacity = 1000
		deviceInfo.Vendor = "Seagate"
		deviceInfo.Media = "hdd"
		deviceInfo.FormFactor = "lff"
		deviceInfo.RPM = 7200

	case "ST2000NM012A-2MP130":
		deviceInfo.Product = "Exos7E8"
		deviceInfo.Capacity = 2000
		deviceInfo.Vendor = "Seagate"
		deviceInfo.Media = "hdd"
		deviceInfo.FormFactor = "lff"
		deviceInfo.RPM = 7200

	case "ST10000NM0096":
		deviceInfo.Product = "ExosX10"
		deviceInfo.Capacity = 10000
		deviceInfo.Vendor = "Seagate"
		deviceInfo.Media = "hdd"
		deviceInfo.FormFactor = "lff"
		deviceInfo.RPM = 7200

	case "ST1000NX0473":
		deviceInfo.Product = "Exos7E2000"
		deviceInfo.Capacity = 1000
		deviceInfo.Vendor = "Seagate"
		deviceInfo.Media = "hdd"
		deviceInfo.FormFactor = "sff"
		deviceInfo.RPM = 7200

	case "ST1000NX0443":
		deviceInfo.Product = "Exos7E2000"
		deviceInfo.Capacity = 1000
		deviceInfo.Vendor = "Seagate"
		deviceInfo.Media = "hdd"
		deviceInfo.FormFactor = "sff"
		deviceInfo.RPM = 7200

	case "ST2000NM013A":
		deviceInfo.Product = "Exos7E8"
		deviceInfo.Capacity = 2000
		deviceInfo.Vendor = "Seagate"
		deviceInfo.Media = "hdd"
		deviceInfo.FormFactor = "lff"
		deviceInfo.RPM = 7200

	case "DL2400MM0159":
		deviceInfo.Product = "Exos10E2400"
		deviceInfo.Capacity = 2400
		deviceInfo.Vendor = "Seagate"
		deviceInfo.Media = "hdd"
		deviceInfo.FormFactor = "sff"
		deviceInfo.RPM = 10000

	case "ST4000NM018B-2TF130":
		deviceInfo.Product = "Exos7E10"
		deviceInfo.Capacity = 4000
		deviceInfo.Vendor = "Seagate"
		deviceInfo.Media = "hdd"
		deviceInfo.FormFactor = "lff"
		deviceInfo.RPM = 7200

	case "TOSHIBA MG03ACA100":
		deviceInfo.DeviceModel = "MG03ACA100"
		deviceInfo.Product = "MG03"
		deviceInfo.Capacity = 3000
		deviceInfo.Vendor = "Toshiba"
		deviceInfo.Media = "hdd"
		deviceInfo.FormFactor = "lff"
		deviceInfo.RPM = 7200

	case "TOSHIBA MG04ACA200NY":
		deviceInfo.DeviceModel = "MG04ACA200NY"
		deviceInfo.Product = "MG04"
		deviceInfo.Capacity = 2000
		deviceInfo.Vendor = "Toshiba"
		deviceInfo.Media = "hdd"
		deviceInfo.FormFactor = "lff"
		deviceInfo.RPM = 7200

	case "TOSHIBA MG04ACA400N":
		deviceInfo.DeviceModel = "MG04ACA400N"
		deviceInfo.Product = "MG04"
		deviceInfo.Capacity = 4000
		deviceInfo.Vendor = "Toshiba"
		deviceInfo.Media = "hdd"
		deviceInfo.FormFactor = "lff"
		deviceInfo.RPM = 7200

	case "TOSHIBA MG08ADA400NY":
		deviceInfo.DeviceModel = "MG08ADA400NY"
		deviceInfo.Product = "MG08-D"
		deviceInfo.Capacity = 4000
		deviceInfo.Vendor = "Toshiba"
		deviceInfo.Media = "hdd"
		deviceInfo.FormFactor = "lff"
		deviceInfo.RPM = 7200

	case "MG06SCA800EY":
		deviceInfo.Product = "MG06"
		deviceInfo.Capacity = 8000
		deviceInfo.Vendor = "Toshiba"
		deviceInfo.Media = "hdd"
		deviceInfo.FormFactor = "lff"
		deviceInfo.RPM = 7200

	case "MG04SCA20ENY":
		deviceInfo.Product = "MG04"
		deviceInfo.Capacity = 2000
		deviceInfo.Vendor = "Toshiba"
		deviceInfo.Media = "hdd"
		deviceInfo.FormFactor = "lff"
		deviceInfo.RPM = 7200

	case "TOSHIBA MG04ACA100NY":
		deviceInfo.DeviceModel = "MG04ACA100NY"
		deviceInfo.Product = "MGA04"
		deviceInfo.Capacity = 1000
		deviceInfo.Vendor = "Toshiba"
		deviceInfo.Media = "hdd"
		deviceInfo.FormFactor = "lff"
		deviceInfo.RPM = 7200

	case "HFS480G32FEH-BA10A":
		deviceInfo.Product = "HFS"
		deviceInfo.Capacity = 480
		deviceInfo.Vendor = "Hynix"
		deviceInfo.Media = "ssd"
		deviceInfo.FormFactor = "sff"
		deviceInfo.DWPD = 3.0

	case "MZ7LH480HBHQ0D3":
		deviceInfo.Product = "PM883a"
		deviceInfo.Capacity = 480
		deviceInfo.Vendor = "Samsung"
		deviceInfo.Media = "ssd"
		deviceInfo.FormFactor = "sff"
		deviceInfo.DWPD = 3.6

	case "MZ7KH480HAHQ0D3":
		deviceInfo.Product = "SM883"
		deviceInfo.Capacity = 480
		deviceInfo.Vendor = "Samsung"
		deviceInfo.Media = "ssd"
		deviceInfo.FormFactor = "sff"
		deviceInfo.DWPD = 3.0

	case "MTFDDAV240TDU":
		deviceInfo.Product = "5300"
		deviceInfo.Capacity = 240
		deviceInfo.Vendor = "Micron"
		deviceInfo.Media = "ssd"
		deviceInfo.FormFactor = "sff"
		deviceInfo.DWPD = 1.0

	case "MTFDDAK960TDN":
		deviceInfo.Product = "5200MAX"
		deviceInfo.Capacity = 960
		deviceInfo.Vendor = "Micron"
		deviceInfo.Media = "ssd"
		deviceInfo.FormFactor = "sff"
		deviceInfo.DWPD = 5.0

	case "MTFDDAK480TDC":
		deviceInfo.Product = "5200ECO"
		deviceInfo.Capacity = 480
		deviceInfo.Vendor = "Micron"
		deviceInfo.Media = "ssd"
		deviceInfo.FormFactor = "sff"
		deviceInfo.DWPD = 0.8
	}
}

// NormalizeVendor determines the vendor based on the device model or model family
// This function updates the Vendor field in the DeviceInfo struct based on known patterns.
func NormalizeVendor(deviceInfo *DeviceInfo) {
	// If the vendor is already populated, we skip further normalization.
	if deviceInfo.Vendor != "" {
		return
	}

	// Normalize the device model and family to lowercase for comparison.
	model := strings.ToLower(deviceInfo.DeviceModel)
	family := strings.ToLower(deviceInfo.ModelFamily)

	// Match common vendor patterns based on the device model or model family.
	switch {
	case strings.Contains(model, "dl2400") || strings.Contains(family, "dl2400"):
		deviceInfo.Vendor = "Seagate"
	case strings.Contains(model, "toshiba") || strings.Contains(family, "toshiba"),
		strings.Contains(model, "mg0") || strings.Contains(family, "mg0"):
		deviceInfo.Vendor = "Toshiba"
	case strings.Contains(model, "intel") || strings.Contains(family, "intel"):
		deviceInfo.Vendor = "Intel"
	case strings.Contains(model, "kioxia") || strings.Contains(family, "kioxia"):
		deviceInfo.Vendor = "Kioxia"
	case strings.Contains(model, "western") || strings.Contains(family, "western"),
		strings.Contains(model, "wdc") || strings.Contains(family, "wdc"),
		strings.Contains(model, "wd100") || strings.Contains(family, "wd100"):
		deviceInfo.Vendor = "WesternDigital"
	case strings.Contains(model, "seagate") || strings.Contains(family, "seagate"),
		strings.Contains(model, "st12") || strings.Contains(family, "st12"):
		deviceInfo.Vendor = "Seagate"
	case strings.Contains(model, "hgst") || strings.Contains(family, "hgst"),
		strings.Contains(model, "huhs") || strings.Contains(family, "huhs"):
		deviceInfo.Vendor = "HGST"
	case strings.Contains(model, "micron") || strings.Contains(family, "micron"),
		strings.Contains(model, "mtfd") || strings.Contains(family, "mtfd"):
		deviceInfo.Vendor = "Micron"
	case strings.Contains(model, "sandisk") || strings.Contains(family, "sandisk"):
		deviceInfo.Vendor = "SanDisk"
	case strings.Contains(model, "samsung") || strings.Contains(family, "samsung"),
		strings.Contains(model, "mz7") || strings.Contains(family, "mz7"):
		deviceInfo.Vendor = "Samsung"
	}

	// If no vendor is detected, we can optionally log or leave it as an empty string.
	if deviceInfo.Vendor == "" {
		log.Warn().Str("device_model", deviceInfo.DeviceModel).Msg("Unknown vendor for device model")
	}
}
