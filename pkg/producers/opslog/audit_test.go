// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and prysm contributors
//
// SPDX-License-Identifier: Apache-2.0

package opslog

import (
	"testing"

	"github.com/sapcc/go-api-declarations/cadf"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBuildRabbitMQConnectionURL verifies that explicitly supplied username and
// password (e.g. sourced from two separate Vault entries synced into a Secret)
// are composed into the AMQP connection URL's userinfo.
func TestBuildRabbitMQConnectionURL(t *testing.T) {
	tests := []struct {
		name     string
		rawURL   string
		username string
		password string
		want     string
	}{
		{
			name:   "no credentials returns url unchanged",
			rawURL: "amqp://rabbitmq.example:5672/",
			want:   "amqp://rabbitmq.example:5672/",
		},
		{
			name:     "username and password injected into bare host url",
			rawURL:   "amqp://rabbitmq.example:5672/",
			username: "audit",
			password: "s3cr3t",
			want:     "amqp://audit:s3cr3t@rabbitmq.example:5672/",
		},
		{
			name:     "username only",
			rawURL:   "amqp://rabbitmq.example:5672/",
			username: "audit",
			want:     "amqp://audit@rabbitmq.example:5672/",
		},
		{
			name:     "explicit credentials override userinfo already in url",
			rawURL:   "amqp://old:oldpass@rabbitmq.example:5672/",
			username: "audit",
			password: "s3cr3t",
			want:     "amqp://audit:s3cr3t@rabbitmq.example:5672/",
		},
		{
			name:     "special characters in password are percent-encoded",
			rawURL:   "amqp://rabbitmq.example:5672/vhost",
			username: "audit",
			password: "p@ss/w:rd",
			want:     "amqp://audit:p%40ss%2Fw%3Ard@rabbitmq.example:5672/vhost",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := buildRabbitMQConnectionURL(tt.rawURL, tt.username, tt.password)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}

	t.Run("invalid url with credentials returns error", func(t *testing.T) {
		_, err := buildRabbitMQConnectionURL("://not a url", "u", "p")
		require.Error(t, err)
	})

	t.Run("password without username is rejected", func(t *testing.T) {
		_, err := buildRabbitMQConnectionURL("amqp://rabbitmq.example:5672/", "", "s3cr3t")
		require.Error(t, err)
	})
}

// TestHasUsableTenant verifies the tenant gate used by AUDIT_REQUIRE_TENANT:
// an event is publishable only if it yields a project_id or domain_id.
func TestHasUsableTenant(t *testing.T) {
	scope := func(projectID, domainID string) *KeystoneScope {
		return &KeystoneScope{
			Project: KeystoneProject{
				ID:     projectID,
				Domain: KeystoneDomain{ID: domainID},
			},
		}
	}

	tests := []struct {
		name  string
		opLog *S3OperationLog
		want  bool
	}{
		{"keystone scope with project id", &S3OperationLog{KeystoneScope: scope("proj-1", "dom-1")}, true},
		{"keystone scope with only domain id", &S3OperationLog{KeystoneScope: scope("", "dom-1")}, true},
		{"keystone scope with neither", &S3OperationLog{KeystoneScope: scope("", "")}, false},
		{"no scope, tenant-encoded user", &S3OperationLog{User: "tenant1$user1"}, true},
		{"no scope, bare user", &S3OperationLog{User: "user1"}, true},
		{"no scope, anonymous user", &S3OperationLog{User: "anonymous"}, false},
		{"no scope, empty user", &S3OperationLog{User: ""}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, hasUsableTenant(tt.opLog))
		})
	}
}

// TestBuildTargetAccountProjectID verifies that the account-level target uses
// the Keystone project ID when present (consistent with the initiator),
// falling back to the parsed user only when no Keystone scope is available.
func TestBuildTargetAccountProjectID(t *testing.T) {
	t.Run("prefers keystone project id over parsed user", func(t *testing.T) {
		opLog := &S3OperationLog{
			User: "rgwuser", // would parse to "rgwuser"
			KeystoneScope: &KeystoneScope{
				Project: KeystoneProject{ID: "proj-hash-123"},
			},
		}
		acct, ok := buildTarget(opLog).(*AccountTarget)
		require.True(t, ok)
		assert.Equal(t, "proj-hash-123", acct.ProjectID)
	})

	t.Run("falls back to parsed user without keystone scope", func(t *testing.T) {
		opLog := &S3OperationLog{User: "tenant1$user1"}
		acct, ok := buildTarget(opLog).(*AccountTarget)
		require.True(t, ok)
		assert.Equal(t, "tenant1", acct.ProjectID)
	})
}

// TestWithRegion verifies that a configured region is stamped onto the target
// as an attachment, and that an empty region leaves the target untouched.
func TestWithRegion(t *testing.T) {
	regionOf := func(r cadf.Resource) string {
		for _, a := range r.Attachments {
			if a.Name == "region" {
				if s, ok := a.Content.(string); ok {
					return s
				}
			}
		}
		return ""
	}

	t.Run("adds region attachment", func(t *testing.T) {
		target := withRegion(&BucketTarget{Bucket: "b1"}, "qa-de-1")
		assert.Equal(t, "qa-de-1", regionOf(target.Render()))
	})

	t.Run("empty region leaves target unchanged", func(t *testing.T) {
		base := &BucketTarget{Bucket: "b1"}
		target := withRegion(base, "")
		assert.Same(t, base, target)
		assert.Equal(t, "", regionOf(target.Render()))
	})
}

// TestBuildObserver verifies the audit observer identifies the storage service
// (not the resource/tool), with a configurable name defaulting to radosgw.
func TestBuildObserver(t *testing.T) {
	t.Run("defaults to radosgw service when name unset", func(t *testing.T) {
		obs := buildObserver(AuditSinkConfig{})
		assert.Equal(t, "service/storage", obs.TypeURI)
		assert.Equal(t, "radosgw", obs.Name)
		assert.NotEmpty(t, obs.ID)
	})

	t.Run("uses configured observer name", func(t *testing.T) {
		obs := buildObserver(AuditSinkConfig{ObserverName: "ceph"})
		assert.Equal(t, "service/storage", obs.TypeURI)
		assert.Equal(t, "ceph", obs.Name)
	})
}

// TestIsReadOperation verifies read classification (get/head/list) used by the
// optional read filter. Reads are audited by default for object storage, but
// can be excluded (mutations-only) via AUDIT_INCLUDE_READS=false.
func TestIsReadOperation(t *testing.T) {
	reads := []string{
		"get_obj", "head_obj", "get_bucket_info", "head_bucket",
		"list_buckets", "list_bucket", "get_acls", "get_bucket_policy",
		"get_lifecycle", "get_obj_tags",
	}
	mutations := []string{
		"put_obj", "create_bucket", "delete_obj", "delete_bucket", "copy_obj",
		"post_obj", "put_acls", "init_multipart", "complete_multipart",
		"abort_multipart",
	}

	for _, op := range reads {
		assert.True(t, isReadOperation(op), "expected %q to be a read", op)
	}
	for _, op := range mutations {
		assert.False(t, isReadOperation(op), "expected %q to be a mutation", op)
	}
}

// TestIsSkippedBucket verifies the loop-prevention filter: operations on the
// Hermes audit bucket are skipped so Hermes' own writes don't re-trigger audit.
// Matching is case-insensitive and supports a comma-separated list.
func TestIsSkippedBucket(t *testing.T) {
	tests := []struct {
		name        string
		bucket      string
		skipBuckets string
		want        bool
	}{
		{"exact match", "hermes", "hermes", true},
		{"case-insensitive bucket", "Hermes", "hermes", true},
		{"case-insensitive config", "hermes", "Hermes", true},
		{"all caps", "HERMES", "hermes", true},
		{"no match", "my-bucket", "hermes", false},
		{"empty bucket", "", "hermes", false},
		{"disabled (empty config)", "hermes", "", false},
		{"list member with spaces", "audit", "hermes, audit , logs", true},
		{"list non-member", "data", "hermes,audit,logs", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, isSkippedBucket(tt.bucket, tt.skipBuckets))
		})
	}
}
