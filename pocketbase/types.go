package pocketbase

import (
	"encoding/json"
	"strconv"
	"strings"
	"time"
)

// FlexibleTime handles multiple timestamp formats from PocketBase
type FlexibleTime struct {
	time.Time
}

func (ft *FlexibleTime) UnmarshalJSON(data []byte) error {
	// Remove quotes from JSON string
	timeStr := strings.Trim(string(data), `"`)
	
	// Try different time formats
	timeFormats := []string{
		time.RFC3339,
		"2006-01-02T15:04:05.000Z",
		"2006-01-02 15:04:05.000Z",
		"2006-01-02T15:04:05Z",
		"2006-01-02 15:04:05Z",
		time.RFC3339Nano,
	}
	
	for _, format := range timeFormats {
		if t, err := time.Parse(format, timeStr); err == nil {
			ft.Time = t
			return nil
		}
	}
	
	// If all formats fail, try the current time
	ft.Time = time.Now()
	return nil
}

func (ft FlexibleTime) MarshalJSON() ([]byte, error) {
	return json.Marshal(ft.Time.Format(time.RFC3339))
}

// FlexibleInt handles both string and int values from PocketBase
type FlexibleInt struct {
	Value int
}

func (fi *FlexibleInt) UnmarshalJSON(data []byte) error {
	// Try to unmarshal as int first
	var intVal int
	if err := json.Unmarshal(data, &intVal); err == nil {
		fi.Value = intVal
		return nil
	}
	
	// Try to unmarshal as string and convert to int
	var strVal string
	if err := json.Unmarshal(data, &strVal); err == nil {
		if strVal == "" {
			fi.Value = 0
			return nil
		}
		if intVal, err := strconv.Atoi(strVal); err == nil {
			fi.Value = intVal
			return nil
		}
	}
	
	// Default to 0 if all fails
	fi.Value = 0
	return nil
}

func (fi FlexibleInt) MarshalJSON() ([]byte, error) {
	return json.Marshal(fi.Value)
}

// FlexibleBool handles both string and bool values from PocketBase
type FlexibleBool struct {
	Value bool
}

func (fb *FlexibleBool) UnmarshalJSON(data []byte) error {
	// Try to unmarshal as bool first
	var boolVal bool
	if err := json.Unmarshal(data, &boolVal); err == nil {
		fb.Value = boolVal
		return nil
	}
	
	// Try to unmarshal as string and convert to bool
	var strVal string
	if err := json.Unmarshal(data, &strVal); err == nil {
		switch strings.ToLower(strVal) {
		case "true", "1", "yes":
			fb.Value = true
		case "false", "0", "no", "":
			fb.Value = false
		default:
			fb.Value = false
		}
		return nil
	}
	
	// Default to false if all fails
	fb.Value = false
	return nil
}

func (fb FlexibleBool) MarshalJSON() ([]byte, error) {
	return json.Marshal(fb.Value)
}

type ServerRecord struct {
	ID             string       `json:"id,omitempty"`
	ServerID       string       `json:"server_id"`
	Name           string       `json:"name"`
	Hostname       string       `json:"hostname"`
	IPAddress      string       `json:"ip_address"`
	OSType         string       `json:"os_type"`
	Status         string       `json:"status"`
	Uptime         string       `json:"uptime"`
	RAMTotal       int64        `json:"ram_total"`
	RAMUsed        int64        `json:"ram_used"`
	CPUCores       int          `json:"cpu_cores"`
	CPUUsage       float64      `json:"cpu_usage"`
	DiskTotal      int64        `json:"disk_total"`
	DiskUsed       int64        `json:"disk_used"`
	LastChecked    FlexibleTime `json:"last_checked"`
	ServerToken    string       `json:"server_token"`
	TemplateID     string       `json:"template_id"`
	NotificationID string       `json:"notification_id"`
	Timestamp      string       `json:"timestamp"`
	Connection     string       `json:"connection"`
	SystemInfo     string       `json:"system_info"`
	AgentStatus    string       `json:"agent_status,omitempty"`
	CheckInterval  FlexibleInt  `json:"check_interval,omitempty"`
	Docker         FlexibleBool `json:"docker,omitempty"`
	Created        FlexibleTime `json:"created,omitempty"`
	Updated        FlexibleTime `json:"updated,omitempty"`
}

type ServerMetricsRecord struct {
	ID              string       `json:"id,omitempty"`
	ServerID        string       `json:"server_id"`
	Timestamp       time.Time    `json:"timestamp"`
	RAMTotal        string       `json:"ram_total"`
	RAMUsed         string       `json:"ram_used"`
	RAMFree         string       `json:"ram_free"`
	CPUCores        string       `json:"cpu_cores"`
	CPUUsage        string       `json:"cpu_usage"`
	CPUFree         string       `json:"cpu_free"`
	DiskTotal       string       `json:"disk_total"`
	DiskUsed        string       `json:"disk_used"`
	DiskFree        string       `json:"disk_free"`
	Status          string       `json:"status"`
	NetworkRxBytes  int64        `json:"network_rx_bytes"`
	NetworkTxBytes  int64        `json:"network_tx_bytes"`
	NetworkRxSpeed  int64        `json:"network_rx_speed"`
	NetworkTxSpeed  int64        `json:"network_tx_speed"`
	Created         FlexibleTime `json:"created,omitempty"`
	Updated         FlexibleTime `json:"updated,omitempty"`
}

type MetricsRecord struct {
	AgentID       string    `json:"agent_id"`
	Timestamp     time.Time `json:"timestamp"`
	CPUUsage      float64   `json:"cpu_usage"`
	MemoryUsage   float64   `json:"memory_usage"`
	DiskUsage     float64   `json:"disk_usage"`
	NetworkStats  string    `json:"network_stats"`
	Uptime        int64     `json:"uptime"`
	GoRoutines    int       `json:"goroutines"`
	Status        string    `json:"status"`
}

type AgentStatusRecord struct {
	ID          string    `json:"id,omitempty"`
	AgentID     string    `json:"agent_id"`
	Status      string    `json:"status"`
	LastSeen    time.Time `json:"last_seen"`
	Version     string    `json:"version"`
	Message     string    `json:"message"`
}

type CommandRecord struct {
	ID         string       `json:"id"`
	AgentID    string       `json:"agent_id"`
	Command    string       `json:"command"`
	Parameters string       `json:"parameters"`
	Executed   bool         `json:"executed"`
	CreatedAt  FlexibleTime `json:"created"`
}

// DockerRecord represents a Docker container record
type DockerRecord struct {
	ID             string       `json:"id,omitempty"`
	DockerID       string       `json:"docker_id"`
	Name           string       `json:"name"`
	Hostname       string       `json:"hostname"`
	IPAddress      string       `json:"ip_address"`
	OSTemplate     string       `json:"os_template"`
	Uptime         string       `json:"uptime"`
	RAMTotal       int64        `json:"ram_total"`
	RAMUsed        int64        `json:"ram_used"`
	CPUCores       int          `json:"cpu_cores"`
	CPUUsage       float64      `json:"cpu_usage"`
	DiskTotal      int64        `json:"disk_total"`
	DiskUsed       int64        `json:"disk_used"`
	LastChecked    FlexibleTime `json:"last_checked"`
	TemplateID     string       `json:"template_id"`
	NotificationID string       `json:"notification_id"`
	Timestamp      string       `json:"timestamp"`
	Status         string       `json:"status"`
	Created        FlexibleTime `json:"created,omitempty"`
	Updated        FlexibleTime `json:"updated,omitempty"`
}

// DockerMetricsRecord represents Docker container metrics
type DockerMetricsRecord struct {
	ID              string       `json:"id,omitempty"`
	DockerID        string       `json:"docker_id"`
	Timestamp       time.Time    `json:"timestamp"`
	RAMTotal        string       `json:"ram_total"`
	RAMUsed         string       `json:"ram_used"`
	RAMFree         string       `json:"ram_free"`
	CPUCores        string       `json:"cpu_cores"`
	CPUUsage        string       `json:"cpu_usage"`
	CPUFree         string       `json:"cpu_free"`
	DiskTotal       string       `json:"disk_total"`
	DiskUsed        string       `json:"disk_used"`
	DiskFree        string       `json:"disk_free"`
	Status          string       `json:"status"`
	NetworkRxBytes  int64        `json:"network_rx_bytes"`
	NetworkTxBytes  int64        `json:"network_tx_bytes"`
	NetworkRxSpeed  int64        `json:"network_rx_speed"`
	NetworkTxSpeed  int64        `json:"network_tx_speed"`
	Created         FlexibleTime `json:"created,omitempty"`
	Updated         FlexibleTime `json:"updated,omitempty"`
}