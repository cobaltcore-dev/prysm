# Disk Health Metrics Test Data

This directory contains test data files for simulating disk health metrics without requiring actual hardware or smartctl access.

## Directory Structure

```
testdata/
└── scenarios/
    ├── healthy/     # All devices are healthy
    ├── failing/     # Devices with critical issues
    └── mixed/       # Mix of healthy and problematic devices
```

## Usage

### Basic Test Mode

Run the disk health metrics producer in test mode:

```bash
# Use default mixed scenario
prysm local-producer disk-health-metrics \
  --test-mode \
  --prometheus \
  --prometheus-port 8080

# Use specific scenario
prysm local-producer disk-health-metrics \
  --test-mode \
  --test-scenario healthy \
  --prometheus \
  --prometheus-port 8080

# Test specific devices only
prysm local-producer disk-health-metrics \
  --test-mode \
  --test-devices nvme0,sda \
  --prometheus \
  --prometheus-port 8080
```

### Environment Variables

```bash
export TEST_MODE=true
export TEST_SCENARIO=failing
export TEST_DEVICES=nvme0,nvme1
prysm local-producer disk-health-metrics --prometheus
```

### Custom Test Data

Create your own test scenarios:

1. Create a new directory under `scenarios/`
2. Add JSON files named after devices (e.g., `nvme0.json`, `sda.json`)
3. Use with `--test-scenario your-scenario-name`

## Test Scenarios

### healthy
- All devices report normal values
- No errors or warnings
- Good for testing normal dashboard appearance

### failing  
- Devices with critical issues:
  - High wear levels (>95%)
  - Critical warnings set
  - Media errors present
  - Temperature warnings
  - Low available spare

### mixed (default)
- Combination of device states:
  - nvme0: Moderate issues (75% wear, some errors)
  - nvme1: Healthy
  - sda: Healthy SSD
  - sdb: Healthy HDD

## Adding Test Data

To add new test data files:

1. Collect real smartctl output:
   ```bash
   smartctl --json --info --health --attributes \
     --tolerance=verypermissive --nocheck=standby \
     --format=brief --log=error /dev/nvme0 > nvme0.json
   ```

2. Modify values to simulate desired conditions:
   - `critical_warning`: 0-31 (bitfield)
   - `percentage_used`: 0-100 (SSD wear)
   - `media_errors`: Error count
   - `temperature`: Current temperature
   - `available_spare`: Percentage (0-100)

## Test Data Values Reference

### NVMe Critical Warning Bits
- Bit 0 (1): Available spare below threshold
- Bit 1 (2): Temperature above threshold  
- Bit 2 (4): Reliability degraded
- Bit 3 (8): Read-only mode
- Bit 4 (16): Volatile memory backup failed

Example: `critical_warning: 3` means bits 0 and 1 are set (spare low + temp high)

### Common Test Values

**Healthy Device:**
```json
"critical_warning": 0,
"percentage_used": 5,
"media_errors": 0,
"temperature": 35,
"available_spare": 100
```

**Wearing Device:**
```json
"critical_warning": 1,
"percentage_used": 85,
"media_errors": 10,
"temperature": 45,
"available_spare": 20
```

**Critical Device:**
```json
"critical_warning": 7,
"percentage_used": 98,
"media_errors": 500,
"temperature": 75,
"available_spare": 2
```