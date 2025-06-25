package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// DatabaseInstance represents a database instance configuration
type DatabaseInstance struct {
	Type            string            `yaml:"type"`
	Host            string            `yaml:"host"`
	Port            int               `yaml:"port"`
	Username        string            `yaml:"username"`
	Password        string            `yaml:"password"`
	Database        string            `yaml:"database,omitempty"` // Optional - if not set, monitor all databases
	SSLMode         string            `yaml:"ssl_mode,omitempty"`
	RefreshInterval time.Duration     `yaml:"refresh_interval,omitempty"` // Default 2s if not set
	Options         map[string]string `yaml:"options,omitempty"`
}

// Config represents the overall configuration structure
type Config struct {
	Instances map[string]DatabaseInstance `yaml:"instances"`
}

// Load reads and parses the configuration file
func Load(configPath string) (*Config, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	if config.Instances == nil {
		config.Instances = make(map[string]DatabaseInstance)
	}

	// Set default refresh interval for instances that don't have it
	for name, instance := range config.Instances {
		if instance.RefreshInterval == 0 {
			instance.RefreshInterval = 2 * time.Second
		}
		config.Instances[name] = instance
	}

	return &config, nil
}

// GetDSN returns the database connection string for the given instance
func (di *DatabaseInstance) GetDSN() string {
	switch di.Type {
	case "postgres", "postgresql":
		dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s sslmode=%s",
			di.Host, di.Port, di.Username, di.Password, di.SSLMode)
		if di.Database != "" {
			dsn += fmt.Sprintf(" dbname=%s", di.Database)
		}
		return dsn
	case "mysql":
		dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/",
			di.Username, di.Password, di.Host, di.Port)
		if di.Database != "" {
			dsn += di.Database
		}
		return dsn
	case "mariadb":
		dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/",
			di.Username, di.Password, di.Host, di.Port)
		if di.Database != "" {
			dsn += di.Database
		}
		return dsn
	case "oracle":
		dsn := fmt.Sprintf("%s/%s@%s:%d/",
			di.Username, di.Password, di.Host, di.Port)
		if di.Database != "" {
			dsn += di.Database
		}
		return dsn
	default:
		return ""
	}
}
