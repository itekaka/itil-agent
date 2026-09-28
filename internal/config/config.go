package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	ServerURL         string `yaml:"server_url"`
	AgentSecret       string `yaml:"agent_secret"`
	HeartbeatInterval int    `yaml:"heartbeat_interval"`
	InventoryInterval int    `yaml:"inventory_interval"`
	LogLevel          string `yaml:"log_level"`
	UUIDFile          string `yaml:"uuid_file"`
	TLSSkipVerify     bool   `yaml:"tls_skip_verify"`
}

func Load(path string) (*Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var c Config
	if err := yaml.Unmarshal(b, &c); err != nil {
		return nil, err
	}
	if c.HeartbeatInterval <= 0 {
		c.HeartbeatInterval = 300
	}
	if c.InventoryInterval <= 0 {
		c.InventoryInterval = 21600
	}
	if c.UUIDFile == "" {
		c.UUIDFile = "agent.uuid"
	}
	return &c, nil
}
