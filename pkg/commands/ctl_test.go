// Copyright (C) 2024 Clyso GmbH
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

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
