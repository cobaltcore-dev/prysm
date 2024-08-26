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
	"gitlab.clyso.com/clyso/radosguard/pkg/producers/quotausagemonitor"
)

var (
	qumAdminURL          string
	qumAccessKey         string
	qumSecretKey         string
	qumNatsURL           string
	qumNatsSubject       string
	qumUseNats           bool
	qumNodeName          string
	qumInstanceID        string
	qumInterval          int
	qumQuotaUsagePercent float64
)

var quotaUsageMonitorCmd = &cobra.Command{
	Use:   "quota-usage-monitor",
	Short: "Quota usage monitor",
	Run: func(cmd *cobra.Command, args []string) {
		config := quotausagemonitor.QuotaUsageMonitorConfig{
			AdminURL:          qumAdminURL,
			AccessKey:         qumAccessKey,
			SecretKey:         qumSecretKey,
			NatsURL:           qumNatsURL,
			NatsSubject:       qumNatsSubject,
			UseNats:           qumUseNats,
			NodeName:          qumNodeName,
			InstanceID:        qumInstanceID,
			Interval:          qumInterval,
			QuotaUsagePercent: qumQuotaUsagePercent,
		}

		config = mergeQuotaUsageMonitorConfigWithEnv(config)
		config.UseNats = config.NatsURL != ""

		event := log.Info()
		event.Bool("use_nats", config.UseNats)
		if config.UseNats {
			event.Str("nats_url", config.NatsURL)
			event.Str("nats_subject", config.NatsSubject)
		}

		event.Str("node_name", config.NodeName)
		event.Str("instance_id", config.InstanceID)
		event.Int("interval_seconds", config.Interval)
		event.Float64("quota_usage_percent", config.QuotaUsagePercent)

		// Finalize the log message with the main message
		event.Msg("configuration_loaded")

		validateQuotaUsageMonitorConfig(config)

		quotausagemonitor.StartMonitoring(config)
	},
}

func mergeQuotaUsageMonitorConfigWithEnv(cfg quotausagemonitor.QuotaUsageMonitorConfig) quotausagemonitor.QuotaUsageMonitorConfig {
	cfg.AdminURL = getEnv("ADMIN_URL", cfg.AdminURL)
	cfg.AccessKey = getEnv("ACCESS_KEY", cfg.AccessKey)
	cfg.SecretKey = getEnv("SECRET_KEY", cfg.SecretKey)
	cfg.NatsURL = getEnv("NATS_URL", cfg.NatsURL)
	cfg.NatsSubject = getEnv("NATS_SUBJECT", cfg.NatsSubject)
	cfg.NodeName = getEnv("NODE_NAME", cfg.NodeName)
	cfg.InstanceID = getEnv("INSTANCE_ID", cfg.InstanceID)
	cfg.Interval = getEnvInt("INTERVAL", cfg.Interval)
	cfg.QuotaUsagePercent = getEnvFloat("QUOTA_USAGE_PERCENT", cfg.QuotaUsagePercent)

	return cfg
}

func init() {
	quotaUsageMonitorCmd.Flags().StringVar(&qumAdminURL, "admin-url", "", "Admin API URL")
	quotaUsageMonitorCmd.Flags().StringVar(&qumAccessKey, "access-key", "", "Access key for Admin API")
	quotaUsageMonitorCmd.Flags().StringVar(&qumSecretKey, "secret-key", "", "Secret key for Admin API")
	quotaUsageMonitorCmd.Flags().StringVar(&qumNatsURL, "nats-url", "", "NATS server URL")
	quotaUsageMonitorCmd.Flags().StringVar(&qumNatsSubject, "nats-subject", "user.quotas.usage", "NATS subject to publish quota usage")
	quotaUsageMonitorCmd.Flags().StringVar(&qumNodeName, "node-name", "", "Name of the node")
	quotaUsageMonitorCmd.Flags().StringVar(&qumInstanceID, "instance-id", "", "Instance ID")
	quotaUsageMonitorCmd.Flags().IntVar(&qumInterval, "interval", 10, "Interval in seconds between quota usage collections")
	quotaUsageMonitorCmd.Flags().Float64Var(&qumQuotaUsagePercent, "quota-usage-percent", 0, "Percentage of quota usage to monitor")

}

func validateQuotaUsageMonitorConfig(config quotausagemonitor.QuotaUsageMonitorConfig) {
	missingParams := false

	if config.AdminURL == "" {
		fmt.Println("Warning: --admin-url or ADMIN_URL must be set")
		missingParams = true
	}
	if config.AccessKey == "" {
		fmt.Println("Warning: --access-key or ACCESS_KEY must be set")
		missingParams = true
	}
	if config.SecretKey == "" {
		fmt.Println("Warning: --secret-key or SECRET_KEY must be set")
		missingParams = true
	}
	if config.Interval <= 0 {
		fmt.Println("Warning: --interval or INTERVAL must be set and greater than 0")
		missingParams = true
	}
	if config.QuotaUsagePercent < 0 || config.QuotaUsagePercent > 100 {
		fmt.Println("Warning: --quota-usage-percent or QUOTA_USAGE_PERCENT must be set between 0 and 100")
		missingParams = true
	}

	if missingParams {
		fmt.Println("One or more required parameters are missing. Please provide them through flags or environment variables.")
		os.Exit(1)
	}
}
