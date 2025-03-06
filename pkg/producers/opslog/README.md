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
- `--log-to-stdout` - Enable logging operations to stdout instead of a file.
- `--log-retention-days 1` - Number of days to retain old log files.
- `--max-log-file-size 10` - Maximum log file size in MB before rotation.
- `--prometheus` - Enable Prometheus metrics.
- `--prometheus-port 8080` - Port for Prometheus metrics.
- `--ignore-anonymous-requests` - Ignore anonymous requests in metrics.

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

## Metrics Collected

| Metric Name                           | Type      | Labels                                 | Description                                         |
|----------------------------------------|----------|----------------------------------------|-----------------------------------------------------|
| `radosgw_total_requests`              | Counter  | `pod`, `user`, `tenant`, `bucket`     | Total number of requests processed.                |
| `radosgw_requests_by_method`          | Counter  | `pod`, `user`, `tenant`, `bucket`, `method` | Number of requests grouped by HTTP method.         |
| `radosgw_requests_by_operation`       | Counter  | `pod`, `user`, `tenant`, `bucket`, `operation`, `method` | Number of requests grouped by operation.           |
| `radosgw_requests_by_status`          | Counter  | `pod`, `user`, `tenant`, `bucket`, `status` | Number of requests grouped by HTTP status code.    |
| `radosgw_bytes_sent`                  | Counter  | `pod`, `user`, `tenant`, `bucket`     | Total bytes sent.                                  |
| `radosgw_bytes_received`              | Counter  | `pod`, `user`, `tenant`, `bucket`     | Total bytes received.                              |
| `radosgw_errors_total`                | Counter  | `pod`, `user`, `tenant`, `bucket`     | Total number of errors.                           |
| `radosgw_latency_min_seconds`         | Gauge    | `pod`, `user`, `tenant`, `bucket`     | Minimum request latency in seconds.               |
| `radosgw_latency_max_seconds`         | Gauge    | `pod`, `user`, `tenant`, `bucket`     | Maximum request latency in seconds.               |
| `radosgw_latency_avg_seconds`         | Gauge    | `pod`, `user`, `tenant`, `bucket`     | Average request latency in seconds.               |
| `radosgw_requests_by_ip`              | Gauge    | `pod`, `user`, `tenant`, `ip`         | Total number of requests per IP and user.         |
| `radosgw_bytes_sent_by_ip`            | Gauge    | `pod`, `user`, `tenant`, `ip`         | Total bytes sent per IP and user.                 |
| `radosgw_bytes_received_by_ip`        | Gauge    | `pod`, `user`, `tenant`, `ip`         | Total bytes received per IP and user.             |
| `radosgw_http_errors_by_user`         | Counter  | `pod`, `user`, `tenant`, `bucket`, `http_status` | Total HTTP errors by user and bucket.             |
| `radosgw_http_errors_by_ip`           | Counter  | `pod`, `bucket`, `ip`, `http_status`  | Total HTTP errors by IP and bucket.               |

## Workflow

1. **Log Processing**: Reads and parses incoming log entries from the Ceph RGW log file.
2. **Metrics Aggregation**: Updates counters and gauges based on extracted information.
3. **Publishing to NATS**: Raw log events and aggregated metrics are sent to specified NATS subjects.
4. **Prometheus Metrics**: Exposes metrics via an HTTP server for Prometheus scraping.
5. **File Rotation Handling**: Monitors log file size and age, triggering rotation when needed.

## Example Workflow

1. Start the local producer with desired configurations:

   ```bash
   prysm local-producer ops-log --nats-url "nats://localhost:4222" --prometheus --prometheus-port 8080
   ```

2. The tool will continuously parse S3 operation logs, collect metrics, and expose them for Prometheus or publish to NATS.

3. Prometheus can scrape the metrics endpoint, and dashboards can visualize the data.

## Notes

- Ensure that the Ceph RGW log format is JSON-based to be compatible with this tool.
- If using NATS, ensure the server is running and accessible from the producer.
- Prometheus should be configured to scrape the exposed metrics endpoint.

> This README will be updated as new features and improvements are introduced. Contributions and feedback are welcome!