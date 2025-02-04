// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and prysm contributors
//
// SPDX-License-Identifier: Apache-2.0

package commands

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var (
	v            string
	runningInPod bool
	// responseBackToOperator bool
)

var rootCmd = &cobra.Command{
	Use:   "prysm",
	Short: "CLI for Ceph & RadosGW observability",
	Long:  "A CLI tool to manage Ceph & RadosGW observability, including logging and metrics collection.",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if err := setUpLogs(v); err != nil {
			return err
		}
		return nil
	},
}

func init() {
	runningInPod = checkIfRunningInPod()

	rootCmd.PersistentFlags().StringVarP(&v, "verbosity", "v", zerolog.WarnLevel.String(), "Log level (debug, info, warn, error, fatal, panic")

	if runningInPod {
		log.Info().Msg("running in pod")
		// rootCmd.PersistentFlags().BoolVar(&responseBackToOperator, "response-back-to-operator", false, "Send response back to operator (k8s only)")
	}

	// Add subcommands
	rootCmd.AddCommand(consumerCmd)
	rootCmd.AddCommand(localProducerCmd)
	rootCmd.AddCommand(remoteProducerCmd)
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Whoops. There was an error while executing your CLI '%s'\n", err)
		os.Exit(1)
	}
}

// setUpLogs sets the log output and the log level
func setUpLogs(level string) error {
	zerolog.SetGlobalLevel(zerolog.WarnLevel) // Default level
	lvl, err := zerolog.ParseLevel(level)
	if err != nil {
		return err
	}
	zerolog.SetGlobalLevel(lvl)
	log.Logger = zerolog.New(os.Stdout).With().Timestamp().Logger() // Default to JSON output
	// log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout})
	return nil
}

// checkIfRunningInPod checks if the application is running in a Kubernetes pod
func checkIfRunningInPod() bool {
	if _, err := os.Stat("/run/secrets/kubernetes.io/serviceaccount/ca.crt"); err == nil {
		if _, err := os.Stat("/run/secrets/kubernetes.io/serviceaccount/token"); err == nil {
			if _, ok := os.LookupEnv("KUBERNETES_SERVICE_HOST"); ok {
				if _, ok := os.LookupEnv("KUBERNETES_SERVICE_PORT"); ok {
					return true
				}
			}
		}
	}
	return false
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}

func getEnvInt(key string, defaultValue int) int {
	valueStr := os.Getenv(key)
	if value, err := strconv.Atoi(valueStr); err == nil {
		return value
	}
	return defaultValue
}

func getEnvInt64(key string, defaultValue int64) int64 {
	valueStr := os.Getenv(key)
	if value, err := strconv.ParseInt(valueStr, 10, 64); err == nil {
		return value
	}
	return defaultValue
}

func getEnvFloat(key string, defaultValue float64) float64 {
	if value, exists := os.LookupEnv(key); exists {
		if floatValue, err := strconv.ParseFloat(value, 64); err == nil {
			return floatValue
		}
	}
	return defaultValue
}

func getEnvInt64Slice(key string, defaultValue []int64) []int64 {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}
	values := strings.Split(valueStr, ",")
	result := make([]int64, len(values))
	for i, v := range values {
		value, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return defaultValue
		}
		result[i] = value
	}
	return result
}

func getEnvBool(key string, defaultValue bool) bool {
	valueStr := os.Getenv(key)
	if value, err := strconv.ParseBool(valueStr); err == nil {
		return value
	}
	return defaultValue
}
