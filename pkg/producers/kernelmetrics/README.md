# Kernel Metrics (local producer)

## Overview

The **Kernel Metrics (Prysm Local Producer)** is a tool designed to collect and monitor kernel-related metrics from your nodes. This tool gathers critical metrics such as context switches, available entropy, and network connections, and can publish these metrics to a NATS server or expose them for Prometheus, providing real-time visibility into the kernel's performance and health.

## Key Features

- **Kernel Metrics Collection**: Monitors essential kernel metrics, including context switches, available entropy, and network connections.
- **NATS Integration**: Publishes collected metrics to a specified NATS subject, enabling seamless integration with other monitoring and observability tools.
- **Prometheus Metrics**: Exposes kernel metrics in Prometheus format, allowing easy integration with monitoring dashboards.
- **Configurable**: Offers flexibility in configuration via command-line flags or environment variables.

## Usage

To run the Prysm local producer for kernel metrics, use the following command:

```bash
prysm local-producer kernel-metrics [flags]
````

Example Flags:

-	`--instance-id "instance-1"`: Unique identifier for the instance being monitored.
-	`--interval 10`: Interval in seconds between metric collections (default is 10 seconds).
-	`--nats-url "nats://localhost:4222"`: NATS server URL for publishing metrics.
-	`--nats-subject "node.kernel.metrics"`: NATS subject to publish metrics (default is “node.kernel.metrics”).
-	`--node-name "node-1"`: Name of the node being monitored.
-	`--prometheus`: Enable Prometheus metrics.
-	`--prometheus-port 8080`: Port for Prometheus metrics (default is 8080).

Environment Variables

Configuration can also be set through environment variables:

-	`NATS_URL`: NATS server URL.
-	`NATS_SUBJECT`: NATS subject to publish metrics.
-	`NODE_NAME`: Name of the node.
-	`INSTANCE_ID`: Instance ID.
-	`INTERVAL`: Interval in seconds between metric collections.
-	`PROMETHEUS_PORT`: Port for Prometheus metrics.

Metrics Collected

-	`node_context_switches_total`: Total number of context switches on the node.
-	`node_entropy_available_bits`: Available entropy in bits, indicating the amount of randomness available.
-	`node_network_connections_total`: Total number of network connections on the node.

These metrics are crucial for understanding the low-level operations of the kernel and can be used to identify performance bottlenecks, security issues, and overall system health.

## Workflow

Metric Collection:
-	The tool collects kernel metrics at regular intervals as specified by the --interval flag or the INTERVAL environment variable.

Publishing Metrics:
-	If NATS integration is enabled, the collected metrics are published to the specified NATS subject.
-	If Prometheus metrics are enabled, the metrics are exposed on the specified port for scraping by Prometheus.

Logging and Monitoring:
-	Metrics can be logged locally for debugging purposes or monitored via Prometheus dashboards to provide a real-time view of kernel performance.

### Example Workflow

-	Start the server with the desired configuration:

```bash
prysm local-producer kernel-metrics --nats-url "nats://localhost:4222" --prometheus --prometheus-port 8080
```

-	Metrics such as context switches, entropy, and network connections will be collected every 10 seconds (default) and can be monitored through Prometheus or forwarded to a NATS server for further processing.

---
> This README is a draft and will be updated as the project continues to evolve. Contributions and feedback are welcome to help refine and enhance the functionality of Prysm.