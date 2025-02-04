// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and prysm contributors
//
// SPDX-License-Identifier: Apache-2.0

package commands

import (
	"log"
	"sync"

	"github.com/cobaltcore-dev/prysm/pkg/producers/config"
	"github.com/spf13/cobra"
)

var configFilePath string

var localProducerCmd = &cobra.Command{
	Use:   "local-producer",
	Short: "Local producer commands",
}

var useConfigCmd = &cobra.Command{
	Use:   "use-config",
	Short: "Start local producers using configuration file",
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := config.LoadConfig(configFilePath)
		if err != nil {
			log.Fatalf("Failed to load config: %v", err)
		}

		var wg sync.WaitGroup

		for _, producer := range cfg.Producers {
			wg.Add(1)
			go config.StartProducers(producer, cfg.Global, &wg)
		}

		wg.Wait()
	},
}

func init() {
	useConfigCmd.Flags().StringVar(&configFilePath, "config", "", "Path to configuration file")
	useConfigCmd.MarkFlagRequired("config")
	localProducerCmd.AddCommand(useConfigCmd)

	localProducerCmd.AddCommand(opsLogCmd)
	localProducerCmd.AddCommand(bucketNotifyCmd)
	localProducerCmd.AddCommand(diskHealthMetricsCmd)
	localProducerCmd.AddCommand(kernelMetricsCmd)
	localProducerCmd.AddCommand(resourceUsageCmd)
}
