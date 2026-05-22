package config

import (
	"os"

	"github.com/goccy/go-yaml"
)

type Config struct {
	Auth AuthConfig `yaml:"auth"`
}

type AuthConfig struct {
	MaxFailedLoginAttempts int `yaml:"max_failed_login_attempts"`
	LockDurationMinutes    int `yaml:"lock_duration_minutes"`
	JWTExpirationHours     int `yaml:"jwt_expiration_hours"`
}

func Load(path string) (*Config, error) {
	file, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	cfg := &Config{}
	if err := yaml.Unmarshal(file, cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}
