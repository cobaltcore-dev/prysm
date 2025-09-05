# Local Producer - S3 Operations Log

## Overview

The **Local Producer - S3 Operations Log** is a tool designed to process and monitor S3 operation logs from Ceph RadosGW. It parses log entries, aggregates metrics, and provides real-time observability by publishing them to NATS or exposing them in Prometheus format.

## Key Features

- **S3 Log Processing**: Reads and parses Ceph RGW operation logs.
- **NATS Integration**: Publishes raw log events and aggregated metrics to NATS.
- **Prometheus Metrics**: Exposes operation metrics for Prometheus scraping.
- **Latency Tracking**: Real-time request latency histograms with multiple aggregation levels.
- **Memory Efficient Architecture**: Dedicated storage maps for each metric type ensure minimal memory usage.
- **Log File Rotation Support**: Monitors log file changes and rotates logs based on size and retention policies.
- **Configurable**: Allows customization via command-line flags or environment variables.
- **Anonymous Request Filtering**: Option to ignore anonymous requests to focus on authenticated users.
- **Granular Metrics Control**: Fine-grained toggles to enable/disable specific metric categories.
- **Auto Log Rotation on Startup**: Option to rotate log on start to avoid reprocessing.
- **Multi-Tenant Support**: Proper tenant separation ensures metrics from different tenants are isolated, even for buckets with identical names.
- **Zero-Value Error Metrics**: Error metrics always report 0 when no errors occur, ensuring visibility in monitoring dashboards.
- **Timeout Error Detection**: Specialized timeout error tracking (408, 504, 598, 499) for detecting OSD-related issues.
- **Error Categorization**: Automatic categorization of HTTP errors into timeout, connection, client, and server errors.

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
- `--prometheus-interval 60` - Prometheus metrics update interval in seconds.
- `--ignore-anonymous-requests` - Ignore anonymous requests in metrics.
- `--truncate-log-on-start` - Rotate log on start to avoid re-processing existing data.
- `--track-everything` - Enable detailed tracking for all metric types (efficient mode).
- `--track-timeout-errors` - Enable tracking of timeout errors (408, 504, 598, 499) for OSD issue detection.
- `--track-errors-by-category` - Enable error categorization (timeout, connection, client, server).

### Latency Tracking Examples:
 
```bash
# Enable all latency tracking
prysm local-producer ops-log \
  --log-file /var/log/ceph/ops-log.log \
  --prometheus --prometheus-port 8080 \
  --track-latency-detailed \
  --track-latency-per-method \
  --track-latency-per-user \
  --track-latency-per-bucket

# Enable everything with shortcut
prysm local-producer ops-log \
  --log-file /var/log/ceph/ops-log.log \
  --prometheus --prometheus-port 8080 \
  --track-everything
```

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
| `PROMETHEUS_INTERVAL`        | Prometheus metrics update interval in seconds.  |
| `IGNORE_ANONYMOUS_REQUESTS`  | Ignore anonymous requests in metrics.           |
| `TRUNCATE_LOG_ON_START`      | Whether to rotate the log file on startup.      |
| `TRACK_EVERYTHING`           | Enable detailed tracking for all metric types.  |

#### Request Tracking Environment Variables:

| Variable                                      | Description                                                    |
|-----------------------------------------------|----------------------------------------------------------------|
| `TRACK_REQUESTS_DETAILED`                     | Track detailed requests with full labels.                     |
| `TRACK_REQUESTS_PER_USER`                     | Track requests aggregated per user.                           |
| `TRACK_REQUESTS_PER_BUCKET`                   | Track requests aggregated per bucket.                         |
| `TRACK_REQUESTS_PER_TENANT`                   | Track requests aggregated per tenant.                         |

#### Method-based Request Tracking:

| Variable                                      | Description                                                    |
|-----------------------------------------------|----------------------------------------------------------------|
| `TRACK_REQUESTS_BY_METHOD_DETAILED`           | Track detailed requests by HTTP method.                       |
| `TRACK_REQUESTS_BY_METHOD_PER_USER`           | Track requests by method per user.                            |
| `TRACK_REQUESTS_BY_METHOD_PER_BUCKET`         | Track requests by method per bucket.                          |
| `TRACK_REQUESTS_BY_METHOD_PER_TENANT`         | Track requests by method per tenant.                          |
| `TRACK_REQUESTS_BY_METHOD_GLOBAL`             | Track requests by method globally.                            |

#### Operation-based Request Tracking:

| Variable                                      | Description                                                    |
|-----------------------------------------------|----------------------------------------------------------------|
| `TRACK_REQUESTS_BY_OPERATION_DETAILED`        | Track detailed requests by operation.                         |
| `TRACK_REQUESTS_BY_OPERATION_PER_USER`        | Track requests by operation per user.                         |
| `TRACK_REQUESTS_BY_OPERATION_PER_BUCKET`      | Track requests by operation per bucket.                       |
| `TRACK_REQUESTS_BY_OPERATION_PER_TENANT`      | Track requests by operation per tenant.                       |
| `TRACK_REQUESTS_BY_OPERATION_GLOBAL`          | Track requests by operation globally.                         |

#### Status-based Request Tracking:

| Variable                                      | Description                                                    |
|-----------------------------------------------|----------------------------------------------------------------|
| `TRACK_REQUESTS_BY_STATUS_DETAILED`           | Track detailed requests by status.                            |
| `TRACK_REQUESTS_BY_STATUS_PER_USER`           | Track requests by status per user.                            |
| `TRACK_REQUESTS_BY_STATUS_PER_BUCKET`         | Track requests by status per bucket.                          |
| `TRACK_REQUESTS_BY_STATUS_PER_TENANT`         | Track requests by status per tenant.                          |

#### Bytes Tracking Environment Variables:

| Variable                                      | Description                                                    |
|-----------------------------------------------|----------------------------------------------------------------|
| `TRACK_BYTES_SENT_DETAILED`                   | Track detailed bytes sent.                                    |
| `TRACK_BYTES_SENT_PER_USER`                   | Track bytes sent per user.                                    |
| `TRACK_BYTES_SENT_PER_BUCKET`                 | Track bytes sent per bucket (with tenant separation).         |
| `TRACK_BYTES_SENT_PER_TENANT`                 | Track bytes sent per tenant.                                  |
| `TRACK_BYTES_RECEIVED_DETAILED`               | Track detailed bytes received.                                |
| `TRACK_BYTES_RECEIVED_PER_USER`               | Track bytes received per user.                                |
| `TRACK_BYTES_RECEIVED_PER_BUCKET`             | Track bytes received per bucket (with tenant separation).     |
| `TRACK_BYTES_RECEIVED_PER_TENANT`             | Track bytes received per tenant.                              |

#### Error Tracking Environment Variables:

| Variable                                      | Description                                                    |
|-----------------------------------------------|----------------------------------------------------------------|
| `TRACK_ERRORS_DETAILED`                       | Track detailed errors.                                        |
| `TRACK_ERRORS_PER_USER`                       | Track errors per user.                                        |
| `TRACK_ERRORS_PER_BUCKET`                     | Track errors per bucket (with tenant separation).             |
| `TRACK_ERRORS_PER_TENANT`                     | Track errors per tenant.                                      |
| `TRACK_ERRORS_PER_STATUS`                     | Track errors per HTTP status code.                            |
| `TRACK_ERRORS_BY_IP`                          | Track errors by IP address.                                   |
| `TRACK_TIMEOUT_ERRORS`                        | Track timeout errors (408, 504, 598, 499) for OSD detection.  |
| `TRACK_ERRORS_BY_CATEGORY`                    | Track errors by category (timeout, connection, client, server).|

#### IP-based Tracking Environment Variables:

| Variable                                      | Description                                                    |
|-----------------------------------------------|----------------------------------------------------------------|
| `TRACK_REQUESTS_BY_IP_DETAILED`               | Track requests by IP.                                         |
| `TRACK_REQUESTS_BY_IP_PER_TENANT`             | Track requests by IP per tenant.                              |
| `TRACK_REQUESTS_BY_IP_BUCKET_METHOD_TENANT`   | Track requests by IP, bucket, method and tenant.              |
| `TRACK_REQUESTS_BY_IP_GLOBAL_PER_TENANT`      | Track requests by IP globally per tenant.                     |
| `TRACK_BYTES_SENT_BY_IP_DETAILED`             | Track bytes sent by IP.                                       |
| `TRACK_BYTES_SENT_BY_IP_PER_TENANT`           | Track bytes sent by IP per tenant.                            |
| `TRACK_BYTES_SENT_BY_IP_GLOBAL_PER_TENANT`    | Track bytes sent by IP globally per tenant.                   |
| `TRACK_BYTES_RECEIVED_BY_IP_DETAILED`         | Track bytes received by IP.                                   |
| `TRACK_BYTES_RECEIVED_BY_IP_PER_TENANT`       | Track bytes received by IP per tenant.                        |
| `TRACK_BYTES_RECEIVED_BY_IP_GLOBAL_PER_TENANT`| Track bytes received by IP globally per tenant.               |

#### Latency Tracking Environment Variables:

| Variable                                      | Description                                                    |
|-----------------------------------------------|----------------------------------------------------------------|
| `TRACK_LATENCY_DETAILED`                      | Track detailed latency with full labels.                      |
| `TRACK_LATENCY_PER_USER`                      | Track latency aggregated per user.                            |
| `TRACK_LATENCY_PER_BUCKET`                    | Track latency aggregated per bucket.                          |
| `TRACK_LATENCY_PER_TENANT`                    | Track latency aggregated per tenant.                          |
| `TRACK_LATENCY_PER_METHOD`                    | Track latency aggregated per HTTP method.                     |
| `TRACK_LATENCY_PER_BUCKET_AND_METHOD`         | Track latency by bucket and method combination.               |

## Metrics Collected

### Request Counters

| Metric Name                           | Type      | Labels                                               | Description                                                        |
|---------------------------------------|-----------|------------------------------------------------------|--------------------------------------------------------------------|
| `radosgw_total_requests`              | Counter   | `pod`, `user`, `tenant`, `bucket`, `method`, `http_status` | Total number of requests processed with full dimensionality.     |
| `radosgw_total_requests_per_user`     | Counter   | `pod`, `user`, `tenant`, `method`, `http_status`     | Total requests aggregated per user (all buckets combined).        |
| `radosgw_total_requests_per_bucket`   | Counter   | `pod`, `tenant`, `bucket`, `method`, `http_status`   | Total requests aggregated per bucket (all users combined).        |
| `radosgw_total_requests_per_tenant`   | Counter   | `pod`, `tenant`, `method`, `http_status`             | Total requests aggregated per tenant (all users and buckets).     |

### Method-based Request Counters

| Metric Name                                   | Type      | Labels                                               | Description                                                        |
|-----------------------------------------------|-----------|------------------------------------------------------|--------------------------------------------------------------------|
| `radosgw_requests_by_method`                  | Counter   | `pod`, `user`, `tenant`, `bucket`, `method`          | Number of requests grouped by HTTP method with full detail.       |
| `radosgw_requests_by_method_per_user`         | Counter   | `pod`, `user`, `tenant`, `method`                    | Number of requests by method aggregated per user.                 |
| `radosgw_requests_by_method_per_bucket`       | Counter   | `pod`, `tenant`, `bucket`, `method`                  | Number of requests by method aggregated per bucket.               |
| `radosgw_requests_by_method_per_tenant`       | Counter   | `pod`, `tenant`, `method`                            | Number of requests by method aggregated per tenant.               |
| `radosgw_requests_by_method_global`           | Counter   | `pod`, `method`                                      | Number of requests by method globally aggregated.                 |

### Operation-based Request Counters

| Metric Name                                   | Type      | Labels                                               | Description                                                        |
|-----------------------------------------------|-----------|------------------------------------------------------|--------------------------------------------------------------------|
| `radosgw_requests_by_operation`               | Counter   | `pod`, `user`, `tenant`, `bucket`, `operation`, `method` | Number of requests grouped by operation with full detail.         |
| `radosgw_requests_by_operation_per_user`      | Counter   | `pod`, `user`, `tenant`, `operation`, `method`       | Number of requests by operation aggregated per user.              |
| `radosgw_requests_by_operation_per_bucket`    | Counter   | `pod`, `tenant`, `bucket`, `operation`, `method`     | Number of requests by operation aggregated per bucket.            |
| `radosgw_requests_by_operation_per_tenant`    | Counter   | `pod`, `tenant`, `operation`, `method`               | Number of requests by operation aggregated per tenant.            |
| `radosgw_requests_by_operation_global`        | Counter   | `pod`, `operation`, `method`                         | Number of requests by operation globally aggregated.              |

### Status-based Request Counters

| Metric Name                                   | Type      | Labels                                               | Description                                                        |
|-----------------------------------------------|-----------|------------------------------------------------------|--------------------------------------------------------------------|
| `radosgw_requests_by_status_detailed`         | Counter   | `pod`, `user`, `tenant`, `bucket`, `status`          | Number of requests grouped by HTTP status with full detail.       |
| `radosgw_requests_by_status_per_user`         | Counter   | `pod`, `user`, `tenant`, `status`                    | Number of requests by status aggregated per user.                 |
| `radosgw_requests_by_status_per_bucket`       | Counter   | `pod`, `tenant`, `bucket`, `status`                  | Number of requests by status aggregated per bucket.               |
| `radosgw_requests_by_status_per_tenant`       | Counter   | `pod`, `tenant`, `status`                            | Number of requests by status aggregated per tenant.               |

### Bytes Transferred Counters

| Metric Name                           | Type      | Labels                                               | Description                                                        |
|---------------------------------------|-----------|------------------------------------------------------|--------------------------------------------------------------------|
| `radosgw_bytes_sent`                  | Counter   | `pod`, `user`, `tenant`, `bucket`                    | Total number of bytes sent with proper tenant separation.         |
| `radosgw_bytes_received`              | Counter   | `pod`, `user`, `tenant`, `bucket`                    | Total number of bytes received with proper tenant separation.     |
| `radosgw_bytes_sent_per_user`         | Counter   | `pod`, `user`, `tenant`                              | Total bytes sent aggregated per user (all buckets combined).      |
| `radosgw_bytes_received_per_user`     | Counter   | `pod`, `user`, `tenant`                              | Total bytes received aggregated per user (all buckets combined).  |
| `radosgw_bytes_sent_per_bucket`       | Counter   | `pod`, `tenant`, `bucket`                            | Total bytes sent aggregated per bucket (all users combined).      |
| `radosgw_bytes_received_per_bucket`   | Counter   | `pod`, `tenant`, `bucket`                            | Total bytes received aggregated per bucket (all users combined).  |
| `radosgw_bytes_sent_per_tenant`       | Counter   | `pod`, `tenant`                                      | Total bytes sent aggregated per tenant (all users and buckets).   |
| `radosgw_bytes_received_per_tenant`   | Counter   | `pod`, `tenant`                                      | Total bytes received aggregated per tenant (all users and buckets). |

### Error Counters

| Metric Name                           | Type      | Labels                                               | Description                                                        |
|---------------------------------------|-----------|------------------------------------------------------|--------------------------------------------------------------------|
| `radosgw_errors_detailed`             | Counter   | `pod`, `user`, `tenant`, `bucket`, `http_status`     | Total number of errors with full detail. **Always shows 0 when no errors**.  |
| `radosgw_errors_per_user`             | Counter   | `pod`, `user`, `tenant`, `http_status`               | Total errors aggregated per user. **Always visible with value 0 when no errors**. |
| `radosgw_errors_per_bucket`           | Counter   | `pod`, `tenant`, `bucket`, `http_status`             | Total errors aggregated per bucket. **Always visible with value 0 when no errors**. |
| `radosgw_errors_per_tenant`           | Counter   | `pod`, `tenant`, `http_status`                       | Total errors aggregated per tenant. **Always visible with value 0 when no errors**. |
| `radosgw_errors_per_status`           | Counter   | `pod`, `http_status`                                 | Total errors aggregated per HTTP status code. **Always visible with value 0 when no errors**. |
| `radosgw_errors_per_ip`               | Counter   | `pod`, `ip`, `tenant`, `http_status`                 | Total errors aggregated per IP address. **Always visible with value 0 when no errors**. |

### Timeout Error Counters (New)

| Metric Name                           | Type      | Labels                                               | Description                                                        |
|---------------------------------------|-----------|------------------------------------------------------|--------------------------------------------------------------------|
| `radosgw_timeout_errors`              | Counter   | `pod`, `user`, `tenant`, `bucket`, `timeout_type`    | Total timeout errors by type (408, 504, 598, 499) for OSD issue detection. |

### Error Category Counters (New)

| Metric Name                           | Type      | Labels                                               | Description                                                        |
|---------------------------------------|-----------|------------------------------------------------------|--------------------------------------------------------------------|
| `radosgw_errors_by_category`          | Counter   | `pod`, `user`, `tenant`, `bucket`, `category`        | Errors categorized as: timeout, connection, client, server for better monitoring. |

### IP-based Gauges

| Metric Name                                  | Type      | Labels                                               | Description                                                        |
|----------------------------------------------|-----------|------------------------------------------------------|--------------------------------------------------------------------|
| `radosgw_requests_by_ip`                     | Gauge     | `pod`, `user`, `tenant`, `ip`                        | Total number of requests grouped by IP and user.                  |
| `radosgw_requests_per_ip`                    | Gauge     | `pod`, `tenant`, `ip`                                | Total requests aggregated per IP (all users combined).            |
| `radosgw_requests_per_tenant_from_ip`        | Gauge     | `pod`, `tenant`                                      | Total requests aggregated per tenant from all IPs.                |
| `radosgw_requests_by_ip_bucket_method_tenant`| Gauge     | `pod`, `ip`, `bucket`, `method`, `tenant`            | Total number of requests grouped by IP, bucket and method.        |
| `radosgw_bytes_sent_by_ip`                   | Gauge     | `pod`, `user`, `tenant`, `ip`                        | Total bytes sent grouped by IP and user.                          |
| `radosgw_bytes_sent_per_ip`                  | Gauge     | `pod`, `tenant`, `ip`                                | Total bytes sent aggregated per IP (all users combined).          |
| `radosgw_bytes_sent_per_tenant_from_ip`      | Gauge     | `pod`, `tenant`                                      | Total bytes sent aggregated per tenant from all IPs.              |
| `radosgw_bytes_received_by_ip`               | Gauge     | `pod`, `user`, `tenant`, `ip`                        | Total bytes received grouped by IP and user.                      |
| `radosgw_bytes_received_per_ip`              | Gauge     | `pod`, `tenant`, `ip`                                | Total bytes received aggregated per IP (all users combined).      |
| `radosgw_bytes_received_per_tenant_from_ip`  | Gauge     | `pod`, `tenant`                                      | Total bytes received aggregated per tenant from all IPs.          |

### Latency Histograms

| Metric Name                                          | Type      | Labels                                               | Description                                                        |
|------------------------------------------------------|-----------|------------------------------------------------------|--------------------------------------------------------------------|
| `radosgw_requests_duration`                          | Histogram | `user`, `tenant`, `bucket`, `method`                 | Histogram of request latencies with full detail (in seconds).     |
| `radosgw_requests_duration_per_user`                 | Histogram | `user`, `tenant`, `method`                           | Histogram for request latencies aggregated per user (all buckets combined). |
| `radosgw_requests_duration_per_bucket`               | Histogram | `tenant`, `bucket`, `method`                         | Histogram for request latencies aggregated per bucket (all users combined). |
| `radosgw_requests_duration_per_tenant`               | Histogram | `tenant`, `method`                                   | Histogram for request latencies aggregated per tenant (all users and buckets combined). |
| `radosgw_requests_duration_per_method`               | Histogram | `method`                                             | Histogram for request latencies aggregated per method (global).   |
| `radosgw_requests_duration_per_bucket_and_method`    | Histogram | `tenant`, `bucket`, `method`                         | Histogram for request latencies aggregated per bucket and method (all users combined). |

> **Note**: Histogram metrics do **not** include the `pod` label to reduce cardinality. Each histogram automatically provides `_bucket`, `_count`, and `_sum` metrics for comprehensive latency analysis.

### Memory Efficiency Architecture

The system uses a **dedicated storage architecture** where each metric type has its own optimized storage map:

- **Memory Efficient**: Only enabled metric types consume memory
- **Optimal Granularity**: Each aggregation level stores exactly the data it needs
- **No Runtime Aggregation**: All aggregation happens at storage time for better performance
- **Independent Metrics**: Each metric can be enabled/disabled independently without affecting others

### Multi-Tenant Support

All bucket-level metrics now properly separate tenants to avoid data collision:
- **Bucket metrics** include tenant information to distinguish between buckets with identical names across different tenants
- **IP-based error tracking** includes tenant context for proper attribution
- **Aggregation levels** provide both tenant-specific and tenant-aggregated views for flexible monitoring

### Metric Aggregation Levels

Many metrics provide multiple aggregation levels for flexible monitoring:
- **Full granularity**: Complete dimensional breakdown (user, tenant, bucket, method, status)
- **Per-user**: Aggregated by user across all buckets
- **Per-bucket**: Aggregated by bucket with tenant separation across all users
- **Per-tenant**: Aggregated across all buckets and users within a tenant
- **Global**: Fully aggregated across all dimensions

## Workflow

1. **Log Processing**: Reads and parses incoming log entries from the Ceph RGW log file.
2. **Dedicated Storage**: Updates dedicated storage maps based on enabled metric types with proper tenant separation.
3. **Latency Recording**: Records request latencies from the `total_time` field directly into Prometheus histograms.
4. **Publishing to NATS**: Raw log events and aggregated metrics are sent to specified NATS subjects.
5. **Prometheus Metrics**: Exposes metrics via an HTTP server for Prometheus scraping.
6. **File Rotation Handling**: Monitors log file size and age, triggering rotation when needed.
7. **Log Rotation on Start** *(optional)*: Backs up and clears the log file at startup to avoid re-processing.

## Example Workflows

### Basic Monitoring with Latency Tracking

```bash
prysm local-producer ops-log \
  --log-file /var/log/ceph/ops-log.log \
  --prometheus --prometheus-port 8080 \
  --track-latency-detailed \
  --track-latency-per-method \
  --truncate-log-on-start
```

### Comprehensive Monitoring

```bash
prysm local-producer ops-log \
  --log-file /var/log/ceph/ops-log.log \
  --nats-url nats://localhost:4222 \
  --prometheus --prometheus-port 8080 \
  --prometheus-interval 30 \
  --track-everything \
  --ignore-anonymous-requests
```

### Minimal Resource Usage

```bash
prysm local-producer ops-log \
  --log-file /var/log/ceph/ops-log.log \
  --prometheus --prometheus-port 8080 \
  --track-latency-per-method \
  --track-requests-per-tenant \
  --track-errors-per-user
```

## Configuration Best Practices

### Performance Considerations

- **Use `--track-everything` carefully**: While convenient, it creates many metrics which can impact performance
- **Selective tracking**: Enable only the metrics you actually need for monitoring
- **Latency tracking**: Start with `--track-latency-per-method` and add more granular tracking as needed
- **Anonymous requests**: Use `--ignore-anonymous-requests` to reduce noise in multi-tenant environments
- **Memory efficiency**: Each metric type uses dedicated storage, so only enabled metrics consume memory

### Recommended Configurations

**For development/testing:**
```bash
--track-everything --prometheus-interval 10
```

**For production (minimal):**
```bash
--track-latency-per-method --track-requests-per-tenant --track-errors-per-user
```

**For production (comprehensive):**
```bash
--track-latency-detailed --track-latency-per-method --track-requests-per-user --track-requests-per-bucket --track-errors-per-user --track-bytes-sent-per-bucket
```

## Error Monitoring Best Practices

### Zero-Value Error Metrics
All error metrics now report 0 when no errors occur, ensuring they remain visible in Prometheus and Grafana dashboards. This improvement:
- Eliminates the "No data" issue in dashboards
- Allows for proper rate calculations even when errors are intermittent
- Ensures alerting rules work correctly with absent metrics

### Timeout Error Detection for OSD Issues
The new `radosgw_timeout_errors` metric specifically tracks timeout-related HTTP status codes:
- **408 (Request Timeout)**: Client took too long to send request
- **504 (Gateway Timeout)**: Upstream server timeout (often indicates OSD issues)
- **598 (Network Read Timeout)**: Network-level timeout
- **499 (Client Closed Request)**: Client disconnected before response

Use these metrics to detect OSD performance issues:
```promql
# Alert when timeout errors exceed threshold
rate(radosgw_timeout_errors[5m]) > 0.1
```

### Error Categorization
The `radosgw_errors_by_category` metric automatically categorizes errors:
- **timeout**: 408, 504, 598, 499 status codes
- **connection**: 502, 503 status codes
- **client**: 4xx errors (excluding timeouts)
- **server**: 5xx errors (excluding timeouts and connection errors)

This simplifies monitoring and alerting:
```promql
# Alert on server errors
rate(radosgw_errors_by_category{category="server"}[5m]) > 0.05

# Alert on connection issues
rate(radosgw_errors_by_category{category="connection"}[5m]) > 0.1
```

## Notes

- Ensure that the Ceph RGW log format is JSON-based to be compatible with this tool.
- If using NATS, ensure the server is running and accessible from the producer.
- Prometheus should be configured to scrape the exposed metrics endpoint.
- **Multi-tenant environments**: The tool automatically extracts tenant information from user identifiers and ensures proper separation of metrics across tenants.
- **Bucket name collision handling**: Buckets with identical names from different tenants are properly isolated in all metrics.
- **Latency units**: All latency histograms use seconds as the unit, converted from the millisecond `total_time` field in log entries.
- **Memory efficiency**: The dedicated storage architecture ensures minimal memory usage by storing only enabled metric types.
- **Error visibility**: Error metrics always maintain visibility by reporting 0 when no errors occur, essential for proper monitoring.
- Sidecar injection is supported via a mutating webhook (see related documentation for Kubernetes usage).

> This README will be updated as new features and improvements are introduced. Contributions and feedback are welcome!