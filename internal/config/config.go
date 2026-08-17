package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	DatabasePath string
	Address      string
	QueueSize    int
	AdminToken   string
}

func Default() Config {
	return Config{DatabasePath: "service-requests.db", Address: ":8080", QueueSize: 32, AdminToken: "local-admin"}
}

func FromEnv(base Config) Config {
	if value := os.Getenv("SRD_DATABASE"); value != "" {
		base.DatabasePath = value
	}
	if value := os.Getenv("SRD_ADDRESS"); value != "" {
		base.Address = value
	}
	if value := os.Getenv("SRD_ADMIN_TOKEN"); value != "" {
		base.AdminToken = value
	}
	if value := os.Getenv("SRD_QUEUE_SIZE"); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil && parsed > 0 {
			base.QueueSize = parsed
		}
	}
	return base
}

func (c Config) Validate() error {
	if c.DatabasePath == "" {
		return fmt.Errorf("database path is required")
	}
	if c.Address == "" {
		return fmt.Errorf("address is required")
	}
	if c.QueueSize < 1 {
		return fmt.Errorf("queue size must be positive")
	}
	if c.AdminToken == "" {
		return fmt.Errorf("admin token is required")
	}
	return nil
}
