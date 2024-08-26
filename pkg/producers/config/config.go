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

package config

import (
	"fmt"

	"github.com/spf13/viper"
)

type GlobalConfig struct {
	NatsURL    string `mapstructure:"nats_url"`
	AdminURL   string `mapstructure:"admin_url"`
	AccessKey  string `mapstructure:"access_key"`
	SecretKey  string `mapstructure:"secret_key"`
	NodeName   string `mapstructure:"node_name"`
	InstanceID string `mapstructure:"instance_id"`
}

type ProducerConfig struct {
	Name     string                 `mapstructure:"name"`
	Type     string                 `mapstructure:"type"`
	Settings map[string]interface{} `mapstructure:"settings"`
}

type Config struct {
	Global    GlobalConfig     `mapstructure:"global"`
	Producers []ProducerConfig `mapstructure:"producers"`
}

func LoadConfig(path string) (*Config, error) {
	viper.SetConfigFile(path)

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("error reading config file: %w", err)
	}

	var config Config
	err := viper.Unmarshal(&config)
	if err != nil {
		return nil, fmt.Errorf("unable to decode into struct: %w", err)
	}

	return &config, nil
}

func GetStringSetting(settings map[string]interface{}, key, defaultValue string) string {
	if value, ok := settings[key].(string); ok {
		return value
	}
	return defaultValue
}

func GetIntSetting(settings map[string]interface{}, key string, defaultValue int) int {
	if value, ok := settings[key].(int); ok {
		return value
	}
	return defaultValue
}

func GetBoolSetting(settings map[string]interface{}, key string, defaultValue bool) bool {
	if value, ok := settings[key].(bool); ok {
		return value
	}
	return defaultValue
}

func GetFloat64Setting(settings map[string]interface{}, key string, defaultValue float64) float64 {
	if value, ok := settings[key].(float64); ok {
		return value
	}
	return defaultValue
}

func GetStringSliceSetting(settings map[string]interface{}, key string, defaultValue []string) []string {
	if value, ok := settings[key].([]interface{}); ok {
		var result []string
		for _, v := range value {
			if str, ok := v.(string); ok {
				result = append(result, str)
			}
		}
		return result
	}
	return defaultValue
}
