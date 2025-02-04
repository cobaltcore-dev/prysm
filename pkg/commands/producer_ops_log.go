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
	opsLogFilePath        string
	opsSocketPath         string
	opsNatsURL            string
	opsNatsSubject        string
	opsNatsMetricsSubject string
	opsLogToStdout        bool
	opsLogRetentionDays   int
	opsMaxLogFileSize     int64
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
			LogFilePath:        opsLogFilePath,
			SocketPath:         opsSocketPath,
			NatsURL:            opsNatsURL,
			NatsSubject:        opsNatsSubject,
			NatsMetricsSubject: opsNatsMetricsSubject,
			LogToStdout:        opsLogToStdout,
			LogRetentionDays:   opsLogRetentionDays,
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
	cfg.SocketPath = getEnv("SOCKET_PATH", cfg.SocketPath)
	cfg.NatsURL = getEnv("NATS_URL", cfg.NatsURL)
	cfg.NatsSubject = getEnv("NATS_SUBJECT", cfg.NatsSubject)
	cfg.NatsMetricsSubject = getEnv("NATS_METRICS_SUBJECT", cfg.NatsMetricsSubject)
	cfg.LogToStdout = getEnvBool("LOG_TO_STDOUT", cfg.LogToStdout)
	cfg.LogRetentionDays = getEnvInt("LOG_RETENTION_DAYS", cfg.LogRetentionDays)
	cfg.MaxLogFileSize = getEnvInt64("MAX_LOG_FILE_SIZE", cfg.MaxLogFileSize)

	return cfg
}

func init() {
	opsLogCmd.Flags().StringVar(&opsLogFilePath, "log-file", "/var/log/ceph/ceph-rgw-ops.json.log", "Path to the S3 operations log file")
	opsLogCmd.Flags().StringVar(&opsSocketPath, "socket-path", "", "Path to the Unix domain socket")
	opsLogCmd.Flags().StringVar(&opsNatsURL, "nats-url", "", "NATS server URL")
	opsLogCmd.Flags().StringVar(&opsNatsSubject, "nats-subject", "rgw.s3.ops", "NATS subject to publish results")
	opsLogCmd.Flags().StringVar(&opsNatsMetricsSubject, "nats-metrics-subject", "rgw.s3.ops.aggregated.metrics", "NATS subject to publish aggregated metrics")
	opsLogCmd.Flags().BoolVar(&opsLogToStdout, "log-to-stdout", false, "Log operations to stdout instead of a file")
	opsLogCmd.Flags().IntVar(&opsLogRetentionDays, "log-retention-days", 1, "Number of days to retain old log files")
	opsLogCmd.Flags().Int64Var(&opsMaxLogFileSize, "max-log-file-size", 10, "Maximum log file size in MB before rotation (e.g., 10 for 10 MB)")
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
