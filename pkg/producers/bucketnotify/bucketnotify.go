// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and prysm contributors
//
// SPDX-License-Identifier: Apache-2.0

package bucketnotify

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/rs/zerolog/log"

	"github.com/nats-io/nats.go"
)

type RGWNotification struct {
	Records []struct {
		EventVersion string `json:"eventVersion"`
		EventSource  string `json:"eventSource"`
		AwsRegion    string `json:"awsRegion"`
		EventTime    string `json:"eventTime"`
		EventName    string `json:"eventName"`
		UserIdentity struct {
			PrincipalID string `json:"principalId"`
		} `json:"userIdentity"`
		RequestParameters struct {
			SourceIPAddress string `json:"sourceIPAddress"`
		} `json:"requestParameters"`
		ResponseElements struct {
			XAmzRequestID string `json:"x-amz-request-id"`
			XAmzID2       string `json:"x-amz-id-2"`
		} `json:"responseElements"`
		S3 struct {
			S3SchemaVersion string `json:"s3SchemaVersion"`
			ConfigurationID string `json:"configurationId"`
			Bucket          struct {
				Name          string `json:"name"`
				OwnerIdentity struct {
					PrincipalID string `json:"principalId"`
				} `json:"ownerIdentity"`
				Arn string `json:"arn"`
			} `json:"bucket"`
			Object struct {
				Key       string `json:"key"`
				Size      int64  `json:"size"`
				ETag      string `json:"eTag"`
				VersionID string `json:"versionId"`
				Sequencer string `json:"sequencer"`
			} `json:"object"`
		} `json:"s3"`
	} `json:"Records"`
}

func StartBucketNotifyServer(cfg BucketNotifyConfig) {
	var nc *nats.Conn
	var err error
	if cfg.UseNats {
		nc, err = nats.Connect(cfg.NatsURL)
		if err != nil {
			log.Error().Err(err).Msg("error connecting to nats server")
			return
		}
		defer nc.Close()
	}

	http.HandleFunc("/notifications", func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "unable to read request body", http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		var notification RGWNotification
		if err := json.Unmarshal(body, &notification); err != nil {
			http.Error(w, "invalid json format", http.StatusBadRequest)
			return
		}

		if cfg.UseNats {
			if err := nc.Publish(cfg.NatsSubject, body); err != nil {
				http.Error(w, "error publishing to nats", http.StatusInternalServerError)
				return
			}
		} else {
			notificationBytes, err := json.MarshalIndent(notification, "", "  ")
			if err != nil {
				log.Error().Err(err).Msg("error marshalling log entry")
			}
			fmt.Println(string(notificationBytes))
		}

		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "notification received and forwarded to nats")
	})

	addr := fmt.Sprintf(":%d", cfg.EndpointPort)
	log.Info().Msgf("starting server on %s", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatal().Err(err).Msg("error starting http server")
	}
}
