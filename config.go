package traefik_token_injector

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// Config represents the plugin configuration from Traefik
type Config struct {
	ServiceId string `json:"serviceId" yaml:"serviceId"`
}

// CreateConfig creates the default plugin config
func CreateConfig() *Config {
	return &Config{}
}

// GlobalConfig represents the global configuration from config.yml
type GlobalConfig struct {
	GraphQLAPIURL      string `yaml:"graphql_api_url"`
	GraphQLAuthType    string `yaml:"graphql_auth_type"` // "none", "basic", "apitoken"
	GraphQLUsername    string `yaml:"graphql_username"`
	GraphQLPassword    string `yaml:"graphql_password"`
	GraphQLAPIToken    string `yaml:"graphql_api_token"`
	GraphQLTokenHeader string `yaml:"graphql_token_header"`
	Timeout            string `yaml:"timeout"`
	CacheEnabled       bool   `yaml:"cache_enabled"`
	TokenRefreshBuffer int    `yaml:"token_refresh_buffer"`
}

// LoadGlobalConfig loads the global configuration from instance/etc/config.yml
func LoadGlobalConfig() (*GlobalConfig, error) {
	// Get the current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get working directory: %w", err)
	}

	// Construct path to config file
	configPath := filepath.Join(cwd, "instance", "etc", "config.yml")

	// Read the config file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file at %s: %w", configPath, err)
	}

	// Parse YAML
	var config GlobalConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Set defaults
	if config.GraphQLAuthType == "" {
		config.GraphQLAuthType = "none"
	}
	if config.GraphQLTokenHeader == "" {
		config.GraphQLTokenHeader = "Authorization"
	}
	if config.Timeout == "" {
		config.Timeout = "10s"
	}
	if config.TokenRefreshBuffer == 0 {
		config.TokenRefreshBuffer = 10
	}

	return &config, nil
}

// GetTimeout parses the timeout string and returns a time.Duration
func (c *GlobalConfig) GetTimeout() (time.Duration, error) {
	return time.ParseDuration(c.Timeout)
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.ServiceId == "" {
		return fmt.Errorf("serviceId is required")
	}
	return nil
}

// Validate validates the global configuration
func (c *GlobalConfig) Validate() error {
	if c.GraphQLAPIURL == "" {
		return fmt.Errorf("graphql_api_url is required")
	}

	// Validate auth type
	switch c.GraphQLAuthType {
	case "none", "basic", "apitoken":
		// Valid
	default:
		return fmt.Errorf("invalid graphql_auth_type: %s (must be 'none', 'basic', or 'apitoken')", c.GraphQLAuthType)
	}

	// Validate basic auth
	if c.GraphQLAuthType == "basic" {
		if c.GraphQLUsername == "" || c.GraphQLPassword == "" {
			return fmt.Errorf("graphql_username and graphql_password are required for basic auth")
		}
	}

	// Validate API token auth
	if c.GraphQLAuthType == "apitoken" {
		if c.GraphQLAPIToken == "" {
			return fmt.Errorf("graphql_api_token is required for apitoken auth")
		}
	}

	return nil
}
