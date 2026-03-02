// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and prysm contributors
//
// SPDX-License-Identifier: Apache-2.0

package radosgwusage

import "testing"

func TestNormalizeUserTenant(t *testing.T) {
	tests := []struct {
		name       string
		user       string
		tenant     string
		wantUser   string
		wantTenant string
	}{
		{
			name:       "plain user no tenant",
			user:       "alice",
			tenant:     "",
			wantUser:   "alice",
			wantTenant: "",
		},
		{
			name:       "user embeds tenant",
			user:       "alice$t1",
			tenant:     "",
			wantUser:   "alice",
			wantTenant: "t1",
		},
		{
			name:       "explicit tenant wins",
			user:       "alice$t1",
			tenant:     "t2",
			wantUser:   "alice",
			wantTenant: "t2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotUser, gotTenant := NormalizeUserTenant(tt.user, tt.tenant)
			if gotUser != tt.wantUser || gotTenant != tt.wantTenant {
				t.Fatalf("NormalizeUserTenant(%q, %q) = (%q, %q), want (%q, %q)",
					tt.user, tt.tenant, gotUser, gotTenant, tt.wantUser, tt.wantTenant)
			}
		})
	}
}

func TestBucketAndUsageKeyParity(t *testing.T) {
	tests := []struct {
		name       string
		owner      string
		tenant     string
		usageUser  string
		bucketName string
	}{
		{
			name:       "owner has inline tenant",
			owner:      "alice$t1",
			tenant:     "",
			usageUser:  "alice$t1",
			bucketName: "b1",
		},
		{
			name:       "owner split from tenant field",
			owner:      "alice",
			tenant:     "t1",
			usageUser:  "alice$t1",
			bucketName: "b1",
		},
		{
			name:       "owner has inline tenant plus explicit tenant",
			owner:      "alice$t1",
			tenant:     "t1",
			usageUser:  "alice$t1",
			bucketName: "b2",
		},
		{
			name:       "no tenant",
			owner:      "alice",
			tenant:     "",
			usageUser:  "alice",
			bucketName: "b3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bucketUser, bucketTenant := NormalizeUserTenant(tt.owner, tt.tenant)
			bucketKey := BuildUserTenantBucketKey(bucketUser, bucketTenant, tt.bucketName)

			usageUser, usageTenant := NormalizeUserTenant(tt.usageUser, "")
			usageKey := BuildUserTenantBucketKey(usageUser, usageTenant, tt.bucketName)

			if bucketKey != usageKey {
				t.Fatalf("bucket/usage key mismatch: bucketKey=%q usageKey=%q", bucketKey, usageKey)
			}
		})
	}
}
