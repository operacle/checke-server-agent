package config

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	// Server configuration
	ServerURL    string
	APIKey       string
	
	// PocketBase configuration
	PocketBaseEnabled bool
	PocketBaseURL     string
	
	// Monitoring intervals
	CheckInterval      time.Duration
	ReportInterval     time.Duration
	CommandCheckInterval time.Duration
	
	// Agent configuration
	AgentID          string
	MaxRetries       int
	RequestTimeout   time.Duration
	
	// Health check configuration
	HealthCheckPort  int
	
	// Remote control
	RemoteControlEnabled bool
	
	// Server identification - for server registration
	ServerName   string
	Hostname     string
	IPAddress    string
	OSType       string
	ServerToken  string
}

func Load() (*Config, error) {
	// Load .env file if it exists
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: .env file not found or could not be loaded: %v", err)
		log.Printf("Using system environment variables instead")
	} else {
		log.Printf("Successfully loaded .env file")
	}

	cfg := &Config{
		// Basic configuration with minimal defaults
		ServerURL:            getEnv("SERVER_URL", ""),
		APIKey:               getEnv("API_KEY", ""),
		PocketBaseEnabled:    getBoolEnv("POCKETBASE_ENABLED", false),
		PocketBaseURL:        getEnv("POCKETBASE_URL", ""),
		CheckInterval:        getDurationEnv("CHECK_INTERVAL", 30*time.Second),
		ReportInterval:       getDurationEnv("REPORT_INTERVAL", 5*time.Minute),
		CommandCheckInterval: getDurationEnv("COMMAND_CHECK_INTERVAL", 10*time.Second),
		AgentID:              getEnv("AGENT_ID", ""),
		MaxRetries:           getIntEnv("MAX_RETRIES", 3),
		RequestTimeout:       getDurationEnv("REQUEST_TIMEOUT", 10*time.Second),
		HealthCheckPort:      getIntEnv("HEALTH_CHECK_PORT", 9091),
		RemoteControlEnabled: getBoolEnv("REMOTE_CONTROL_ENABLED", false),
		
		// Server identification - all required from env
		ServerName:   getEnv("SERVER_NAME", ""),
		Hostname:     getEnv("HOSTNAME", ""),
		IPAddress:    getEnv("IP_ADDRESS", ""),
		OSType:       getEnv("OS_TYPE", ""),
		ServerToken:  getEnv("SERVER_TOKEN", ""),
	}

	// Validate required configuration
	if err := validateConfig(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

func validateConfig(cfg *Config) error {
	// At least one communication method must be enabled
	if !cfg.PocketBaseEnabled && cfg.APIKey == "" {
		return fmt.Errorf("at least one communication method must be enabled (POCKETBASE_ENABLED=true or API_KEY set)")
	}

	// Validate PocketBase configuration
	if cfg.PocketBaseEnabled {
		if cfg.PocketBaseURL == "" {
			return fmt.Errorf("POCKETBASE_URL is required when POCKETBASE_ENABLED=true")
		}
	}

	// Validate HTTP REST API configuration
	if !cfg.PocketBaseEnabled && cfg.ServerURL == "" {
		return fmt.Errorf("SERVER_URL is required when using HTTP REST API as fallback")
	}

	// Validate server identification
	if cfg.AgentID == "" {
		return fmt.Errorf("AGENT_ID is required")
	}

	// Validate server registration fields if PocketBase is enabled
	if cfg.PocketBaseEnabled {
		if cfg.ServerName == "" {
			return fmt.Errorf("SERVER_NAME is required when POCKETBASE_ENABLED=true")
		}
		if cfg.IPAddress == "" {
			return fmt.Errorf("IP_ADDRESS is required when POCKETBASE_ENABLED=true")
		}
		if cfg.ServerToken == "" {
			return fmt.Errorf("SERVER_TOKEN is required when POCKETBASE_ENABLED=true")
		}
	}

	return nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getIntEnv(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getBoolEnv(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

func getDurationEnv(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}