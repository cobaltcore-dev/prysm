// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and prysm contributors
//
// SPDX-License-Identifier: Apache-2.0

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
