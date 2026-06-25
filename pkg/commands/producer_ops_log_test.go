// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and prysm contributors
//
// SPDX-License-Identifier: Apache-2.0

package commands

import (
	"testing"

	"github.com/cobaltcore-dev/prysm/pkg/producers/opslog"
	"github.com/stretchr/testify/assert"
)

// TestMergeOpsLogConfigWithEnv_AuditSink verifies that the audit/RabbitMQ sink
// can be configured via environment variables (e.g. injected by the mutating
// webhook through a Secret/ConfigMap), not just via command-line flags.
func TestMergeOpsLogConfigWithEnv_AuditSink(t *testing.T) {
	// Base config simulates the flag-derived defaults.
	base := opslog.OpsLogConfig{
		AuditSink: opslog.AuditSinkConfig{
			Enabled:           false,
			RabbitMQURL:       "",
			QueueName:         "keystone.notifications.info",
			InternalQueueSize: 20,
			Debug:             false,
			RequireTenant:     true,
		},
	}

	t.Run("env vars override flag defaults", func(t *testing.T) {
		t.Setenv("AUDIT_ENABLED", "true")
		t.Setenv("AUDIT_RABBITMQ_URL", "amqp://rabbit:5672/")
		t.Setenv("AUDIT_RABBITMQ_USERNAME", "audit")
		t.Setenv("AUDIT_RABBITMQ_PASSWORD", "s3cr3t")
		t.Setenv("AUDIT_QUEUE_NAME", "custom.audit.queue")
		t.Setenv("AUDIT_QUEUE_SIZE", "100")
		t.Setenv("AUDIT_DEBUG", "true")
		t.Setenv("AUDIT_REQUIRE_TENANT", "false")
		t.Setenv("AUDIT_REGION", "qa-de-1")
		t.Setenv("AUDIT_OBSERVER_NAME", "ceph")
		t.Setenv("AUDIT_INCLUDE_READS", "true")
		t.Setenv("AUDIT_SKIP_BUCKETS", "hermes,_default")

		cfg := mergeOpsLogConfigWithEnv(base)

		assert.True(t, cfg.AuditSink.Enabled)
		assert.Equal(t, "amqp://rabbit:5672/", cfg.AuditSink.RabbitMQURL)
		assert.Equal(t, "audit", cfg.AuditSink.RabbitMQUsername)
		assert.Equal(t, "s3cr3t", cfg.AuditSink.RabbitMQPassword)
		assert.Equal(t, "custom.audit.queue", cfg.AuditSink.QueueName)
		assert.Equal(t, 100, cfg.AuditSink.InternalQueueSize)
		assert.True(t, cfg.AuditSink.Debug)
		assert.False(t, cfg.AuditSink.RequireTenant)
		assert.Equal(t, "qa-de-1", cfg.AuditSink.Region)
		assert.Equal(t, "ceph", cfg.AuditSink.ObserverName)
		assert.True(t, cfg.AuditSink.IncludeReads)
		assert.Equal(t, "hermes,_default", cfg.AuditSink.SkipBuckets)
	})

	t.Run("unset env vars preserve flag defaults", func(t *testing.T) {
		cfg := mergeOpsLogConfigWithEnv(base)

		assert.False(t, cfg.AuditSink.Enabled)
		assert.Equal(t, "", cfg.AuditSink.RabbitMQURL)
		assert.Equal(t, "", cfg.AuditSink.RabbitMQUsername)
		assert.Equal(t, "", cfg.AuditSink.RabbitMQPassword)
		assert.Equal(t, "keystone.notifications.info", cfg.AuditSink.QueueName)
		assert.Equal(t, 20, cfg.AuditSink.InternalQueueSize)
		assert.False(t, cfg.AuditSink.Debug)
		assert.True(t, cfg.AuditSink.RequireTenant)
		assert.Equal(t, "", cfg.AuditSink.Region)
		assert.Equal(t, "", cfg.AuditSink.ObserverName)
		assert.False(t, cfg.AuditSink.IncludeReads)
		assert.Equal(t, "", cfg.AuditSink.SkipBuckets)
	})
}
