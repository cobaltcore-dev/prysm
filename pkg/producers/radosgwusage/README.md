# RadosGW Usage Exporter (remote producer)

## Overview

The **RadosGW Usage Exporter (Prysm Remote Producer)** is a tool designed to collect and export
detailed usage metrics from RadosGW (Rados Gateway) instances. This exporter gathers data on
operations, byte metrics, bucket usage, quotas, and more, and can publish these metrics to a NATS
server or expose them for Prometheus, providing comprehensive visibility into the usage and
performance of your RadosGW environment.

## Key Features

- **Comprehensive Metric Collection**: Gathers a wide range of metrics including operations, bytes
  sent/received, bucket usage, quotas, and more.
- **NATS Integration**: Publishes collected usage data to a specified NATS subject, enabling
  real-time processing and integration with other observability tools.
- **Prometheus Metrics**: Exposes usage metrics in Prometheus format, allowing easy integration with
  monitoring dashboards.
- **Configurable**: Offers flexibility in configuration via command-line flags or environment
  variables.

## Usage

To run the Prysm remote producer for RadosGW usage, use the following command:

```bash
prysm remote-producer radosgw-usage [flags]
```

## Example Flags:

- `--admin-url "http://rgw-admin-url"`: Admin URL for the RadosGW instance.
- `--access-key "your-access-key"`: Access key for the RadosGW admin.
- `--secret-key "your-secret-key"`: Secret key for the RadosGW admin.
- `--interval 10`: Interval in seconds between usage collections (default is 10 seconds).
- `--nats-url "nats://localhost:4222"`: NATS server URL for publishing usage data.
- `--nats-subject "rgw.usage"`: NATS subject to publish usage data (default is “rgw.usage”).
- `--rgw-cluster-id`: RGW Cluster ID added to metrics.
- `--prometheus`: Enable Prometheus metrics.
- `--prometheus-port 8080`: Port for Prometheus metrics (default is 8080).

## Environment Variables

Configuration can also be set through environment variables:

- `ADMIN_URL`: Admin URL for the RadosGW instance.
- `ACCESS_KEY`: Access key for the RadosGW admin.
- `SECRET_KEY`: Secret key for the RadosGW admin.
- `NATS_URL`: NATS server URL.
- `NATS_SUBJECT`: NATS subject to publish usage data.
- `NODE_NAME`: Name of the node.
- `INSTANCE_ID`: Instance ID.
- `PROMETHEUS_ENABLED`: Enable Prometheus metrics.
- `PROMETHEUS_PORT`: Port for Prometheus metrics.
- `INTERVAL`: Interval in seconds between usage collections.
- `RGW_CLUSTER_ID`: RGW Cluster ID added to metrics.

## Metrics Collected

The RadosGW Usage Exporter collects and exposes the following metrics:


### Bucket Usage Metrics

- `radosgw_usage_bucket_bytes`: Bucket used bytes.
- `radosgw_usage_bucket_utilized_bytes`: Bucket utilized bytes.
- `radosgw_usage_bucket_objects`: Number of objects in the bucket.

### Quota Metrics

- `radosgw_usage_bucket_quota_enabled`: Indicates if quota is enabled for the bucket.
- `radosgw_usage_bucket_quota_size`: Maximum allowed bucket size.
- `radosgw_usage_bucket_quota_size_bytes`: Maximum allowed bucket size in bytes.
- `radosgw_usage_bucket_quota_size_objects`: Maximum allowed number of objects in the bucket.
- `radosgw_usage_user_quota_enabled`: Indicates if user quota is enabled.
- `radosgw_usage_user_quota_size`: Maximum allowed size for the user.
- `radosgw_usage_user_quota_size_objects`: Maximum allowed number of objects across all user buckets.

### Shards and User Metadata

- `radosgw_usage_bucket_shards`: Number of shards in the bucket.
- `radosgw_user_metadata`: User metadata (e.g., display name, email, storage class).

### Miscellaneous Metrics

- `radosgw_usage_scrape_duration_seconds`: Amount of time each scrape takes.

## Example Workflow

- Start the exporter with the desired configuration:

```bash
prysm remote-producer radosgw-usage --admin-url "http://rgw-admin-url" --access-key "your-access-key" --secret-key "your-secret-key" --rgw-cluster-id "rgw-cluster-id" --nats-url "nats://localhost:4222" --prometheus --prometheus-port 8080
```

- Metrics such as operations, bytes sent/received, and bucket usage will be collected every 10
  seconds (default) and can be monitored through Prometheus or forwarded to a NATS server for
  further processing.

## Acknowledgment

The basic idea for the RadosGW Usage Exporter and the prefix for metrics were inspired by the work
done in the [RadosGW Usage Exporter](https://github.com/blemmenes/radosgw_usage_exporter) by
Blemmenes. This project provided valuable insights and foundational concepts that have been adapted
and expanded upon in this implementation. We extend our thanks to the original authors for their
contributions to the open-source community.

---

> This README is a draft and will be updated as the project continues to evolve. Contributions and
> feedback are welcome to help refine and enhance the functionality of Prysm.
