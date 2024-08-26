// Copyright (C) 2024 Clyso GmbH
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package commands

import (
	"fmt"
	"os"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"gitlab.clyso.com/clyso/radosguard/pkg/producers/opslog"
)

var (
	opsLogFilePath string
	opsSocketPath  string
	opsNatsURL     string
	opsNatsSubject string
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
			LogFilePath: opsLogFilePath,
			SocketPath:  opsSocketPath,
			NatsURL:     opsNatsURL,
			NatsSubject: opsNatsSubject,
		}

		config = mergeOpsLogConfigWithEnv(config)

		config.UseNats = config.NatsURL != ""

		event := log.Info()
		event.Bool("use_nats", config.UseNats)
		if config.UseNats {
			event.Str("nats_url", config.NatsURL)
			event.Str("nats_subject", config.NatsSubject)
		}

		if config.LogFilePath != "" {
			event.Str("log_file_path", config.LogFilePath)
		}

		if config.SocketPath != "" {
			event.Str("socket_path", config.SocketPath)
		}

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

	return cfg
}

func init() {
	opsLogCmd.Flags().StringVar(&opsLogFilePath, "log-file", "/var/log/ceph/ceph-rgw-ops.json.log", "Path to the S3 operations log file")
	opsLogCmd.Flags().StringVar(&opsSocketPath, "socket-path", "", "Path to the Unix domain socket")
	opsLogCmd.Flags().StringVar(&opsNatsURL, "nats-url", "", "NATS server URL")
	opsLogCmd.Flags().StringVar(&opsNatsSubject, "nats-subject", "rgw.s3.ops", "NATS subject to publish results")
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
