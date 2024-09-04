# Resource Usage Metrics (local producer)

## Overview

The **Resource Usage Metrics Collector (Prysm Local Producer)** is a tool
designed to monitor and report the resource usage of a node. This local producer
collects metrics related to CPU usage, memory usage, disk I/O, and network I/O.
The collected metrics can be published to a NATS server for real-time processing
or exposed via Prometheus for integration with monitoring dashboards.

## Key Features

- **Comprehensive Resource Monitoring**: Collects key metrics including CPU
  usage, memory usage, disk I/O, and network I/O.
- **NATS Integration**: Publishes collected metrics to a specified NATS subject
  for real-time monitoring and alerting.
- **Prometheus Metrics**: Exposes resource usage metrics in Prometheus format,
  allowing easy integration with monitoring systems.
- **Configurable**: Flexible configuration via command-line flags or environment
  variables.

## Usage

To run the Prysm resource usage metrics collector, use the following command:

```bash
prysm local-producer resource-usage [flags]
```

## Example Flags:

- `--disks "sda,sdb"`: Comma-separated list of disks to monitor (default is
  “sda,sdb”).
- `--nats-url "nats://localhost:4222"`: NATS server URL for publishing metrics.
- `--nats-subject "node.resource.usage"`: NATS subject to publish metrics
  (default is “node.resource.usage”).
- `--instance-id "instance-1"`: Instance ID for identifying the source of the
  metrics.
- `--node-name "node-1"`: Name of the node for identifying the source of the
  metrics.
- `--interval 10`: Interval in seconds between metric collections (default is 10
  seconds).
- `--prometheus`: Enable Prometheus metrics.
- `--prometheus-port 8080`: Port for Prometheus metrics (default is 8080).

## Environment Variables

Configuration can also be set through environment variables:

- `NATS_URL`: NATS server URL.
- `NATS_SUBJECT`: NATS subject to publish metrics.
- `PROMETHEUS_PORT`: Port for Prometheus metrics.
- `DISKS`: Comma-separated list of disks to monitor.
- `NODE_NAME`: Name of the node.
- `INSTANCE_ID`: Instance ID.
- `INTERVAL`: Interval in seconds between metric collections.

## Metrics Collected

The Resource Usage Metrics Collector gathers and exposes the following metrics:

- `node_cpu_usage_percent`: CPU usage percentage of the node.
- `node_memory_usage_percent`: Memory usage percentage of the node.
- `node_disk_io_bytes`: Disk I/O in bytes of the node.
- `node_network_io_bytes`: Network I/O in bytes of the node.

## Logic and Workflow

The Resource Usage Metrics Collector operates as follows:

Collect Resource Usage Data:

- The tool collects metrics related to CPU usage, memory usage, disk I/O, and
  network I/O from the node.
- Disk I/O is aggregated from the specified disks, and network I/O is measured
  for the entire node. Publish to NATS or Expose via Prometheus:
- If NATS integration is enabled, the collected metrics are published to the
  specified NATS subject.
- If Prometheus metrics are enabled, the metrics are exposed on the specified
  port for scraping by Prometheus. Repeat at Regular Intervals:
- The tool repeats the resource usage collection and publication process at
  regular intervals as specified by the --interval flag or the INTERVAL
  environment variable.

## Example Workflow

- Start the resource usage metrics collector with the desired configuration:

```bash
prysm local-producer resource-usage --nats-url "nats://localhost:4222" --prometheus --prometheus-port 8080
```

- The collector will monitor CPU usage, memory usage, disk I/O, and network I/O
  every 10 seconds (default) and can either publish these metrics to the
  node.resource.usage NATS subject or expose them via Prometheus.

---

> This README is a draft and will be updated as the project continues to evolve.
> Contributions and feedback are welcome to help refine and enhance the
> functionality of Prysm.
