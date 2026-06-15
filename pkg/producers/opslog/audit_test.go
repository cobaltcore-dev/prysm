// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and prysm contributors
//
// SPDX-License-Identifier: Apache-2.0

package opslog

import (
	"testing"

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
