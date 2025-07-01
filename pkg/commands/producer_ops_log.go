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

	// Shortcut config
	opsTrackEverything bool

	// Request metrics flags
	opsTrackRequestsDetailed  bool
	opsTrackRequestsPerUser   bool
	opsTrackRequestsPerBucket bool
	opsTrackRequestsPerTenant bool

	// Method-based request flags
	opsTrackRequestsByMethodDetailed  bool
	opsTrackRequestsByMethodPerUser   bool
	opsTrackRequestsByMethodPerBucket bool
	opsTrackRequestsByMethodPerTenant bool
	opsTrackRequestsByMethodGlobal    bool

	// Operation-based request flags
	opsTrackRequestsByOperationDetailed  bool
	opsTrackRequestsByOperationPerUser   bool
	opsTrackRequestsByOperationPerBucket bool
	opsTrackRequestsByOperationPerTenant bool
	opsTrackRequestsByOperationGlobal    bool

	// Status-based request flags
	opsTrackRequestsByStatusDetailed  bool
	opsTrackRequestsByStatusPerUser   bool
	opsTrackRequestsByStatusPerBucket bool
	opsTrackRequestsByStatusPerTenant bool

	// Bytes metrics flags
	opsTrackBytesSentDetailed  bool
	opsTrackBytesSentPerUser   bool
	opsTrackBytesSentPerBucket bool
	opsTrackBytesSentPerTenant bool

	opsTrackBytesReceivedDetailed  bool
	opsTrackBytesReceivedPerUser   bool
	opsTrackBytesReceivedPerBucket bool
	opsTrackBytesReceivedPerTenant bool

	// Error metrics flags
	opsTrackErrorsDetailed  bool
	opsTrackErrorsPerUser   bool
	opsTrackErrorsPerBucket bool
	opsTrackErrorsPerTenant bool
	opsTrackErrorsPerStatus bool
	opsTrackErrorsByIP      bool

	// IP-based metrics flags
	opsTrackRequestsByIPDetailed           bool
	opsTrackRequestsByIPPerTenant          bool
	opsTrackRequestsByIPBucketMethodTenant bool
	opsTrackRequestsByIPGlobalPerTenant    bool

	opsTrackBytesSentByIPDetailed        bool
	opsTrackBytesSentByIPPerTenant       bool
	opsTrackBytesSentByIPGlobalPerTenant bool

	opsTrackBytesReceivedByIPDetailed        bool
	opsTrackBytesReceivedByIPPerTenant       bool
	opsTrackBytesReceivedByIPGlobalPerTenant bool

	// Latency metrics flags
	opsTrackLatencyDetailed           bool
	opsTrackLatencyPerUser            bool
	opsTrackLatencyPerBucket          bool
	opsTrackLatencyPerTenant          bool
	opsTrackLatencyPerMethod          bool
	opsTrackLatencyPerBucketAndMethod bool
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
				// Shortcut config
				TrackEverything: opsTrackEverything,

				// Request metrics
				TrackRequestsDetailed:  opsTrackRequestsDetailed,
				TrackRequestsPerUser:   opsTrackRequestsPerUser,
				TrackRequestsPerBucket: opsTrackRequestsPerBucket,
				TrackRequestsPerTenant: opsTrackRequestsPerTenant,

				// Method-based requests
				TrackRequestsByMethodDetailed:  opsTrackRequestsByMethodDetailed,
				TrackRequestsByMethodPerUser:   opsTrackRequestsByMethodPerUser,
				TrackRequestsByMethodPerBucket: opsTrackRequestsByMethodPerBucket,
				TrackRequestsByMethodPerTenant: opsTrackRequestsByMethodPerTenant,
				TrackRequestsByMethodGlobal:    opsTrackRequestsByMethodGlobal,

				// Operation-based requests
				TrackRequestsByOperationDetailed:  opsTrackRequestsByOperationDetailed,
				TrackRequestsByOperationPerUser:   opsTrackRequestsByOperationPerUser,
				TrackRequestsByOperationPerBucket: opsTrackRequestsByOperationPerBucket,
				TrackRequestsByOperationPerTenant: opsTrackRequestsByOperationPerTenant,
				TrackRequestsByOperationGlobal:    opsTrackRequestsByOperationGlobal,

				// Status-based requests
				TrackRequestsByStatusDetailed:  opsTrackRequestsByStatusDetailed,
				TrackRequestsByStatusPerUser:   opsTrackRequestsByStatusPerUser,
				TrackRequestsByStatusPerBucket: opsTrackRequestsByStatusPerBucket,
				TrackRequestsByStatusPerTenant: opsTrackRequestsByStatusPerTenant,

				// Bytes metrics
				TrackBytesSentDetailed:  opsTrackBytesSentDetailed,
				TrackBytesSentPerUser:   opsTrackBytesSentPerUser,
				TrackBytesSentPerBucket: opsTrackBytesSentPerBucket,
				TrackBytesSentPerTenant: opsTrackBytesSentPerTenant,

				TrackBytesReceivedDetailed:  opsTrackBytesReceivedDetailed,
				TrackBytesReceivedPerUser:   opsTrackBytesReceivedPerUser,
				TrackBytesReceivedPerBucket: opsTrackBytesReceivedPerBucket,
				TrackBytesReceivedPerTenant: opsTrackBytesReceivedPerTenant,

				// Error metrics
				TrackErrorsDetailed:  opsTrackErrorsDetailed,
				TrackErrorsPerUser:   opsTrackErrorsPerUser,
				TrackErrorsPerBucket: opsTrackErrorsPerBucket,
				TrackErrorsPerTenant: opsTrackErrorsPerTenant,
				TrackErrorsPerStatus: opsTrackErrorsPerStatus,

				// IP-based metrics
				TrackRequestsByIPDetailed:           opsTrackRequestsByIPDetailed,
				TrackRequestsByIPPerTenant:          opsTrackRequestsByIPPerTenant,
				TrackRequestsByIPBucketMethodTenant: opsTrackRequestsByIPBucketMethodTenant,
				TrackRequestsByIPGlobalPerTenant:    opsTrackRequestsByIPGlobalPerTenant,

				TrackBytesSentByIPDetailed:        opsTrackBytesSentByIPDetailed,
				TrackBytesSentByIPPerTenant:       opsTrackBytesSentByIPPerTenant,
				TrackBytesSentByIPGlobalPerTenant: opsTrackBytesSentByIPGlobalPerTenant,

				TrackBytesReceivedByIPDetailed:        opsTrackBytesReceivedByIPDetailed,
				TrackBytesReceivedByIPPerTenant:       opsTrackBytesReceivedByIPPerTenant,
				TrackBytesReceivedByIPGlobalPerTenant: opsTrackBytesReceivedByIPGlobalPerTenant,

				TrackErrorsByIP: opsTrackErrorsByIP,

				// Latency metrics
				TrackLatencyDetailed:           opsTrackLatencyDetailed,
				TrackLatencyPerUser:            opsTrackLatencyPerUser,
				TrackLatencyPerBucket:          opsTrackLatencyPerBucket,
				TrackLatencyPerTenant:          opsTrackLatencyPerTenant,
				TrackLatencyPerMethod:          opsTrackLatencyPerMethod,
				TrackLatencyPerBucketAndMethod: opsTrackLatencyPerBucketAndMethod,
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
		event.Bool("track_everything", config.MetricsConfig.TrackEverything)
		event.Bool("track_requests_detailed", config.MetricsConfig.TrackRequestsDetailed)
		event.Bool("track_requests_per_user", config.MetricsConfig.TrackRequestsPerUser)
		event.Bool("track_bytes_sent_detailed", config.MetricsConfig.TrackBytesSentDetailed)
		event.Bool("track_errors_detailed", config.MetricsConfig.TrackErrorsDetailed)
		event.Bool("track_latency_detailed", config.MetricsConfig.TrackLatencyDetailed)

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

	// Shortcut config
	cfg.MetricsConfig.TrackEverything = getEnvBool("TRACK_EVERYTHING", cfg.MetricsConfig.TrackEverything)

	// Request metrics environment variables
	cfg.MetricsConfig.TrackRequestsDetailed = getEnvBool("TRACK_REQUESTS_DETAILED", cfg.MetricsConfig.TrackRequestsDetailed)
	cfg.MetricsConfig.TrackRequestsPerUser = getEnvBool("TRACK_REQUESTS_PER_USER", cfg.MetricsConfig.TrackRequestsPerUser)
	cfg.MetricsConfig.TrackRequestsPerBucket = getEnvBool("TRACK_REQUESTS_PER_BUCKET", cfg.MetricsConfig.TrackRequestsPerBucket)
	cfg.MetricsConfig.TrackRequestsPerTenant = getEnvBool("TRACK_REQUESTS_PER_TENANT", cfg.MetricsConfig.TrackRequestsPerTenant)

	// Method-based requests
	cfg.MetricsConfig.TrackRequestsByMethodDetailed = getEnvBool("TRACK_REQUESTS_BY_METHOD_DETAILED", cfg.MetricsConfig.TrackRequestsByMethodDetailed)
	cfg.MetricsConfig.TrackRequestsByMethodPerUser = getEnvBool("TRACK_REQUESTS_BY_METHOD_PER_USER", cfg.MetricsConfig.TrackRequestsByMethodPerUser)
	cfg.MetricsConfig.TrackRequestsByMethodPerBucket = getEnvBool("TRACK_REQUESTS_BY_METHOD_PER_BUCKET", cfg.MetricsConfig.TrackRequestsByMethodPerBucket)
	cfg.MetricsConfig.TrackRequestsByMethodPerTenant = getEnvBool("TRACK_REQUESTS_BY_METHOD_PER_TENANT", cfg.MetricsConfig.TrackRequestsByMethodPerTenant)
	cfg.MetricsConfig.TrackRequestsByMethodGlobal = getEnvBool("TRACK_REQUESTS_BY_METHOD_GLOBAL", cfg.MetricsConfig.TrackRequestsByMethodGlobal)

	// Operation-based requests
	cfg.MetricsConfig.TrackRequestsByOperationDetailed = getEnvBool("TRACK_REQUESTS_BY_OPERATION_DETAILED", cfg.MetricsConfig.TrackRequestsByOperationDetailed)
	cfg.MetricsConfig.TrackRequestsByOperationPerUser = getEnvBool("TRACK_REQUESTS_BY_OPERATION_PER_USER", cfg.MetricsConfig.TrackRequestsByOperationPerUser)
	cfg.MetricsConfig.TrackRequestsByOperationPerBucket = getEnvBool("TRACK_REQUESTS_BY_OPERATION_PER_BUCKET", cfg.MetricsConfig.TrackRequestsByOperationPerBucket)
	cfg.MetricsConfig.TrackRequestsByOperationPerTenant = getEnvBool("TRACK_REQUESTS_BY_OPERATION_PER_TENANT", cfg.MetricsConfig.TrackRequestsByOperationPerTenant)
	cfg.MetricsConfig.TrackRequestsByOperationGlobal = getEnvBool("TRACK_REQUESTS_BY_OPERATION_GLOBAL", cfg.MetricsConfig.TrackRequestsByOperationGlobal)

	// Status-based requests
	cfg.MetricsConfig.TrackRequestsByStatusDetailed = getEnvBool("TRACK_REQUESTS_BY_STATUS_DETAILED", cfg.MetricsConfig.TrackRequestsByStatusDetailed)
	cfg.MetricsConfig.TrackRequestsByStatusPerUser = getEnvBool("TRACK_REQUESTS_BY_STATUS_PER_USER", cfg.MetricsConfig.TrackRequestsByStatusPerUser)
	cfg.MetricsConfig.TrackRequestsByStatusPerBucket = getEnvBool("TRACK_REQUESTS_BY_STATUS_PER_BUCKET", cfg.MetricsConfig.TrackRequestsByStatusPerBucket)
	cfg.MetricsConfig.TrackRequestsByStatusPerTenant = getEnvBool("TRACK_REQUESTS_BY_STATUS_PER_TENANT", cfg.MetricsConfig.TrackRequestsByStatusPerTenant)

	// Bytes metrics
	cfg.MetricsConfig.TrackBytesSentDetailed = getEnvBool("TRACK_BYTES_SENT_DETAILED", cfg.MetricsConfig.TrackBytesSentDetailed)
	cfg.MetricsConfig.TrackBytesSentPerUser = getEnvBool("TRACK_BYTES_SENT_PER_USER", cfg.MetricsConfig.TrackBytesSentPerUser)
	cfg.MetricsConfig.TrackBytesSentPerBucket = getEnvBool("TRACK_BYTES_SENT_PER_BUCKET", cfg.MetricsConfig.TrackBytesSentPerBucket)
	cfg.MetricsConfig.TrackBytesSentPerTenant = getEnvBool("TRACK_BYTES_SENT_PER_TENANT", cfg.MetricsConfig.TrackBytesSentPerTenant)

	cfg.MetricsConfig.TrackBytesReceivedDetailed = getEnvBool("TRACK_BYTES_RECEIVED_DETAILED", cfg.MetricsConfig.TrackBytesReceivedDetailed)
	cfg.MetricsConfig.TrackBytesReceivedPerUser = getEnvBool("TRACK_BYTES_RECEIVED_PER_USER", cfg.MetricsConfig.TrackBytesReceivedPerUser)
	cfg.MetricsConfig.TrackBytesReceivedPerBucket = getEnvBool("TRACK_BYTES_RECEIVED_PER_BUCKET", cfg.MetricsConfig.TrackBytesReceivedPerBucket)
	cfg.MetricsConfig.TrackBytesReceivedPerTenant = getEnvBool("TRACK_BYTES_RECEIVED_PER_TENANT", cfg.MetricsConfig.TrackBytesReceivedPerTenant)

	// Error metrics
	cfg.MetricsConfig.TrackErrorsDetailed = getEnvBool("TRACK_ERRORS_DETAILED", cfg.MetricsConfig.TrackErrorsDetailed)
	cfg.MetricsConfig.TrackErrorsPerUser = getEnvBool("TRACK_ERRORS_PER_USER", cfg.MetricsConfig.TrackErrorsPerUser)
	cfg.MetricsConfig.TrackErrorsPerBucket = getEnvBool("TRACK_ERRORS_PER_BUCKET", cfg.MetricsConfig.TrackErrorsPerBucket)
	cfg.MetricsConfig.TrackErrorsPerTenant = getEnvBool("TRACK_ERRORS_PER_TENANT", cfg.MetricsConfig.TrackErrorsPerTenant)
	cfg.MetricsConfig.TrackErrorsPerStatus = getEnvBool("TRACK_ERRORS_PER_STATUS", cfg.MetricsConfig.TrackErrorsPerStatus)
	cfg.MetricsConfig.TrackErrorsByIP = getEnvBool("TRACK_ERRORS_BY_IP", cfg.MetricsConfig.TrackErrorsByIP)

	// IP-based metrics
	cfg.MetricsConfig.TrackRequestsByIPDetailed = getEnvBool("TRACK_REQUESTS_BY_IP_DETAILED", cfg.MetricsConfig.TrackRequestsByIPDetailed)
	cfg.MetricsConfig.TrackRequestsByIPPerTenant = getEnvBool("TRACK_REQUESTS_BY_IP_PER_TENANT", cfg.MetricsConfig.TrackRequestsByIPPerTenant)
	cfg.MetricsConfig.TrackRequestsByIPBucketMethodTenant = getEnvBool("TRACK_REQUESTS_BY_IP_BUCKET_METHOD_TENANT", cfg.MetricsConfig.TrackRequestsByIPBucketMethodTenant)
	cfg.MetricsConfig.TrackRequestsByIPGlobalPerTenant = getEnvBool("TRACK_REQUESTS_BY_IP_GLOBAL_PER_TENANT", cfg.MetricsConfig.TrackRequestsByIPGlobalPerTenant)

	cfg.MetricsConfig.TrackBytesSentByIPDetailed = getEnvBool("TRACK_BYTES_SENT_BY_IP_DETAILED", cfg.MetricsConfig.TrackBytesSentByIPDetailed)
	cfg.MetricsConfig.TrackBytesSentByIPPerTenant = getEnvBool("TRACK_BYTES_SENT_BY_IP_PER_TENANT", cfg.MetricsConfig.TrackBytesSentByIPPerTenant)
	cfg.MetricsConfig.TrackBytesSentByIPGlobalPerTenant = getEnvBool("TRACK_BYTES_SENT_BY_IP_GLOBAL_PER_TENANT", cfg.MetricsConfig.TrackBytesSentByIPGlobalPerTenant)

	cfg.MetricsConfig.TrackBytesReceivedByIPDetailed = getEnvBool("TRACK_BYTES_RECEIVED_BY_IP_DETAILED", cfg.MetricsConfig.TrackBytesReceivedByIPDetailed)
	cfg.MetricsConfig.TrackBytesReceivedByIPPerTenant = getEnvBool("TRACK_BYTES_RECEIVED_BY_IP_PER_TENANT", cfg.MetricsConfig.TrackBytesReceivedByIPPerTenant)
	cfg.MetricsConfig.TrackBytesReceivedByIPGlobalPerTenant = getEnvBool("TRACK_BYTES_RECEIVED_BY_IP_GLOBAL_PER_TENANT", cfg.MetricsConfig.TrackBytesReceivedByIPGlobalPerTenant)

	// Latency metrics
	cfg.MetricsConfig.TrackLatencyDetailed = getEnvBool("TRACK_LATENCY_DETAILED", cfg.MetricsConfig.TrackLatencyDetailed)
	cfg.MetricsConfig.TrackLatencyPerUser = getEnvBool("TRACK_LATENCY_PER_USER", cfg.MetricsConfig.TrackLatencyPerUser)
	cfg.MetricsConfig.TrackLatencyPerBucket = getEnvBool("TRACK_LATENCY_PER_BUCKET", cfg.MetricsConfig.TrackLatencyPerBucket)
	cfg.MetricsConfig.TrackLatencyPerTenant = getEnvBool("TRACK_LATENCY_PER_TENANT", cfg.MetricsConfig.TrackLatencyPerTenant)
	cfg.MetricsConfig.TrackLatencyPerMethod = getEnvBool("TRACK_LATENCY_PER_METHOD", cfg.MetricsConfig.TrackLatencyPerMethod)
	cfg.MetricsConfig.TrackLatencyPerBucketAndMethod = getEnvBool("TRACK_LATENCY_PER_BUCKET_AND_METHOD", cfg.MetricsConfig.TrackLatencyPerBucketAndMethod)

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

	// Shortcut flag
	opsLogCmd.Flags().BoolVar(&opsTrackEverything, "track-everything", false, "Enable detailed tracking for all metric types (efficient mode)")

	// Essential request metrics (most commonly used)
	opsLogCmd.Flags().BoolVar(&opsTrackRequestsDetailed, "track-requests-detailed", false, "Track detailed requests with full labels")
	opsLogCmd.Flags().BoolVar(&opsTrackRequestsPerUser, "track-requests-per-user", false, "Track requests aggregated per user")
	opsLogCmd.Flags().BoolVar(&opsTrackRequestsPerBucket, "track-requests-per-bucket", false, "Track requests aggregated per bucket")
	opsLogCmd.Flags().BoolVar(&opsTrackRequestsPerTenant, "track-requests-per-tenant", false, "Track requests aggregated per tenant")

	// Method-based request metrics
	opsLogCmd.Flags().BoolVar(&opsTrackRequestsByMethodDetailed, "track-requests-by-method-detailed", false, "Track detailed requests by HTTP method")
	opsLogCmd.Flags().BoolVar(&opsTrackRequestsByMethodPerUser, "track-requests-by-method-per-user", false, "Track requests by method per user")
	opsLogCmd.Flags().BoolVar(&opsTrackRequestsByMethodPerBucket, "track-requests-by-method-per-bucket", false, "Track requests by method per bucket")
	opsLogCmd.Flags().BoolVar(&opsTrackRequestsByMethodPerTenant, "track-requests-by-method-per-tenant", false, "Track requests by method per tenant")
	opsLogCmd.Flags().BoolVar(&opsTrackRequestsByMethodGlobal, "track-requests-by-method-global", false, "Track requests by method globally")

	// Operation-based request metrics
	opsLogCmd.Flags().BoolVar(&opsTrackRequestsByOperationDetailed, "track-requests-by-operation-detailed", false, "Track detailed requests by operation")
	opsLogCmd.Flags().BoolVar(&opsTrackRequestsByOperationPerUser, "track-requests-by-operation-per-user", false, "Track requests by operation per user")
	opsLogCmd.Flags().BoolVar(&opsTrackRequestsByOperationPerBucket, "track-requests-by-operation-per-bucket", false, "Track requests by operation per bucket")
	opsLogCmd.Flags().BoolVar(&opsTrackRequestsByOperationPerTenant, "track-requests-by-operation-per-tenant", false, "Track requests by operation per tenant")
	opsLogCmd.Flags().BoolVar(&opsTrackRequestsByOperationGlobal, "track-requests-by-operation-global", false, "Track requests by operation globally")

	// Status-based request metrics
	opsLogCmd.Flags().BoolVar(&opsTrackRequestsByStatusDetailed, "track-requests-by-status-detailed", false, "Track detailed requests by status")
	opsLogCmd.Flags().BoolVar(&opsTrackRequestsByStatusPerUser, "track-requests-by-status-per-user", false, "Track requests by status per user")
	opsLogCmd.Flags().BoolVar(&opsTrackRequestsByStatusPerBucket, "track-requests-by-status-per-bucket", false, "Track requests by status per bucket")
	opsLogCmd.Flags().BoolVar(&opsTrackRequestsByStatusPerTenant, "track-requests-by-status-per-tenant", false, "Track requests by status per tenant")

	// Bytes metrics
	opsLogCmd.Flags().BoolVar(&opsTrackBytesSentDetailed, "track-bytes-sent-detailed", false, "Track detailed bytes sent")
	opsLogCmd.Flags().BoolVar(&opsTrackBytesSentPerUser, "track-bytes-sent-per-user", false, "Track bytes sent per user")
	opsLogCmd.Flags().BoolVar(&opsTrackBytesSentPerBucket, "track-bytes-sent-per-bucket", false, "Track bytes sent per bucket")
	opsLogCmd.Flags().BoolVar(&opsTrackBytesSentPerTenant, "track-bytes-sent-per-tenant", false, "Track bytes sent per tenant")

	opsLogCmd.Flags().BoolVar(&opsTrackBytesReceivedDetailed, "track-bytes-received-detailed", false, "Track detailed bytes received")
	opsLogCmd.Flags().BoolVar(&opsTrackBytesReceivedPerUser, "track-bytes-received-per-user", false, "Track bytes received per user")
	opsLogCmd.Flags().BoolVar(&opsTrackBytesReceivedPerBucket, "track-bytes-received-per-bucket", false, "Track bytes received per bucket")
	opsLogCmd.Flags().BoolVar(&opsTrackBytesReceivedPerTenant, "track-bytes-received-per-tenant", false, "Track bytes received per tenant")

	// Error metrics
	opsLogCmd.Flags().BoolVar(&opsTrackErrorsDetailed, "track-errors-detailed", false, "Track detailed errors")
	opsLogCmd.Flags().BoolVar(&opsTrackErrorsPerUser, "track-errors-per-user", false, "Track errors per user")
	opsLogCmd.Flags().BoolVar(&opsTrackErrorsPerBucket, "track-errors-per-bucket", false, "Track errors per bucket")
	opsLogCmd.Flags().BoolVar(&opsTrackErrorsPerTenant, "track-errors-per-tenant", false, "Track errors per tenant")
	opsLogCmd.Flags().BoolVar(&opsTrackErrorsPerStatus, "track-errors-per-status", false, "Track errors per HTTP status")

	// IP-based metrics
	opsLogCmd.Flags().BoolVar(&opsTrackRequestsByIPDetailed, "track-requests-by-ip-detailed", false, "Track requests by IP")
	opsLogCmd.Flags().BoolVar(&opsTrackRequestsByIPPerTenant, "track-requests-by-ip-per-tenant", false, "Track requests by IP per tenant")
	opsLogCmd.Flags().BoolVar(&opsTrackRequestsByIPBucketMethodTenant, "track-requests-by-ip-bucket-method-tenant", false, "Track requests by IP, bucket, method and tenant")
	opsLogCmd.Flags().BoolVar(&opsTrackRequestsByIPGlobalPerTenant, "track-requests-by-ip-global-per-tenant", false, "Track requests by IP globally per tenant")

	opsLogCmd.Flags().BoolVar(&opsTrackBytesSentByIPDetailed, "track-bytes-sent-by-ip-detailed", false, "Track bytes sent by IP")
	opsLogCmd.Flags().BoolVar(&opsTrackBytesSentByIPPerTenant, "track-bytes-sent-by-ip-per-tenant", false, "Track bytes sent by IP per tenant")
	opsLogCmd.Flags().BoolVar(&opsTrackBytesSentByIPGlobalPerTenant, "track-bytes-sent-by-ip-global-per-tenant", false, "Track bytes sent by IP globally per tenant")

	opsLogCmd.Flags().BoolVar(&opsTrackBytesReceivedByIPDetailed, "track-bytes-received-by-ip-detailed", false, "Track bytes received by IP")
	opsLogCmd.Flags().BoolVar(&opsTrackBytesReceivedByIPPerTenant, "track-bytes-received-by-ip-per-tenant", false, "Track bytes received by IP per tenant")
	opsLogCmd.Flags().BoolVar(&opsTrackBytesReceivedByIPGlobalPerTenant, "track-bytes-received-by-ip-global-per-tenant", false, "Track bytes received by IP globally per tenant")

	opsLogCmd.Flags().BoolVar(&opsTrackErrorsByIP, "track-errors-by-ip", false, "Track errors by IP")

	// Latency metrics
	opsLogCmd.Flags().BoolVar(&opsTrackLatencyDetailed, "track-latency-detailed", false, "Track detailed latency")
	opsLogCmd.Flags().BoolVar(&opsTrackLatencyPerUser, "track-latency-per-user", false, "Track latency per user")
	opsLogCmd.Flags().BoolVar(&opsTrackLatencyPerBucket, "track-latency-per-bucket", false, "Track latency per bucket")
	opsLogCmd.Flags().BoolVar(&opsTrackLatencyPerTenant, "track-latency-per-tenant", false, "Track latency per tenant")
	opsLogCmd.Flags().BoolVar(&opsTrackLatencyPerMethod, "track-latency-per-method", false, "Track latency per method")
	opsLogCmd.Flags().BoolVar(&opsTrackLatencyPerBucketAndMethod, "track-latency-per-bucket-and-method", false, "Track latency per bucket and method")
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
