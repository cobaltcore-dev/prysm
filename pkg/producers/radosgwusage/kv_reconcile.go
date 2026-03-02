// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and prysm contributors
//
// SPDX-License-Identifier: Apache-2.0

package radosgwusage

import (
	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog/log"
)

func reconcileKVKeys(kv nats.KeyValue, keep map[string]struct{}, storeLabel string) {
	keys, err := kv.Keys()
	if err != nil {
		log.Warn().Err(err).Str("store", storeLabel).Msg("Failed to list keys for KV reconciliation")
		return
	}

	deleted := 0
	for _, key := range keys {
		if _, ok := keep[key]; ok {
			continue
		}
		if err := kv.Delete(key); err != nil {
			log.Warn().Err(err).Str("store", storeLabel).Str("key", key).Msg("Failed to delete stale KV key")
			continue
		}
		deleted++
	}

	log.Debug().
		Str("store", storeLabel).
		Int("keys_kept", len(keep)).
		Int("keys_deleted", deleted).
		Msg("Completed KV key reconciliation")
}
