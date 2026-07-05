package config

import (
	"errors"
	"os"

	"github.com/goccy/go-yaml"
)

type Config struct {
	Auth        AuthConfig        `yaml:"auth"`
	Email       EmailConfig       `yaml:"-"`
	ServiceSlot ServiceSlotConfig `yaml:"service_slot"`
}

type EmailConfig struct {
	SendGridAPIKey string
	FromEmail      string
	FromName       string
}

type AuthConfig struct {
	MaxFailedLoginAttempts int     `yaml:"max_failed_login_attempts"`
	LockDurationMinutes    int     `yaml:"lock_duration_minutes"`
	JWTExpirationHours     float64 `yaml:"jwt_expiration_hours"`
	JWTSecret              string  `yaml:"-"`
}

type ServiceSlotConfig struct {
	RecurringHorizonWeeks int `yaml:"recurring_horizon_weeks"`
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
	cfg.Auth.JWTSecret = os.Getenv("JWT_SECRET")
	if cfg.Auth.JWTSecret == "" {
		return nil, errors.New("JWT_SECRET is required in environment variables")
	}
	cfg.Email.SendGridAPIKey = os.Getenv("SENDGRID_API_KEY")
	cfg.Email.FromEmail = os.Getenv("FROM_EMAIL")
	cfg.Email.FromName = os.Getenv("FROM_NAME")
	return cfg, nil
}
