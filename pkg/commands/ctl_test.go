// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and prysm contributors
//
// SPDX-License-Identifier: Apache-2.0

package commands

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetEnv(t *testing.T) {
	key := "TEST_KEY"
	fallback := "default_value"

	// Test when the environment variable is not set
	value := getEnv(key, fallback)
	assert.Equal(t, fallback, value)

	// Test when the environment variable is set
	expectedValue := "expected_value"
	os.Setenv(key, expectedValue)
	value = getEnv(key, fallback)
	assert.Equal(t, expectedValue, value)

	// Clean up
	os.Unsetenv(key)
}
