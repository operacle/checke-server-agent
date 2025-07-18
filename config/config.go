package config

import (
	"fmt"
	"log"
	"net"
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
	// Try to load environment file from multiple locations
	envFiles := []string{
		"/etc/monitoring-agent/monitoring-agent.env",
		"/etc/monitoring-agent/.env",
		".env",
		"monitoring-agent.env",
	}

	envLoaded := false
	for _, envFile := range envFiles {
		if err := godotenv.Load(envFile); err == nil {
			log.Printf("Successfully loaded environment file: %s", envFile)
			envLoaded = true
			break
		}
	}

	if !envLoaded {
		log.Printf("Warning: No environment file found in any of these locations: %v", envFiles)
		log.Printf("Using system environment variables only")
	}

	// Auto-detect hostname if not set
	hostname := getEnv("HOSTNAME", "")
	if hostname == "" {
		if sysHostname, err := os.Hostname(); err == nil {
			hostname = sysHostname
		}
	}

	// Auto-detect IP address if not set
	ipAddress := getEnv("IP_ADDRESS", "")
	if ipAddress == "" {
		if detectedIP := detectLocalIP(); detectedIP != "" {
			ipAddress = detectedIP
		}
	}

	// Auto-detect OS type if not set
	osType := getEnv("OS_TYPE", "")
	if osType == "" {
		osType = "linux" // Default assumption for service deployments
	}

	// Log some environment variables for debugging (without sensitive data)
	log.Printf("Environment check:")
	log.Printf("  - AGENT_ID: %s", getEnvSafe("AGENT_ID"))
	log.Printf("  - POCKETBASE_ENABLED: %s", getEnvSafe("POCKETBASE_ENABLED"))
	log.Printf("  - POCKETBASE_URL: %s", getEnvSafe("POCKETBASE_URL"))
	log.Printf("  - SERVER_NAME: %s", getEnvSafe("SERVER_NAME"))
	log.Printf("  - HOSTNAME (detected): %s", hostname)
	log.Printf("  - IP_ADDRESS (detected): %s", ipAddress)

	cfg := &Config{
		// Basic configuration with minimal defaults
		ServerURL:            getEnv("SERVER_URL", ""),
		APIKey:               getEnv("API_KEY", ""),
		PocketBaseEnabled:    getBoolEnv("POCKETBASE_ENABLED", true), // Default to true
		PocketBaseURL:        getEnv("POCKETBASE_URL", ""),
		CheckInterval:        getDurationEnv("CHECK_INTERVAL", 30*time.Second),
		ReportInterval:       getDurationEnv("REPORT_INTERVAL", 5*time.Minute),
		CommandCheckInterval: getDurationEnv("COMMAND_CHECK_INTERVAL", 10*time.Second),
		AgentID:              getEnv("AGENT_ID", "monitoring-agent-001"), // Provide default
		MaxRetries:           getIntEnv("MAX_RETRIES", 3),
		RequestTimeout:       getDurationEnv("REQUEST_TIMEOUT", 10*time.Second),
		HealthCheckPort:      getIntEnv("HEALTH_CHECK_PORT", 8081),
		RemoteControlEnabled: getBoolEnv("REMOTE_CONTROL_ENABLED", true), // Default to true
		
		// Server identification - use detected values as fallbacks
		ServerName:   getEnv("SERVER_NAME", hostname), // Use hostname as fallback
		Hostname:     hostname,
		IPAddress:    ipAddress,
		OSType:       osType,
		ServerToken:  getEnv("SERVER_TOKEN", ""),
	}

	// Validate required configuration
	if err := validateConfig(cfg); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %v", err)
	}

	return cfg, nil
}

func detectLocalIP() string {
	// Try to get the local IP address by connecting to a remote address
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		log.Printf("Could not detect local IP address: %v", err)
		return ""
	}
	defer conn.Close()
	
	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP.String()
}

func validateConfig(cfg *Config) error {
	var errors []string

	// Basic validation
	if cfg.AgentID == "" {
		errors = append(errors, "AGENT_ID is required")
	}

	// Validate PocketBase configuration if enabled
	if cfg.PocketBaseEnabled {
		if cfg.PocketBaseURL == "" {
			errors = append(errors, "POCKETBASE_URL is required when POCKETBASE_ENABLED=true")
		}
		if cfg.ServerName == "" {
			errors = append(errors, "SERVER_NAME is required when POCKETBASE_ENABLED=true (or will use hostname as fallback)")
		}
		if cfg.ServerToken == "" {
			errors = append(errors, "SERVER_TOKEN is required when POCKETBASE_ENABLED=true")
		}
		// IP_ADDRESS and HOSTNAME are now auto-detected, so no longer required
	}

	// Validate fallback HTTP configuration if PocketBase is disabled
	if !cfg.PocketBaseEnabled {
		if cfg.ServerURL == "" {
			errors = append(errors, "SERVER_URL is required when POCKETBASE_ENABLED=false")
		}
		if cfg.APIKey == "" {
			log.Printf("Warning: API_KEY not set for HTTP fallback")
		}
	}

	if len(errors) > 0 {
		errorMsg := "Configuration errors:\n"
		for _, err := range errors {
			errorMsg += fmt.Sprintf("  - %s\n", err)
		}
		return fmt.Errorf(errorMsg)
	}

	log.Printf("Configuration validation passed")
	return nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvSafe(key string) string {
	value := os.Getenv(key)
	if value == "" {
		return "(not set)"
	}
	// Don't log sensitive values completely
	if key == "SERVER_TOKEN" || key == "API_KEY" {
		if len(value) > 8 {
			return value[:4] + "****" + value[len(value)-4:]
		}
		return "****"
	}
	return value
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