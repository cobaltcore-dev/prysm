# Local Producer - S3 Operations Log

## Overview

The **Local Producer - S3 Operations Log** is a tool designed to process and monitor S3 operation logs from Ceph RadosGW. It parses log entries, aggregates metrics, and provides real-time observability by publishing them to NATS or exposing them in Prometheus format.

## Key Features

- **S3 Log Processing**: Reads and parses Ceph RGW operation logs.
- **NATS Integration**: Publishes raw log events and aggregated metrics to NATS.
- **Prometheus Metrics**: Exposes operation metrics for Prometheus scraping.
- **Log File Rotation Support**: Monitors log file changes and rotates logs based on size and retention policies.
- **Configurable**: Allows customization via command-line flags or environment variables.
- **Anonymous Request Filtering**: Option to ignore anonymous requests to focus on authenticated users.
- **Granular Metrics Control**: Fine-grained toggles to enable/disable specific metric categories.
- **Auto Log Rotation on Startup**: Option to rotate log on start to avoid reprocessing.

## Usage

To run the local producer for S3 operations log, use the following command:

```bash
prysm local-producer ops-log [flags]
```

### Example Flags:

- `--log-file "/var/log/ceph/ceph-rgw-ops.json.log"` - Path to the S3 operations log file.
- `--socket-path "/tmp/ops-log.sock"` - Path to the Unix domain socket.
- `--nats-url "nats://localhost:4222"` - NATS server URL for publishing logs.
- `--nats-subject "rgw.s3.ops"` - NATS subject to publish raw log events.
- `--nats-metrics-subject "rgw.s3.ops.aggregated.metrics"` - NATS subject for aggregated metrics.
- `--log-to-stdout` - Enable logging operations to stdout.
- `--log-retention-days 1` - Number of days to retain old log files.
- `--max-log-file-size 10` - Maximum log file size in MB before rotation.
- `--prometheus` - Enable Prometheus metrics.
- `--prometheus-port 8080` - Port for Prometheus metrics.
- `--ignore-anonymous-requests` - Ignore anonymous requests in metrics.
- `--truncate-log-on-start` - Rotate log on start to avoid re-processing existing data.

### Environment Variables

| Environment Variable         | Description                                      |
|------------------------------|--------------------------------------------------|
| `LOG_FILE_PATH`              | Path to the S3 operations log file.             |
| `SOCKET_PATH`                | Path to the Unix domain socket.                 |
| `NATS_URL`                   | NATS server URL.                                |
| `NATS_SUBJECT`               | NATS subject for raw log events.                |
| `NATS_METRICS_SUBJECT`       | NATS subject for aggregated metrics.            |
| `LOG_TO_STDOUT`              | Enable logging operations to stdout.            |
| `LOG_RETENTION_DAYS`         | Number of days to retain old log files.         |
| `MAX_LOG_FILE_SIZE`          | Maximum log file size before rotation (in MB).  |
| `PROMETHEUS_PORT`            | Port for Prometheus metrics.                    |
| `IGNORE_ANONYMOUS_REQUESTS`  | Ignore anonymous requests in metrics.           |
| `TRUNCATE_LOG_ON_START`        | Whether to rotate the log file on startup.      |


#### Metric Toggle Environment Variables:

| Variable                                      | Description                                                    |
|-----------------------------------------------|----------------------------------------------------------------|
| `TRACK_REQUESTS_BY_IP`                        | Track requests per IP.                                         |
| `TRACK_BYTES_SENT_BY_IP`                     | Track bytes sent per IP.                                       |
| `TRACK_BYTES_RECEIVED_BY_IP`                 | Track bytes received per IP.                                   |
| `TRACK_BYTES_SENT_BY_USER`                   | Track bytes sent per user.                                     |
| `TRACK_BYTES_RECEIVED_BY_USER`               | Track bytes received per user.                                 |
| `TRACK_BYTES_SENT_BY_BUCKET`                 | Track bytes sent per bucket.                                   |
| `TRACK_BYTES_RECEIVED_BY_BUCKET`             | Track bytes received per bucket.                               |
| `TRACK_ERRORS_BY_IP`                         | Track HTTP errors per IP.                                      |
| `TRACK_ERRORS_BY_USER`                       | Track HTTP errors per user.                                    |
| `TRACK_ERRORS_BY_BUCKET`                     | Track errors per bucket.                                       |
| `TRACK_ERRORS_BY_STATUS`                     | Track errors by HTTP status code.                              |
| `TRACK_REQUESTS_BY_METHOD`                   | Track requests per HTTP method.                                |
| `TRACK_REQUESTS_BY_OPERATION`                | Track requests per operation.                                  |
| `TRACK_REQUESTS_BY_STATUS`                   | Track requests by HTTP status.                                 |
| `TRACK_REQUESTS_BY_BUCKET`                   | Track requests per bucket.                                     |
| `TRACK_REQUESTS_BY_USER`                     | Track requests per user.                                       |
| `TRACK_REQUESTS_BY_TENANT`                   | Track requests per tenant.                                     |
| `TRACK_REQUESTS_BY_IP_BUCKET_METHOD_TENANT`  | Track requests per IP, bucket, HTTP method and tenant.         |
| `TRACK_LATENCY_BY_USER`                      | Track latency per user.                                        |
| `TRACK_LATENCY_BY_BUCKET`                    | Track latency per bucket.                                      |
| `TRACK_LATENCY_BY_TENANT`                    | Track latency per tenant.                                      |
| `TRACK_LATENCY_BY_METHOD`                    | Track latency per HTTP method.                                 |
| `TRACK_LATENCY_BY_BUCKET_AND_METHOD`         | Track latency by bucket and method combination.                |


## Metrics Collected

| Metric Name                           | Type      | Labels                                               | Description                                                        |
|---------------------------------------|-----------|------------------------------------------------------|--------------------------------------------------------------------|
| `radosgw_total_requests`              | Counter   | `pod`, `user`, `tenant`, `bucket`, `method`, `http_status` | Total number of requests processed.                              |
| `radosgw_requests_by_method`          | Counter   | `pod`, `user`, `tenant`, `bucket`, `method`          | Number of requests grouped by HTTP method (GET, PUT, DELETE, etc.). |
| `radosgw_requests_by_operation`       | Counter   | `pod`, `user`, `tenant`, `bucket`, `operation`, `method` | Number of requests grouped by operation and HTTP method.          |
| `radosgw_requests_by_status`          | Counter   | `pod`, `user`, `tenant`, `bucket`, `status`          | Number of requests grouped by HTTP status code (200, 404, etc.).  |
| `radosgw_bytes_sent`                  | Counter   | `pod`, `user`, `tenant`, `bucket`                    | Total number of bytes sent.                                       |
| `radosgw_bytes_received`              | Counter   | `pod`, `user`, `tenant`, `bucket`                    | Total number of bytes received.                                   |
| `radosgw_errors_total`                | Counter   | `pod`, `user`, `tenant`, `bucket`, `http_status`     | Total number of error responses by user, bucket, and status code. |
| `radosgw_http_errors_by_user`         | Counter   | `pod`, `user`, `tenant`, `bucket`, `http_status`     | HTTP errors grouped by user, tenant, bucket, and status code.     |
| `radosgw_http_errors_by_ip`           | Counter   | `pod`, `bucket`, `ip`, `http_status`                 | HTTP errors grouped by IP, bucket, and status code.               |
| `radosgw_requests_by_ip`              | Gauge     | `pod`, `user`, `tenant`, `ip`                        | Total number of requests grouped by IP and user.                  |
| `radosgw_requests_by_ip_bucket_method_tenant`              | Gauge     | `pod`, `ip`, `bucket`, `method`, `tenant`                        | Total number of requests grouped by IP, bucket and method.                  |
| `radosgw_bytes_sent_by_ip`            | Gauge     | `pod`, `user`, `tenant`, `ip`                        | Total bytes sent grouped by IP and user.                          |
| `radosgw_bytes_received_by_ip`        | Gauge     | `pod`, `user`, `tenant`, `ip`                        | Total bytes received grouped by IP and user.                      |
| `radosgw_requests_duration`           | Histogram | `user`, `tenant`, `bucket`, `method`                 | Histogram of request latencies (in seconds).                      |

> Histograms do **not** include the `pod` label to reduce cardinality.


## Workflow

1. **Log Processing**: Reads and parses incoming log entries from the Ceph RGW log file.
2. **Metrics Aggregation**: Updates counters and gauges based on extracted information.
3. **Publishing to NATS**: Raw log events and aggregated metrics are sent to specified NATS subjects.
4. **Prometheus Metrics**: Exposes metrics via an HTTP server for Prometheus scraping.
5. **File Rotation Handling**: Monitors log file size and age, triggering rotation when needed.
6. **Log Rotation on Start** *(optional)*: Backs up and clears the log file at startup to avoid re-processing.

## Example Workflow

```bash
prysm local-producer ops-log \
  --log-file /var/log/ceph/ops-log.log \
  --nats-url nats://localhost:4222 \
  --prometheus --prometheus-port 8080 \
  --rotate-log-on-start
```

## Notes

- Ensure that the Ceph RGW log format is JSON-based to be compatible with this tool.
- If using NATS, ensure the server is running and accessible from the producer.
- Prometheus should be configured to scrape the exposed metrics endpoint.
- Sidecar injection is supported via a mutating webhook (see related documentation for Kubernetes usage).

> This README will be updated as new features and improvements are introduced. Contributions and feedback are welcome!