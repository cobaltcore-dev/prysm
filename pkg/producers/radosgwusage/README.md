# RadosGW Usage Exporter (remote producer)

## Overview

The **RadosGW Usage Exporter (Prysm Remote Producer)** is a tool designed to
collect and export detailed usage metrics from RadosGW (Rados Gateway)
instances. This exporter gathers data on operations, byte metrics, bucket usage,
quotas, and more, and can publish these metrics to a NATS server or expose them
for Prometheus, providing comprehensive visibility into the usage and
performance of your RadosGW environment.

## Key Features

- **Comprehensive Metric Collection**: Gathers a wide range of metrics including
  operations, bytes sent/received, bucket usage, quotas, and more.
- **NATS Integration**: Publishes collected usage data to a specified NATS
  subject, enabling real-time processing and integration with other
  observability tools.
- **Prometheus Metrics**: Exposes usage metrics in Prometheus format, allowing
  easy integration with monitoring dashboards.
- **Configurable**: Offers flexibility in configuration via command-line flags
  or environment variables.

## Usage

To run the Prysm remote producer for RadosGW usage, use the following command:

```bash
prysm remote-producer radosgw-usage [flags]
```

## Example Flags:

- `--admin-url "http://rgw-admin-url"`: Admin URL for the RadosGW instance.
- `--access-key "your-access-key"`: Access key for the RadosGW admin.
- `--secret-key "your-secret-key"`: Secret key for the RadosGW admin.
- `--interval 10`: Interval in seconds between usage collections (default is 10
  seconds).
- `--nats-url "nats://localhost:4222"`: NATS server URL for publishing usage
  data.
- `--nats-subject "rgw.usage"`: NATS subject to publish usage data (default is
  “rgw.usage”).
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
### Operation Metrics

- `radosgw_usage_ops_total`: Total number of operations across all buckets and users.
- `radosgw_usage_successful_ops_total`: Total number of successful operations across all buckets and users.
- `radosgw_user_ops_total`: Total operations performed by each user.
- `radosgw_user_read_ops_total`: Total read operations performed by each user.
- `radosgw_user_write_ops_total`: Total write operations performed by each user.
- `radosgw_user_success_ops_total`: Total number of successful operations per user.
- `radosgw_bucket_ops_total`: Total operations performed in each bucket.
- `radosgw_user_ops_per_sec`: Current number of operations (reads/writes) per second for each user.
- `radosgw_bucket_ops_per_sec`: Current number of operations per second for each bucket.
- `radosgw_bucket_read_ops_total`: Total read operations in each bucket.
- `radosgw_bucket_write_ops_total`: Total write operations in each bucket.
- `radosgw_bucket_success_ops_total`: Total successful operations for each bucket.

### Byte Metrics

- `radosgw_usage_sent_bytes_total`: Total bytes sent by RadosGW.
- `radosgw_usage_received_bytes_total`: Total bytes received by RadosGW.
- `radosgw_user_bytes_sent_total`: Total bytes sent by each user (cumulative).
- `radosgw_user_bytes_received_total`: Total bytes received by each user (cumulative).
- `radosgw_user_bytes_sent_per_sec`: Bytes sent by each user per second (rate).
- `radosgw_user_bytes_received_per_sec`: Bytes received by each user per second (rate).
- `radosgw_user_throughput_bytes_total`: Total throughput for each user in bytes (read and write combined).
- `radosgw_user_throughput_bytes_per_sec`: Current throughput in bytes per second for each user (read and write combined).
- `radosgw_bucket_bytes_sent_total`: Total bytes sent from each bucket.
- `radosgw_bucket_bytes_received_total`: Total bytes received by each bucket.
- `radosgw_bucket_bytes_sent_per_sec`: Current bytes sent per second from each bucket.
- `radosgw_bucket_bytes_received_per_sec`: Current bytes received per second by each bucket.
- `radosgw_bucket_throughput_bytes_per_sec`: Current throughput in bytes per second for each bucket (read and write combined).
- `radosgw_bucket_throughput_bytes_total`: Total throughput for each bucket in bytes (read and write combined).

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

### Cluster-Level Metrics

- `radosgw_cluster_ops_total`: Total operations performed in the cluster.
- `radosgw_cluster_reads_per_sec`: Total read operations per second for the entire cluster.
- `radosgw_cluster_writes_per_sec`: Total write operations per second for the entire cluster.
- `radosgw_cluster_ops_per_sec`: Current number of operations per second for the cluster.
- `radosgw_cluster_bytes_sent_total`: Total bytes sent in the cluster.
- `radosgw_cluster_bytes_received_total`: Total bytes received in the cluster.
- `radosgw_cluster_bytes_sent_per_sec`: Total bytes sent per second for the entire cluster.
- `radosgw_cluster_bytes_received_per_sec`: Total bytes received per second for the entire cluster.
- `radosgw_cluster_throughput_bytes_total`: Total throughput of the cluster in bytes (read and write combined).
- `radosgw_cluster_throughput_bytes_per_sec`: Total throughput in bytes per second for the entire cluster.
- `radosgw_cluster_error_rate`: Error rate (percentage) for the entire cluster.
- `radosgw_cluster_capacity_usage_bytes`: Total capacity used across the entire cluster in bytes.
- `radosgw_cluster_success_ops_total`: Total successful operations across the entire cluster.

### API Usage Metrics

- `radosgw_api_usage_per_user`: API usage per user and per category.
- `radosgw_bucket_api_usage_total`: Total number of API operations by category for each bucket.

### Miscellaneous Metrics

- `radosgw_usage_scrape_duration_seconds`: Amount of time each scrape takes.

## Example Workflow

- Start the exporter with the desired configuration:

```bash
prysm remote-producer radosgw-usage --admin-url "http://rgw-admin-url" --access-key "your-access-key" --secret-key "your-secret-key" --nats-url "nats://localhost:4222" --prometheus --prometheus-port 8080
```

- Metrics such as operations, bytes sent/received, and bucket usage will be
  collected every 10 seconds (default) and can be monitored through Prometheus
  or forwarded to a NATS server for further processing.

## Acknowledgment

The basic idea for the RadosGW Usage Exporter and the prefix for metrics were
inspired by the work done in the
[RadosGW Usage Exporter](https://github.com/blemmenes/radosgw_usage_exporter) by
Blemmenes. This project provided valuable insights and foundational concepts
that have been adapted and expanded upon in this implementation. We extend our
thanks to the original authors for their contributions to the open-source
community.

---

> This README is a draft and will be updated as the project continues to evolve.
> Contributions and feedback are welcome to help refine and enhance the
> functionality of Prysm.
