SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and prysm contributors

SPDX-License-Identifier: Apache-2.0

# Monitoring Quota Usage (consumer)

## Overview

The **Quota Usage (Prysm Consumer)** is a tool designed to monitor and track quota usage for users
in a RadosGW environment. This consumer subscribes to quota usage data via a NATS subject, processes
it, and can alert when quota usage exceeds a specified threshold. The tool can also expose these
metrics for Prometheus, allowing for integration into monitoring dashboards.

## Key Features

- **Quota Monitoring**: Continuously monitors user quota usage and compares it against a defined
  threshold.
- **NATS Integration**: Subscribes to quota usage data from a specified NATS subject for real-time
  processing.
- **Prometheus Metrics**: Exposes quota usage metrics in Prometheus format for easy integration with
  monitoring systems.
- **Configurable**: Flexible configuration via command-line flags or environment variables.

## Usage

To run the Prysm quota usage consumer, use the following command:

```bash
prysm consumer quota-usage-consumer [flags]
```

## Example Flags:

- `--nats-url "nats://localhost:4222"`: NATS server URL for subscribing to quota usage data.
- `--nats-subject "user.quotas.usage"`: NATS subject to subscribe to (default is
  “user.quotas.usage”).
- `--instance-id "instance-1"`: Instance ID for identifying the source of the quotas.
- `--node-name "node-1"`: Node name for identifying the source of the quotas.
- `--quota-usage-percent 80`: Percentage of quota usage to monitor (default is 80%).
- `--prometheus`: Enable Prometheus metrics.
- `--prometheus-port 8080`: Port for Prometheus metrics (default is 8080).

## Environment Variables

Configuration can also be set through environment variables:

- `NATS_URL`: NATS server URL.
- `NATS_SUBJECT`: NATS subject to subscribe to.
- `PROMETHEUS_PORT`: Port for Prometheus metrics.
- `QUOTA_USAGE_PERCENT`: Percentage of quota usage to monitor.
- `NODE_NAME`: Name of the node.
- `INSTANCE_ID`: Instance ID.

## Logic and Workflow

The Quota Usage Consumer operates as follows:

Subscribe to NATS:

- The consumer connects to the specified NATS server and subscribes to the provided NATS subject to
  receive quota usage data.

Monitor Quota Usage:

- The consumer processes each quota usage message, calculating the percentage of used quota against
  the total quota.
- If the usage percentage exceeds the configured threshold, the metric is recorded and, if
  configured, exposed as a Prometheus metric.

Publish to Prometheus:

- If Prometheus integration is enabled, the consumer exposes quota usage metrics via Prometheus on
  the specified port.
- Metrics are updated in real-time as new data is received.

## Example Workflow

Start the quota usage consumer with the desired configuration:

```bash
prysm consumer quota-usage-consumer --nats-url "nats://localhost:4222" --quota-usage-percent 80 --prometheus --prometheus-port 8080
```

The consumer will subscribe to the user.quotas.usage NATS subject, monitor quota usage data, and
expose metrics via Prometheus if the usage exceeds 80%.

## Metrics Exposed

- `quota_usage_percent`: The percentage of quota usage for each user, exposed with labels for user
  ID, node name, and instance ID.

---

> This README is a draft and will be updated as the project continues to evolve. Contributions and
> feedback are welcome to help refine and enhance the functionality of Prysm.
