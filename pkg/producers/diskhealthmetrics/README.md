# Disk Health Metrics (local producer)

## Overview

The **Disk Health Metrics (Prysm Local Producer)** is a tool designed to monitor
the health of hardware devices by collecting and normalizing data from disk
drives using `smartctl`. This tool helps ensure that disks within your
infrastructure are operating optimally by providing real-time metrics and alerts
based on SMART attributes. It supports multiple output formats, including
Prometheus metrics for monitoring dashboards and NATS subjects for alerting.

## Key Features

- **SMART Attribute Normalization**: Automatically normalizes SMART attribute
  data to account for inconsistencies across different manufacturers and models.
  This ensures accurate interpretation and consistent data representation across
  various drives in your environment.
- **Device Info Normalization**: Standardizes device information such as product
  name, capacity, vendor, and media type to maintain consistency across
  different systems and databases.
- **Prometheus Metrics**: Exposes disk health metrics for Prometheus, including
  temperature, reallocated sectors, pending sectors, power-on hours, SSD life
  used percentage, and error counts.
- **NATS Alerts**: Sends alerts to NATS subjects when certain thresholds for
  disk health attributes are exceeded, such as grown defects, reallocated
  sectors, pending sectors, and SSD lifetime usage.
- **Flexible Configuration**: Allows configuration through command-line flags or
  environment variables, providing flexibility in deployment and integration
  with existing monitoring setups.

## Metrics Exposed

- **smart_attributes**: Gauges various SMART attributes of the disk.
- **disk_temperature_celsius**: Monitors disk temperature in Celsius.
- **disk_reallocated_sectors**: Tracks the number of reallocated sectors.
- **disk_pending_sectors**: Monitors the number of pending sectors.
- **disk_power_on_hours**: Reports the number of hours the disk has been powered
  on.
- **ssd_life_used_percentage**: Indicates the percentage of SSD life used.
- **disk_error_counts**: Tracks various error counts for the disk.
- **disk_capacity_gb**: Reports the capacity of the disk in GB.

## Alerts and Thresholds

The local producer can generate alerts based on specific thresholds for key
SMART attributes:

- **Grown Defects**: Triggers a warning if the number of grown defects exceeds
  the configured threshold.
- **Pending Sectors**: Triggers a warning if the number of pending sectors
  exceeds the configured threshold.
- **Reallocated Sectors**: Triggers a warning if the number of reallocated
  sectors exceeds the configured threshold.
- **SSD Lifetime Used**: Triggers a critical alert if the SSD lifetime used
  percentage exceeds the configured threshold.

## Usage

To run the Prysm local producer for disk health metrics, use the following
command:

```bash
prysm local-producer disk-health-metrics [flags]
```

### Example Flags:

- `--disks "/dev/sda,/dev/sdb"`: Comma-separated list of disks to monitor.
- `--prometheus`: Enable Prometheus metrics.
- `--prometheus-port 8080`: Port for Prometheus metrics (default is 8080).
- `--nats-url "nats://localhost:4222"`: NATS server URL for publishing alerts.
- `--grown-defects-threshold 10`: Threshold for grown defects to trigger a
  warning.
- `--pending-sectors-threshold 3`: Threshold for pending sectors to trigger a
  warning.
- `--reallocated-sectors-threshold 10`: Threshold for reallocated sectors to
  trigger a warning.
- `--lifetime-used-threshold 80`: Threshold for SSD lifetime used percentage to
  trigger a critical alert.

### Environment Variables

Configuration can also be set through environment variables:

- `NATS_URL`: NATS server URL.
- `DISKS`: Comma-separated list of disks to monitor.
- `PROMETHEUS_PORT`: Port for Prometheus metrics.
- `GROWN_DEFECTS_THRESHOLD`: Threshold for grown defects.
- `PENDING_SECTORS_THRESHOLD`: Threshold for pending sectors.
- `REALLOCATED_SECTORS_THRESHOLD`: Threshold for reallocated sectors.
- `LIFETIME_USED_THRESHOLD`: Threshold for SSD lifetime used percentage.

## Acknowledgment

The development of the local-producer disk-health-metrics was greatly supported
by insights and guidance from **Anthony D'Atri** and **Mohit Rajain**. Their
expertise on SMART attributes and practical advice on implementation were
invaluable in bringing this component to life.

---

> This README is a draft and will be updated as the project continues to evolve.
> Contributions and feedback are welcome to help refine and enhance the
> functionality of Prysm.
