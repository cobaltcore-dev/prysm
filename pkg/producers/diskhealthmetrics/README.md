# Disk Health Metrics (local producer)

## Overview

The **Disk Health Metrics (Prysm Local Producer)** is a tool designed to monitor the health of
hardware devices by collecting and normalizing data from disk drives using `smartctl`. This tool
helps ensure that disks within your infrastructure are operating optimally by providing real-time
metrics and alerts based on SMART attributes. It supports multiple output formats, including
Prometheus metrics for monitoring dashboards and NATS subjects for alerting.

## Key Features

- **SMART Attribute Normalization**: Automatically normalizes SMART attribute data to account for
  inconsistencies across different manufacturers and models. This ensures accurate interpretation
  and consistent data representation across various drives in your environment.
- **Device Info Normalization**: Standardizes device information such as product name, capacity,
  vendor, and media type to maintain consistency across different systems and databases.
- **Ceph OSD Integration**: Automatically maps physical disk devices to their corresponding Ceph OSD
  numbers, providing enhanced observability for Ceph storage clusters. Works seamlessly with both
  direct block devices and LVM logical volumes.
- **Prometheus Metrics**: Exposes disk health metrics for Prometheus, including temperature,
  reallocated sectors, pending sectors, power-on hours, SSD life used percentage, and error counts.
  All metrics include an `osd_id` label when Ceph integration is enabled.
- **NATS Alerts**: Sends alerts to NATS subjects when certain thresholds for disk health attributes
  are exceeded, such as grown defects, reallocated sectors, pending sectors, and SSD lifetime usage.
- **Flexible Configuration**: Allows configuration through command-line flags or environment
  variables, providing flexibility in deployment and integration with existing monitoring setups.

## Ceph OSD Integration

The tool provides seamless integration with Ceph storage clusters by automatically mapping physical
disk devices to their corresponding OSD numbers. This feature works in both deployment scenarios:

### Direct Block Devices
When Ceph OSDs use direct block devices (e.g., `/dev/sda`), the mapping is straightforward:
- Physical device `/dev/sda` → OSD ID `osd.1`

### LVM Logical Volumes
When Ceph OSDs use LVM logical volumes (e.g., `/dev/mapper/xyz`), the tool automatically resolves
the device mapper dependencies to find the underlying physical devices:
- Physical device `/dev/sda` → LVM volume `/dev/mapper/xyz` → OSD ID `osd.1`

### Configuration
Enable Ceph integration by setting the `--ceph-osd-base-path` flag to your Ceph OSD base directory:
```bash
--ceph-osd-base-path "/var/lib/rook/rook-ceph/"
```

When enabled, all Prometheus metrics will include an `osd_id` label, allowing for enhanced
monitoring and alerting based on specific OSD performance and health metrics.

## Metrics Exposed

All metrics include standard labels (`disk`, `node`, `instance`) and an optional `osd_id` label when
Ceph integration is enabled:

- **smart_attributes**: Gauges various SMART attributes of the disk.
- **disk_temperature_celsius**: Monitors disk temperature in Celsius.
- **disk_reallocated_sectors**: Tracks the number of reallocated sectors.
- **disk_pending_sectors**: Monitors the number of pending sectors.
- **disk_power_on_hours_total**: Reports the cumulative number of hours the disk has been powered on.
- **ssd_life_used_percentage**: Indicates the percentage of SSD life used.
- **disk_error_counts_total**: Tracks various error counts for the disk.
- **disk_capacity_gb**: Reports the capacity of the disk in GB.

## Alerts and Thresholds

The local producer can generate alerts based on specific thresholds for key SMART attributes:

- **Grown Defects**: Triggers a warning if the number of grown defects exceeds the configured
  threshold.
- **Pending Sectors**: Triggers a warning if the number of pending sectors exceeds the configured
  threshold.
- **Reallocated Sectors**: Triggers a warning if the number of reallocated sectors exceeds the configured threshold.
- **SSD Lifetime Used**: Triggers a critical alert if the SSD lifetime used percentage exceeds the configured threshold.

## Usage

To run the Prysm local producer for disk health metrics, use the following command:

```bash
prysm local-producer disk-health-metrics [flags]
```

### Example Flags:

- `--nats-url "nats://localhost:4222"`: Specifies the NATS server URL for publishing alerts.
- `--nats-subject "osd.disk.health"`: Sets the NATS subject under which metrics are published.
- `--prometheus`: Enables Prometheus metrics.
- `--prometheus-port 8080`: Specifies the port for Prometheus metrics server (default is 8080).
- `--disks "/dev/sda,/dev/sdb"`: Comma-separated list of disks to monitor. Use "*" to monitor all available disks.
- `--interval 10`: Sets the interval in seconds between metric collections.
- `--grown-defects-threshold 10`: Threshold for grown defects to trigger a warning.
- `--pending-sectors-threshold 3`: Threshold for pending sectors to trigger a warning.
- `--reallocated-sectors-threshold 10`: Threshold for reallocated sectors to trigger a warning.
- `--lifetime-used-threshold 80`: Threshold for SSD lifetime used percentage to trigger a critical alert.
- `--ceph-osd-base-path "/var/lib/rook/rook-ceph/"`: Base path for mapping devices to Ceph OSD numbers.

### Environment Variables

Configuration can also be set through environment variables:

- `NATS_URL`: Overrides the NATS server URL.
- `NATS_SUBJECT`: Overrides the NATS subject for publishing metrics.
- `PROMETHEUS_PORT`: Overrides the port for the Prometheus metrics server.
- `DISKS`: Overrides the comma-separated list of disks to monitor.
- `INTERVAL`: Overrides the interval between metric collections.
- `GROWN_DEFECTS_THRESHOLD`: Overrides the threshold for grown defects.
- `PENDING_SECTORS_THRESHOLD`: Overrides the threshold for pending sectors.
- `REALLOCATED_SECTORS_THRESHOLD`: Overrides the threshold for reallocated sectors.
- `LIFETIME_USED_THRESHOLD`: Overrides the threshold for SSD lifetime used percentage.
- `CEPH_OSD_BASE_PATH`: Overrides the base path for mapping devices to Ceph OSD numbers.

## Deployment Example

For Kubernetes/Rook deployments, mount the Ceph OSD base path as a volume:

```yaml
volumeMounts:
  - name: host-rook-ceph
    mountPath: /var/lib/rook/rook-ceph
    readOnly: true
volumes:
  - name: host-rook-ceph
    hostPath:
      path: /var/lib/rook/rook-ceph
      type: Directory
```

## Acknowledgment

The development of the local-producer disk-health-metrics was greatly supported by insights and
guidance from **Anthony D'Atri** and **Mohit Rajain**. Their expertise on SMART attributes and
practical advice on implementation were invaluable in bringing this component to life.

---

> This README is a draft and will be updated as the project continues to evolve. Contributions and
> feedback are welcome to help refine and enhance the functionality of Prysm.
