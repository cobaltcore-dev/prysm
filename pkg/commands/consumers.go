// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and prysm contributors
//
// SPDX-License-Identifier: Apache-2.0

package commands

import (
	"github.com/spf13/cobra"
)

var consumerCmd = &cobra.Command{
	Use:   "consumer",
	Short: "Consumer commands",
}

func init() {
	consumerCmd.AddCommand(quotaUsageConsumerCmd)
}
