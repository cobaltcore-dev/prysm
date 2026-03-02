// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and prysm contributors
//
// SPDX-License-Identifier: Apache-2.0

package radosgwusage

import (
	"encoding/json"
	"testing"

	"github.com/cobaltcore-dev/prysm/pkg/producers/radosgwusage/rgwadmin"
)

func TestProcessUserMetrics_PreservesExpectedFields(t *testing.T) {
	userID := "alice$t1"
	tenant := ""
	normUser, normTenant := NormalizeUserTenant(userID, tenant)
	userKey := BuildUserTenantKey(normUser, normTenant)

	numObjects := uint64(123)
	sizeBytes := uint64(4567)
	quotaEnabled := true
	quotaMaxSize := int64(98765)
	quotaMaxObjects := int64(321)

	user := rgwadmin.KVUser{
		ID:                  userID,
		Tenant:              tenant,
		DisplayName:         "Alice",
		Email:               "alice@example.com",
		DefaultStorageClass: "STANDARD",
		Stats: rgwadmin.UserStat{
			NumObjects: &numObjects,
			Size:       &sizeBytes,
		},
		UserQuota: rgwadmin.QuotaSpec{
			Enabled:    &quotaEnabled,
			MaxSize:    &quotaMaxSize,
			MaxObjects: &quotaMaxObjects,
		},
	}

	userJSON, err := json.Marshal(user)
	if err != nil {
		t.Fatalf("marshal user: %v", err)
	}

	userData := newTestKV("user_data", map[string][]byte{
		userKey: userJSON,
	})
	userMetrics := newTestKV("user_metrics", nil)

	bucketKeyMap := map[string]uint64{
		userKey: 3,
	}

	processUserMetrics(userKey, userData, userMetrics, bucketKeyMap)

	entry, err := userMetrics.Get(userKey)
	if err != nil {
		t.Fatalf("expected user metric for key %q: %v", userKey, err)
	}

	var got UserLevelMetrics
	if err := json.Unmarshal(entry.Value(), &got); err != nil {
		t.Fatalf("unmarshal user metric: %v", err)
	}

	if got.User != normUser {
		t.Fatalf("unexpected user: got=%q want=%q", got.User, normUser)
	}
	if got.Tenant != normTenant {
		t.Fatalf("unexpected tenant: got=%q want=%q", got.Tenant, normTenant)
	}
	if got.DisplayName != user.DisplayName {
		t.Fatalf("unexpected display name: got=%q want=%q", got.DisplayName, user.DisplayName)
	}
	if got.Email != user.Email {
		t.Fatalf("unexpected email: got=%q want=%q", got.Email, user.Email)
	}
	if got.DefaultStorageClass != user.DefaultStorageClass {
		t.Fatalf("unexpected storage class: got=%q want=%q", got.DefaultStorageClass, user.DefaultStorageClass)
	}
	if got.BucketsTotal != 3 {
		t.Fatalf("unexpected buckets total: got=%d want=3", got.BucketsTotal)
	}
	if got.ObjectsTotal != numObjects {
		t.Fatalf("unexpected objects total: got=%d want=%d", got.ObjectsTotal, numObjects)
	}
	if got.DataSizeTotal != sizeBytes {
		t.Fatalf("unexpected data size total: got=%d want=%d", got.DataSizeTotal, sizeBytes)
	}
	if !got.UserQuotaEnabled {
		t.Fatalf("expected user quota enabled")
	}
	if got.UserQuotaMaxSize == nil || *got.UserQuotaMaxSize != quotaMaxSize {
		t.Fatalf("unexpected user quota max size: %+v", got.UserQuotaMaxSize)
	}
	if got.UserQuotaMaxObjects == nil || *got.UserQuotaMaxObjects != quotaMaxObjects {
		t.Fatalf("unexpected user quota max objects: %+v", got.UserQuotaMaxObjects)
	}
}
