# RadosGW Usage Exporter (remote producer)

## Overview

The **RadosGW Usage Exporter (Prysm Remote Producer)** is a tool designed to collect and export detailed usage metrics from RadosGW (Rados Gateway) instances. This exporter gathers data on operations, byte metrics, bucket usage, quotas, and more, and can publish these metrics to a NATS server or expose them for Prometheus, providing comprehensive visibility into the usage and performance of your RadosGW environment.

## Key Features

- **Comprehensive Metric Collection**: Gathers a wide range of metrics including operations, bytes sent/received, bucket usage, quotas, and more.
- **NATS Integration**: Publishes collected usage data to a specified NATS subject, enabling real-time processing and integration with other observability tools.
- **Prometheus Metrics**: Exposes usage metrics in Prometheus format, allowing easy integration with monitoring dashboards.
- **Configurable**: Offers flexibility in configuration via command-line flags or environment variables.

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
- `--store "us-east-1"`: Store name added to metrics (default is “us-east-1”).
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
- `STORE`: Store name added to metrics.

## Metrics Collected

The RadosGW Usage Exporter collects and exposes the following metrics:

### Operation Metrics

- `radosgw_usage_ops_total`: Total number of operations.
- `radosgw_usage_successful_ops_total`: Total number of successful operations.

### Byte Metrics

- `radosgw_usage_sent_bytes_total`: Total bytes sent by RadosGW.
- `radosgw_usage_received_bytes_total`: Total bytes received by RadosGW.

### Bucket Usage Metrics

- `radosgw_usage_bucket_bytes`: Bucket used bytes.
- `radosgw_usage_bucket_utilized_bytes`: Bucket utilized bytes.
- `radosgw_usage_bucket_objects`: Number of objects in the bucket.

### Quota Metrics

- `radosgw_usage_bucket_quota_enabled`: Indicates if quota is enabled for the bucket.
- `radosgw_usage_bucket_quota_size`: Maximum allowed bucket size.
- `radosgw_usage_bucket_quota_size_bytes`: Maximum allowed bucket size in bytes.
- `radosgw_usage_bucket_quota_size_objects`: Maximum allowed number of objects in the bucket.

### Shards and User Metadata

- `radosgw_usage_bucket_shards`: Number of shards in the bucket.
- `radosgw_user_metadata`: User metadata (e.g., display name, email, storage class).

### User Quota Metrics

- `radosgw_usage_user_quota_enabled`: Indicates if user quota is enabled.
- `radosgw_usage_user_quota_size`: Maximum allowed size for the user.
- `radosgw_usage_user_quota_size_bytes`: Maximum allowed size in bytes for the user.
- `radosgw_usage_user_quota_size_objects`: Maximum allowed number of objects across all user buckets.

### Cluster-Level Metrics

- `radosgw_cluster_ops_total`: Total operations performed in the cluster.
- `radosgw_cluster_bytes_sent_total`: Total bytes sent in the cluster.
- `radosgw_cluster_bytes_received_total`: Total bytes received in the cluster.
- `radosgw_cluster_current_ops`: Current number of operations in the cluster.
- `radosgw_cluster_max_ops`: Maximum observed operations in the cluster.
- `radosgw_cluster_throughput_bytes_total`: Total throughput of the cluster in bytes.
- `radosgw_cluster_latency_seconds`: Latency/response times at the cluster level in seconds.

### User-Level Metrics

- `radosgw_user_buckets_total`: Total number of buckets for each user.
- `radosgw_user_objects_total`: Total number of objects for each user.
- `radosgw_user_data_size_bytes`: Total size of data for each user in bytes.
- `radosgw_user_ops_total`: Total operations performed by each user.
- `radosgw_user_bytes_sent_total`: Total bytes sent by each user.
- `radosgw_user_bytes_received_total`: Total bytes received by each user.
- `radosgw_user_current_ops`: Current number of operations for each user.
- `radosgw_user_max_ops`: Maximum observed operations for each user.
- `radosgw_user_requests_total`: Total number of requests made by each user.
- `radosgw_user_throughput_bytes_total`: Total throughput for each user in bytes.
- `radosgw_user_latency_seconds`: Latency/response times for each user in seconds.

### Bucket-Level Metrics

- `radosgw_bucket_ops_total`: Total operations performed in each bucket.
- `radosgw_bucket_bytes_sent_total`: Total bytes sent from each bucket.
- `radosgw_bucket_bytes_received_total`: Total bytes received by each bucket.
- `radosgw_bucket_current_ops`: Current number of operations in each bucket.
- `radosgw_bucket_max_ops`: Maximum observed operations in each bucket.
- `radosgw_bucket_requests_total`: Total number of requests made to each bucket.
- `radosgw_bucket_throughput_bytes_total`: Total throughput for each bucket in bytes.
- `radosgw_bucket_latency_seconds`: Latency/response times for each bucket in seconds.

### Miscellaneous Metrics

- `radosgw_usage_scrape_duration_seconds`: Amount of time each scrape takes.

## Example Workflow

-	Start the exporter with the desired configuration:
```bash
prysm remote-producer radosgw-usage --admin-url "http://rgw-admin-url" --access-key "your-access-key" --secret-key "your-secret-key" --nats-url "nats://localhost:4222" --prometheus --prometheus-port 8080
```
- Metrics such as operations, bytes sent/received, and bucket usage will be collected every 10 seconds (default) and can be monitored through Prometheus or forwarded to a NATS server for further processing.

## Acknowledgment

The basic idea for the RadosGW Usage Exporter and the prefix for metrics were inspired by the work done in the [RadosGW Usage Exporter](https://github.com/blemmenes/radosgw_usage_exporter) by Blemmenes. This project provided valuable insights and foundational concepts that have been adapted and expanded upon in this implementation. We extend our thanks to the original authors for their contributions to the open-source community.

---
> This README is a draft and will be updated as the project continues to evolve. Contributions and feedback are welcome to help refine and enhance the functionality of Prysm.