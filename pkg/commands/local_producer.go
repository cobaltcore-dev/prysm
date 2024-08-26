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
	"log"
	"sync"

	"github.com/spf13/cobra"
	"gitlab.clyso.com/clyso/radosguard/pkg/producers/config"
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
