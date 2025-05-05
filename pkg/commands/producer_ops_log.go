// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and prysm contributors
//
// SPDX-License-Identifier: Apache-2.0

package commands

import (
	"fmt"
	"os"

	"github.com/cobaltcore-dev/prysm/pkg/producers/opslog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var (
	opsLogFilePath             string
	opsTruncateLogOnStart      bool
	opsSocketPath              string
	opsNatsURL                 string
	opsNatsSubject             string
	opsNatsMetricsSubject      string
	opsLogToStdout             bool
	opsLogRetentionDays        int
	opsMaxLogFileSize          int64
	opsPromEnabled             bool
	opsPromPort                int
	opsIgnoreAnonymousRequests bool
	opsPromIntervalSeconds     int

	// MetricsConfig-related flags
	opsTrackRequestsByIP                   bool
	opsTrackBytesSentByIP                  bool
	opsTrackBytesReceivedByIP              bool
	opsTrackErrorsByIP                     bool
	opsTrackErrorsByUser                   bool
	opsTrackRequestsByIPBucketMethodTenant bool
	opsTrackRequestsByMethod               bool
	opsTrackRequestsByOperation            bool
	opsTrackRequestsByStatus               bool
	opsTrackRequestsByBucket               bool
	opsTrackRequestsByUser                 bool
	opsTrackRequestsByTenant               bool
	opsTrackErrorsByBucket                 bool
	opsTrackErrorsByStatus                 bool
	opsTrackLatencyByUser                  bool
	opsTrackLatencyByBucket                bool
	opsTrackLatencyByMethod                bool
	opsTrackLatencyByTenant                bool
	opsTrackBytesSentByUser                bool
	opsTrackBytesReceivedByUser            bool
	opsTrackBytesSentByBucket              bool
	opsTrackBytesReceivedByBucket          bool
)

var opsLogCmd = &cobra.Command{
	Use:   "ops-log",
	Short: "Start the S3 operations logger",
	Long: `Start the S3 operations logger.

Note: Before using this command, ensure that RGW is configured to log S3 operations with the necessary details.

To enable RGW ops log to file feature, run the following commands:

  # ceph config set global rgw_ops_log_rados false
  # ceph config set global rgw_ops_log_file_path '/var/log/ceph/ops-log-$cluster-$name.log'
  # ceph config set global rgw_enable_ops_log true

Then restart all RadosGW daemons:

  # ceph orch ps
  # ceph orch daemon restart <rgw>

Following this configuration change, the RadosGW will log operations to the file /var/log/ceph/ceph-rgw-ops.json.log.`,
	Run: func(cmd *cobra.Command, args []string) {
		config := opslog.OpsLogConfig{
			LogFilePath:               opsLogFilePath,
			TruncateLogOnStart:        opsTruncateLogOnStart,
			SocketPath:                opsSocketPath,
			NatsURL:                   opsNatsURL,
			NatsSubject:               opsNatsSubject,
			NatsMetricsSubject:        opsNatsMetricsSubject,
			LogToStdout:               opsLogToStdout,
			LogRetentionDays:          opsLogRetentionDays,
			MaxLogFileSize:            opsMaxLogFileSize,
			Prometheus:                opsPromEnabled,
			PrometheusPort:            opsPromPort,
			IgnoreAnonymousRequests:   opsIgnoreAnonymousRequests,
			PrometheusIntervalSeconds: opsPromIntervalSeconds,
			MetricsConfig: opslog.MetricsConfig{
				TrackRequestsByIP:                   opsTrackRequestsByIP,
				TrackBytesSentByIP:                  opsTrackBytesSentByIP,
				TrackBytesReceivedByIP:              opsTrackBytesReceivedByIP,
				TrackBytesSentByUser:                opsTrackBytesSentByUser,
				TrackBytesReceivedByUser:            opsTrackBytesReceivedByUser,
				TrackBytesSentByBucket:              opsTrackBytesSentByBucket,
				TrackBytesReceivedByBucket:          opsTrackBytesReceivedByBucket,
				TrackErrorsByIP:                     opsTrackErrorsByIP,
				TrackErrorsByUser:                   opsTrackErrorsByUser,
				TrackRequestsByIPBucketMethodTenant: opsTrackRequestsByIPBucketMethodTenant,
				TrackRequestsByMethod:               opsTrackRequestsByMethod,
				TrackRequestsByOperation:            opsTrackRequestsByOperation,
				TrackRequestsByStatus:               opsTrackRequestsByStatus,
				TrackRequestsByBucket:               opsTrackRequestsByBucket,
				TrackRequestsByUser:                 opsTrackRequestsByUser,
				TrackRequestsByTenant:               opsTrackRequestsByTenant,
				TrackErrorsByBucket:                 opsTrackErrorsByBucket,
				TrackErrorsByStatus:                 opsTrackErrorsByStatus,
				TrackLatencyByUser:                  opsTrackLatencyByUser,
				TrackLatencyByBucket:                opsTrackLatencyByBucket,
				TrackLatencyByMethod:                opsTrackLatencyByMethod,
				TrackLatencyByTenant:                opsTrackLatencyByTenant,
			},
		}

		config = mergeOpsLogConfigWithEnv(config)

		config.UseNats = config.NatsURL != ""

		event := log.Info()
		event.Bool("use_nats", config.UseNats)
		if config.UseNats {
			event.Str("nats_url", config.NatsURL)
			event.Str("nats_subject", config.NatsSubject)
			event.Str("nats_metrics_subject", config.NatsMetricsSubject)
		}

		if config.LogFilePath != "" {
			event.Str("log_file_path", config.LogFilePath)
		}

		if config.SocketPath != "" {
			event.Str("socket_path", config.SocketPath)
		}

		if config.LogToStdout {
			event.Bool("log_to_stdout", config.LogToStdout)
		}

		event.Int("log_retention_days", config.LogRetentionDays)
		event.Int64("max_log_file_size", config.MaxLogFileSize)

		event.Bool("prometheus_enabled", config.Prometheus)
		if config.Prometheus {
			event.Int("prometheus_port", config.PrometheusPort)
		}

		// Debugging all tracking options from config.MetricsConfig
		event.Bool("track_requests_by_ip", config.MetricsConfig.TrackRequestsByIP)
		event.Bool("track_bytes_sent_by_ip", config.MetricsConfig.TrackBytesSentByIP)
		event.Bool("track_bytes_received_by_ip", config.MetricsConfig.TrackBytesReceivedByIP)
		event.Bool("track_bytes_sent_by_user", config.MetricsConfig.TrackBytesSentByUser)
		event.Bool("track_bytes_received_by_user", config.MetricsConfig.TrackBytesReceivedByUser)
		event.Bool("track_bytes_sent_by_bucket", config.MetricsConfig.TrackBytesSentByBucket)
		event.Bool("track_bytes_received_by_bucket", config.MetricsConfig.TrackBytesReceivedByBucket)
		event.Bool("track_errors_by_ip", config.MetricsConfig.TrackErrorsByIP)
		event.Bool("track_errors_by_user", config.MetricsConfig.TrackErrorsByUser)
		event.Bool("track_requests_by_method", config.MetricsConfig.TrackRequestsByMethod)
		event.Bool("track_requests_by_operation", config.MetricsConfig.TrackRequestsByOperation)
		event.Bool("track_requests_by_status", config.MetricsConfig.TrackRequestsByStatus)
		event.Bool("track_requests_by_bucket", config.MetricsConfig.TrackRequestsByBucket)
		event.Bool("track_requests_by_user", config.MetricsConfig.TrackRequestsByUser)
		event.Bool("track_requests_by_ip_bucket_method_tenant", config.MetricsConfig.TrackRequestsByIPBucketMethodTenant)
		event.Bool("track_requests_by_tenant", config.MetricsConfig.TrackRequestsByTenant)
		event.Bool("track_errors_by_bucket", config.MetricsConfig.TrackErrorsByBucket)
		event.Bool("track_errors_by_status", config.MetricsConfig.TrackErrorsByStatus)
		event.Bool("track_latency_by_user", config.MetricsConfig.TrackLatencyByUser)
		event.Bool("track_latency_by_bucket", config.MetricsConfig.TrackLatencyByBucket)
		event.Bool("track_latency_by_method", config.MetricsConfig.TrackLatencyByMethod)
		event.Bool("track_latency_by_tenant", config.MetricsConfig.TrackLatencyByTenant)

		event.Msg("OpsLog configuration initialized")

		validateOpsLogConfig(config)

		if config.SocketPath != "" {
			opslog.StartSocketOpsLogger(config)
		} else {
			opslog.StartFileOpsLogger(config)
		}
	},
}

func mergeOpsLogConfigWithEnv(cfg opslog.OpsLogConfig) opslog.OpsLogConfig {
	cfg.LogFilePath = getEnv("LOG_FILE_PATH", cfg.LogFilePath)
	cfg.TruncateLogOnStart = getEnvBool("TRUNCATE_LOG_ON_START", cfg.TruncateLogOnStart)
	cfg.SocketPath = getEnv("SOCKET_PATH", cfg.SocketPath)
	cfg.NatsURL = getEnv("NATS_URL", cfg.NatsURL)
	cfg.NatsSubject = getEnv("NATS_SUBJECT", cfg.NatsSubject)
	cfg.NatsMetricsSubject = getEnv("NATS_METRICS_SUBJECT", cfg.NatsMetricsSubject)
	cfg.LogToStdout = getEnvBool("LOG_TO_STDOUT", cfg.LogToStdout)
	cfg.LogRetentionDays = getEnvInt("LOG_RETENTION_DAYS", cfg.LogRetentionDays)
	cfg.MaxLogFileSize = getEnvInt64("MAX_LOG_FILE_SIZE", cfg.MaxLogFileSize)
	cfg.PrometheusPort = getEnvInt("PROMETHEUS_PORT", cfg.PrometheusPort)
	cfg.PodName = getEnv("POD_NAME", cfg.PodName)
	cfg.IgnoreAnonymousRequests = getEnvBool("IGNORE_ANONYMOUS_REQUESTS", cfg.IgnoreAnonymousRequests)
	cfg.PrometheusIntervalSeconds = getEnvInt("PROMETHEUS_INTERVAL", cfg.PrometheusIntervalSeconds)

	// MetricsConfig environment variables
	cfg.MetricsConfig.TrackRequestsByIP = getEnvBool("TRACK_REQUESTS_BY_IP", cfg.MetricsConfig.TrackRequestsByIP)
	cfg.MetricsConfig.TrackBytesSentByIP = getEnvBool("TRACK_BYTES_SENT_BY_IP", cfg.MetricsConfig.TrackBytesSentByIP)
	cfg.MetricsConfig.TrackBytesReceivedByIP = getEnvBool("TRACK_BYTES_RECEIVED_BY_IP", cfg.MetricsConfig.TrackBytesReceivedByIP)
	cfg.MetricsConfig.TrackBytesSentByUser = getEnvBool("TRACK_BYTES_SENT_BY_USER", cfg.MetricsConfig.TrackBytesSentByUser)
	cfg.MetricsConfig.TrackBytesReceivedByUser = getEnvBool("TRACK_BYTES_RECEIVED_BY_USER", cfg.MetricsConfig.TrackBytesReceivedByUser)
	cfg.MetricsConfig.TrackBytesSentByBucket = getEnvBool("TRACK_BYTES_SENT_BY_BUCKET", cfg.MetricsConfig.TrackBytesSentByBucket)
	cfg.MetricsConfig.TrackBytesReceivedByBucket = getEnvBool("TRACK_BYTES_RECEIVED_BY_BUCKET", cfg.MetricsConfig.TrackBytesReceivedByBucket)
	cfg.MetricsConfig.TrackErrorsByIP = getEnvBool("TRACK_ERRORS_BY_IP", cfg.MetricsConfig.TrackErrorsByIP)
	cfg.MetricsConfig.TrackErrorsByUser = getEnvBool("TRACK_ERRORS_BY_USER", cfg.MetricsConfig.TrackErrorsByUser)
	cfg.MetricsConfig.TrackRequestsByMethod = getEnvBool("TRACK_REQUESTS_BY_METHOD", cfg.MetricsConfig.TrackRequestsByMethod)
	cfg.MetricsConfig.TrackRequestsByOperation = getEnvBool("TRACK_REQUESTS_BY_OPERATION", cfg.MetricsConfig.TrackRequestsByOperation)
	cfg.MetricsConfig.TrackRequestsByStatus = getEnvBool("TRACK_REQUESTS_BY_STATUS", cfg.MetricsConfig.TrackRequestsByStatus)
	cfg.MetricsConfig.TrackRequestsByBucket = getEnvBool("TRACK_REQUESTS_BY_BUCKET", cfg.MetricsConfig.TrackRequestsByBucket)
	cfg.MetricsConfig.TrackRequestsByUser = getEnvBool("TRACK_REQUESTS_BY_USER", cfg.MetricsConfig.TrackRequestsByUser)
	cfg.MetricsConfig.TrackRequestsByIPBucketMethodTenant = getEnvBool("TRACK_REQUESTS_BY_IP_BUCKET_METHOD_TENANT", cfg.MetricsConfig.TrackRequestsByIPBucketMethodTenant)
	cfg.MetricsConfig.TrackRequestsByTenant = getEnvBool("TRACK_REQUESTS_BY_TENANT", cfg.MetricsConfig.TrackRequestsByTenant)
	cfg.MetricsConfig.TrackErrorsByBucket = getEnvBool("TRACK_ERRORS_BY_BUCKET", cfg.MetricsConfig.TrackErrorsByBucket)
	cfg.MetricsConfig.TrackErrorsByStatus = getEnvBool("TRACK_ERRORS_BY_STATUS", cfg.MetricsConfig.TrackErrorsByStatus)
	cfg.MetricsConfig.TrackLatencyByUser = getEnvBool("TRACK_LATENCY_BY_USER", cfg.MetricsConfig.TrackLatencyByUser)
	cfg.MetricsConfig.TrackLatencyByBucket = getEnvBool("TRACK_LATENCY_BY_BUCKET", cfg.MetricsConfig.TrackLatencyByBucket)
	cfg.MetricsConfig.TrackLatencyByTenant = getEnvBool("TRACK_LATENCY_BY_TENANT", cfg.MetricsConfig.TrackLatencyByTenant)
	cfg.MetricsConfig.TrackLatencyByMethod = getEnvBool("TRACK_LATENCY_BY_METHOD", cfg.MetricsConfig.TrackLatencyByMethod)

	return cfg
}

func init() {
	opsLogCmd.Flags().StringVar(&opsLogFilePath, "log-file", "/var/log/ceph/ceph-rgw-ops.json.log", "Path to the S3 operations log file")
	opsLogCmd.Flags().BoolVar(&opsTruncateLogOnStart, "truncate-log-on-start", true, "Truncate ops log file at startup to avoid duplicate processing")
	opsLogCmd.Flags().StringVar(&opsSocketPath, "socket-path", "", "Path to the Unix domain socket")
	opsLogCmd.Flags().StringVar(&opsNatsURL, "nats-url", "", "NATS server URL")
	opsLogCmd.Flags().StringVar(&opsNatsSubject, "nats-subject", "rgw.s3.ops", "NATS subject to publish results")
	opsLogCmd.Flags().StringVar(&opsNatsMetricsSubject, "nats-metrics-subject", "rgw.s3.ops.aggregated.metrics", "NATS subject to publish aggregated metrics")
	opsLogCmd.Flags().BoolVar(&opsLogToStdout, "log-to-stdout", false, "Log operations to stdout instead of a file")
	opsLogCmd.Flags().IntVar(&opsLogRetentionDays, "log-retention-days", 1, "Number of days to retain old log files")
	opsLogCmd.Flags().Int64Var(&opsMaxLogFileSize, "max-log-file-size", 10, "Maximum log file size in MB before rotation (e.g., 10 for 10 MB)")
	opsLogCmd.Flags().BoolVar(&opsPromEnabled, "prometheus", false, "Enable Prometheus metrics")
	opsLogCmd.Flags().IntVar(&opsPromPort, "prometheus-port", 8080, "Prometheus metrics port")
	opsLogCmd.Flags().BoolVar(&opsIgnoreAnonymousRequests, "ignore-anonymous-requests", true, "Ignore anonymous requests")
	opsLogCmd.Flags().IntVar(&opsPromIntervalSeconds, "prometheus-interval", 60, "Prometheus metrics update interval in seconds")

	// Metrics Tracking Flags (All Disabled by Default)
	opsLogCmd.Flags().BoolVar(&opsTrackRequestsByIP, "track-requests-by-ip", false, "Track requests by IP")
	opsLogCmd.Flags().BoolVar(&opsTrackBytesSentByIP, "track-bytes-sent-by-ip", false, "Track bytes sent by IP")
	opsLogCmd.Flags().BoolVar(&opsTrackBytesReceivedByIP, "track-bytes-received-by-ip", false, "Track bytes received by IP")
	opsLogCmd.Flags().BoolVar(&opsTrackBytesSentByUser, "track-bytes-sent-by-user", false, "Track bytes sent per user")
	opsLogCmd.Flags().BoolVar(&opsTrackBytesReceivedByUser, "track-bytes-received-by-user", false, "Track bytes received per user")
	opsLogCmd.Flags().BoolVar(&opsTrackBytesSentByBucket, "track-bytes-sent-by-bucket", false, "Track bytes sent per bucket")
	opsLogCmd.Flags().BoolVar(&opsTrackBytesReceivedByBucket, "track-bytes-received-by-bucket", false, "Track bytes received per bucket")
	opsLogCmd.Flags().BoolVar(&opsTrackErrorsByIP, "track-errors-by-ip", false, "Track errors by IP")
	opsLogCmd.Flags().BoolVar(&opsTrackErrorsByUser, "track-errors-by-user", false, "Track errors per user")
	opsLogCmd.Flags().BoolVar(&opsTrackRequestsByMethod, "track-requests-by-method", false, "Track requests by HTTP method")
	opsLogCmd.Flags().BoolVar(&opsTrackRequestsByOperation, "track-requests-by-operation", false, "Track requests by operation")
	opsLogCmd.Flags().BoolVar(&opsTrackRequestsByStatus, "track-requests-by-status", false, "Track requests by HTTP status code")
	opsLogCmd.Flags().BoolVar(&opsTrackRequestsByBucket, "track-requests-by-bucket", false, "Track requests by bucket")
	opsLogCmd.Flags().BoolVar(&opsTrackRequestsByUser, "track-requests-by-user", false, "Track requests by user")
	opsLogCmd.Flags().BoolVar(&opsTrackRequestsByIPBucketMethodTenant, "track-requests-by-ip-bucket-method-tenant", false, "Track requests by IP, bucket, HTTP method and tenant")
	opsLogCmd.Flags().BoolVar(&opsTrackRequestsByTenant, "track-requests-by-tenant", false, "Track requests by tenant")
	opsLogCmd.Flags().BoolVar(&opsTrackErrorsByBucket, "track-errors-by-bucket", false, "Track errors per bucket")
	opsLogCmd.Flags().BoolVar(&opsTrackErrorsByStatus, "track-errors-by-status", false, "Track errors per HTTP status")
	opsLogCmd.Flags().BoolVar(&opsTrackLatencyByMethod, "track-latency-by-method", false, "Track latency per method")
	opsLogCmd.Flags().BoolVar(&opsTrackLatencyByUser, "track-latency-by-user", false, "Track latency per user")
	opsLogCmd.Flags().BoolVar(&opsTrackLatencyByBucket, "track-latency-by-bucket", false, "Track latency per bucket")
	opsLogCmd.Flags().BoolVar(&opsTrackLatencyByTenant, "track-latency-by-tenant", false, "Track latency per tenant")
}

func validateOpsLogConfig(config opslog.OpsLogConfig) {
	missingParams := false

	if config.LogFilePath == "" && config.SocketPath == "" {
		fmt.Println("Warning: --log-file or LOG_FILE_PATH or --socket-path or SOCKET_PATH must be set")
		missingParams = true
	}

	if missingParams {
		fmt.Println("One or more required parameters are missing. Please provide them through flags or environment variables.")
		os.Exit(1)
	}
}
