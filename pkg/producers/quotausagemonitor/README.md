# Quota Usage Monitor (remote producer)

## Overview

The **Quota Usage Monitor (Prysm Remote Producer)** is a tool designed to monitor and track the
quota usage for users in a RadosGW environment. This remote producer collects quota usage data from
the RadosGW Admin API, processes it, and publishes the results to a NATS server. The tool can also
log the quota usage data locally for monitoring and alerting purposes.

## Key Features

- **Quota Monitoring**: Collects and monitors user quota usage data, calculating the percentage of
  used quota against the total available quota.
- **NATS Integration**: Publishes collected quota usage data to a specified NATS subject for
  real-time processing and alerting.
- **Configurable**: Flexible configuration via command-line flags or environment variables.

## Usage

To run the Prysm quota usage monitor, use the following command:

```bash
prysm remote-producer quota-usage-monitor [flags]
```

## Example Flags:

- `--admin-url "http://rgw-admin-url"`: Admin API URL for the RadosGW instance.
- `--access-key "your-access-key"`: Access key for the RadosGW Admin API.
- `--secret-key "your-secret-key"`: Secret key for the RadosGW Admin API.
- `--nats-url "nats://localhost:4222`: NATS server URL for publishing quota usage data.
- `--nats-subject "user.quotas.usage"`: NATS subject to publish quota usage data (default is
  “user.quotas.usage”).
- `--instance-id "instance-1"`: Instance ID for identifying the source of the quota data.
- `--node-name "node-1"`: Node name for identifying the source of the quota data.
- `--quota-usage-percent 80`: Percentage of quota usage to monitor (default is 80%).
- `--interval 10`: Interval in seconds between quota usage collections (default is 10 seconds).

## Environment Variables

Configuration can also be set through environment variables:

- `ADMIN_URL`: Admin API URL for the RadosGW instance.
- `ACCESS_KEY`: Access key for the RadosGW Admin API.
- `SECRET_KEY`: Secret key for the RadosGW Admin API.
- `NATS_URL`: NATS server URL.
- `NATS_SUBJECT`: NATS subject to publish quota usage data.
- `NODE_NAME`: Name of the node.
- `INSTANCE_ID`: Instance ID.
- `INTERVAL`: Interval in seconds between quota usage collections.
- `QUOTA_USAGE_PERCENT`: Percentage of quota usage to monitor.

## Logic and Workflow

The Quota Usage Monitor operates as follows:

Connect to RadosGW Admin API:

- The tool connects to the RadosGW Admin API using the provided credentials (access key and secret
  key) and admin URL.

Collect Quota Usage Data:

- The tool retrieves a list of users from the RadosGW instance.
- For each user, it gathers the quota usage statistics, including the total quota, used quota, and
  remaining quota.
- If the user’s quota usage exceeds the configured percentage threshold, the quota usage data is
  recorded.

Publish to NATS or Log Locally:

- If NATS integration is enabled, the quota usage data is published to the specified NATS subject.
- If NATS is not configured, the quota usage data is logged to the console in JSON format.

Repeat at Regular Intervals:

- The tool repeats the quota usage collection and publication process at regular intervals as
  specified by the --interval flag or the INTERVAL environment variable.

## Example Workflow

- Start the quota usage monitor with the desired configuration:

```bash
prysm remote-producer quota-usage-monitor --admin-url "http://rgw-admin-url" --access-key "your-access-key" --secret-key "your-secret-key" --nats-url "nats://localhost:4222" --quota-usage-percent 80 --interval 10
```

- The monitor will connect to the RadosGW Admin API, collect quota usage data every 10 seconds, and
  publish any quota usage exceeding 80% to the user.quotas.usage NATS subject.

---

> This README is a draft and will be updated as the project continues to evolve. Contributions and
> feedback are welcome to help refine and enhance the functionality of Prysm.
