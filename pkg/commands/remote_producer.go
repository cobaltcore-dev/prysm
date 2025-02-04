// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and prysm contributors
//
// SPDX-License-Identifier: Apache-2.0

package commands

import (
	"github.com/spf13/cobra"
)

var remoteProducerCmd = &cobra.Command{
	Use:   "remote-producer",
	Short: "Remote producer commands",
}

func init() {
	remoteProducerCmd.AddCommand(bucketNotifyCmd)
	// remoteProducerCmd.AddCommand(metricsCmd)
	remoteProducerCmd.AddCommand(quotaUsageMonitorCmd)
	remoteProducerCmd.AddCommand(radosGWUsageCmd)
}
