// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and prysm contributors
//
// SPDX-License-Identifier: Apache-2.0

package radosgwusage

import "testing"

func TestReconcileKVKeys_DeletesStaleKeys(t *testing.T) {
	kv := newTestKV("test", map[string][]byte{
		"keep":  []byte("1"),
		"stale": []byte("2"),
	})

	keep := map[string]struct{}{
		"keep": {},
	}

	reconcileKVKeys(kv, keep, "test")

	if _, err := kv.Get("keep"); err != nil {
		t.Fatalf("expected keep key to remain: %v", err)
	}
	if _, err := kv.Get("stale"); err == nil {
		t.Fatalf("expected stale key to be deleted")
	}
}
