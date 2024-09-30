// Copyright 2024 Clyso GmbH
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package commands

import (
	"fmt"
	"os"

	"github.com/cobaltcore-dev/prysm/pkg/producers/bucketnotify"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var (
	bucketNotifyEndpointPort int
	bucketNotifyNatsURL      string
	bucketNotifySubject      string
)

var bucketNotifyCmd = &cobra.Command{
	Use:   "bucket-notify",
	Short: "Start the RGW bucket notifications handler",
	Long: `Start the RGW bucket notifications handler.

Note: Before using this command, ensure that RGW is configured to send notifications with the necessary details.
More info can be found here:
Ceph: https://docs.ceph.com/en/latest/radosgw/notifications/
Rook: https://rook.io/docs/rook/latest-release/Storage-Configuration/Object-Storage-RGW/ceph-object-bucket-notifications/
`,
	Run: func(cmd *cobra.Command, args []string) {
		config := bucketnotify.BucketNotifyConfig{
			EndpointPort: bucketNotifyEndpointPort,
			NatsURL:      bucketNotifyNatsURL,
			NatsSubject:  bucketNotifySubject,
		}

		config = mergeBucketNotifyConfigWithEnv(config)

		config.UseNats = config.NatsURL != ""

		event := log.Info()
		event.Bool("use_nats", config.UseNats)
		if config.UseNats {
			event.Str("nats_url", config.NatsURL)
			event.Str("nats_subject", config.NatsSubject)
		}
		// Finalize the log message with the main message
		event.Msg("configuration_loaded")

		validateBucketNotifyConfig(config)

		bucketnotify.StartBucketNotifyServer(config)
	},
}

func mergeBucketNotifyConfigWithEnv(cfg bucketnotify.BucketNotifyConfig) bucketnotify.BucketNotifyConfig {
	cfg.EndpointPort = getEnvInt("BUCKET_NOTIFY_ENDPOINT_PORT", cfg.EndpointPort)
	cfg.NatsURL = getEnv("NATS_URL", cfg.NatsURL)
	cfg.NatsSubject = getEnv("NATS_SUBJECT", cfg.NatsSubject)

	return cfg
}

func init() {
	bucketNotifyCmd.Flags().IntVar(&bucketNotifyEndpointPort, "port", 8080, "HTTP endpoint port to listen for bucket notifications")
	bucketNotifyCmd.Flags().StringVar(&bucketNotifyNatsURL, "nats-url", "", "NATS server URL")
	bucketNotifyCmd.Flags().StringVar(&bucketNotifySubject, "nats-subject", "rgw.buckets.notify", "NATS subject to publish results")
}

func validateBucketNotifyConfig(config bucketnotify.BucketNotifyConfig) {
	missingParams := false

	if config.EndpointPort == 0 {
		fmt.Println("Warning: --port must be set")
		missingParams = true
	}

	if missingParams {
		fmt.Println("One or more required parameters are missing. Please provide them through flags or environment variables.")
		os.Exit(1)
	}
}
