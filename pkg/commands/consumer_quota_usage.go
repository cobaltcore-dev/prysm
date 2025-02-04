// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and prysm contributors
//
// SPDX-License-Identifier: Apache-2.0

package commands

import (
	"fmt"
	"os"

	"github.com/cobaltcore-dev/prysm/pkg/consumer/quotausageconsumer"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var (
	qucNatsURL           string
	qucNatsSubject       string
	qucPrometheus        bool
	qucPrometheusPort    int
	qucQuotaUsagePercent float64
	qucNodeName          string
	qucInstanceID        string
)

var quotaUsageConsumerCmd = &cobra.Command{
	Use:   "quota-usage-consumer",
	Short: "Consumer for monitoring quota usage",
	Run: func(cmd *cobra.Command, args []string) {
		config := quotausageconsumer.QuotaUsageConsumerConfig{
			NatsURL:           qucNatsURL,
			NatsSubject:       qucNatsSubject,
			Prometheus:        qucPrometheus,
			PrometheusPort:    qucPrometheusPort,
			QuotaUsagePercent: qucQuotaUsagePercent,
			NodeName:          qucNodeName,
			InstanceID:        qucInstanceID,
		}

		config = mergeQuotaUsageConsumerConfigWithEnv(config)

		event := log.Info()
		event.Str("nats_url", config.NatsURL)
		event.Str("nats_subject", config.NatsSubject)

		event.Bool("prometheus_enabled", config.Prometheus)
		if config.Prometheus {
			event.Int("prometheus_port", config.PrometheusPort)
		}

		event.Str("node_name", config.NodeName)
		event.Str("instance_id", config.InstanceID)
		event.Float64("quota_usage_percent", config.QuotaUsagePercent)

		// Finalize the log message with the main message
		event.Msg("configuration_loaded")

		validateQuotaUsageConsumerConfig(config)

		quotausageconsumer.StartQuotaUsageConsumer(config)
	},
}

func mergeQuotaUsageConsumerConfigWithEnv(cfg quotausageconsumer.QuotaUsageConsumerConfig) quotausageconsumer.QuotaUsageConsumerConfig {
	cfg.NatsURL = getEnv("NATS_URL", cfg.NatsURL)
	cfg.NatsSubject = getEnv("NATS_SUBJECT", cfg.NatsSubject)
	cfg.PrometheusPort = getEnvInt("PROMETHEUS_PORT", cfg.PrometheusPort)
	cfg.QuotaUsagePercent = getEnvFloat("QUOTA_USAGE_PERCENT", cfg.QuotaUsagePercent)
	cfg.NodeName = getEnv("NODE_NAME", cfg.NodeName)
	cfg.InstanceID = getEnv("INSTANCE_ID", cfg.InstanceID)

	return cfg
}

func init() {
	quotaUsageConsumerCmd.Flags().StringVar(&qucNatsURL, "nats-url", "", "NATS server URL")
	quotaUsageConsumerCmd.Flags().StringVar(&qucNatsSubject, "nats-subject", "user.quotas.usage", "NATS subject to subscribe to")
	quotaUsageConsumerCmd.Flags().BoolVar(&qucPrometheus, "prometheus", false, "Enable Prometheus metrics")
	quotaUsageConsumerCmd.Flags().IntVar(&qucPrometheusPort, "prometheus-port", 8080, "Prometheus metrics port")
	quotaUsageConsumerCmd.Flags().Float64Var(&qucQuotaUsagePercent, "quota-usage-percent", 80.0, "Percentage of quota usage to monitor")
	quotaUsageConsumerCmd.Flags().StringVar(&qucNodeName, "node-name", "", "Node name for identifying the source of the quotas")
	quotaUsageConsumerCmd.Flags().StringVar(&qucInstanceID, "instance-id", "", "Instance ID for identifying the source of the quotas")
}

func validateQuotaUsageConsumerConfig(config quotausageconsumer.QuotaUsageConsumerConfig) {
	missingParams := false

	if config.NatsURL == "" {
		fmt.Println("Warning: --nats-url or NATS_URL must be set")
		missingParams = true
	}
	if config.NatsSubject == "" {
		fmt.Println("Warning: --nats-subject or NATS_SUBJECT must be set")
		missingParams = true
	}
	if config.PrometheusPort <= 0 {
		fmt.Println("Warning: --prometheus-port or PROMETHEUS_PORT must be set and greater than 0")
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
