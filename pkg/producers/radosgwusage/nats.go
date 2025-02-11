// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and prysm contributors
//
// SPDX-License-Identifier: Apache-2.0

package radosgwusage

import (
	"encoding/json"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog/log"
)

func publishToNATS(nc *nats.Conn, subject string, usages []UsageEntry) {
	usageJSON, err := json.Marshal(usages)
	if err != nil {
		log.Error().
			Err(err).
			Msg("error marshalling usage to JSON")
		return
	}

	err = nc.Publish(subject, usageJSON)
	if err != nil {
		log.Error().
			Err(err).
			Str("subject", subject).
			Msg("error publishing usage to NATS")
	}
}

// ensureStream ensures that the JetStream stream exists with proper configuration
func ensureStream(js nats.JetStreamContext, streamName string) error {
	stream, err := js.StreamInfo(streamName)
	if err == nil && stream != nil {
		log.Info().Str("name", streamName).Msg("Stream already exists")
		return nil
	}

	log.Info().Str("name", streamName).Msg("Creating stream")

	_, err = js.AddStream(&nats.StreamConfig{
		Name:      streamName,                // Stream Name
		Subjects:  []string{"notifications"}, // Subscribe to "notifications" topic
		Storage:   nats.FileStorage,          // File-based storage (to avoid memory pressure)
		Retention: nats.LimitsPolicy,         // Auto-delete old messages based on limits
		MaxAge:    10 * time.Minute,          // Retain messages for 10 minutes
		MaxBytes:  100 * 1024 * 1024,         // Retain up to 100MB of messages
		Discard:   nats.DiscardOld,           // Discard old messages when full
	})

	if err != nil {
		log.Err(err).Msg("Failed to create stream")
		return err
	}

	log.Info().Str("name", streamName).Msg("Stream successfully created")
	return nil
}

// publishEvent(nc, "sync_users", "in_progress", nil, map[string]string{"sync_mode": "full"})
func publishEvent(nc *nats.Conn, eventType string, status string, ids []string, metadata map[string]string) error {
	eventData := map[string]interface{}{
		"event":    eventType,
		"status":   status,
		"ids":      ids,
		"metadata": metadata,
	}
	data, err := json.Marshal(eventData)
	if err != nil {
		return err
	}
	return nc.Publish("notifications", data)
}

func listenForEvents(nc *nats.Conn) {
	sub, err := nc.Subscribe("notifications", func(msg *nats.Msg) {
		var event map[string]interface{}
		if err := json.Unmarshal(msg.Data, &event); err != nil {
			log.Error().Err(err).Msg("Failed to parse event")
			return
		}

		eventType := event["event"].(string)
		status := event["status"].(string)

		switch eventType {
		case "sync_users":
			if status == "in_progress" {
				// syncUsers()
				publishEvent(nc, "sync_users", "completed", nil, nil)
			}
		case "sync_buckets":
			if status == "in_progress" {
				// syncBuckets()
				publishEvent(nc, "sync_buckets", "completed", nil, nil)
			}
		case "sync_usage":
			if status == "in_progress" {
				// syncUsage()
				publishEvent(nc, "sync_usage", "completed", nil, nil)
			}
		case "generate_metrics":
			if status == "in_progress" {
				// generateMetrics()
				publishEvent(nc, "generate_metrics", "completed", nil, nil)
			}
		default:
			log.Warn().Str("event", eventType).Msg("Unknown event received")
		}
	})
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to subscribe to notifications")
	}
	defer sub.Unsubscribe()
	select {}
}

func retryFailedEvents(nc *nats.Conn) {
	sub, err := nc.Subscribe("notifications", func(msg *nats.Msg) {
		var event map[string]interface{}
		if err := json.Unmarshal(msg.Data, &event); err != nil {
			log.Error().Err(err).Msg("Failed to parse event")
			return
		}

		if event["status"].(string) == "failed" {
			log.Warn().Str("event", event["event"].(string)).Msg("Retrying failed event")
			publishEvent(nc, event["event"].(string), "in_progress", nil, nil)
		}
	})
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to subscribe to failed events")
	}
	defer sub.Unsubscribe()
	select {}
}
