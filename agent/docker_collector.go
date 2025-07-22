package agent

import (
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

// DockerStats represents Docker container statistics
type DockerStats struct {
	ID        string
	Name      string
	CPUUsage  float64
	MemUsage  int64
	MemTotal  int64
	DiskUsage int64
	DiskTotal int64
	Status    string
	Uptime    string
	NetworkRxBytes int64
	NetworkTxBytes int64
	NetworkRxSpeed int64
	NetworkTxSpeed int64
}

// DockerInfo represents general Docker system information
type DockerInfo struct {
	Available bool
	Version   string
	Containers []DockerStats
}

// IsDockerAvailable checks if Docker service is running with enhanced detection for systemd
func (sc *SystemCollector) IsDockerAvailable() bool {
	// Try multiple approaches to detect Docker availability
	
	// Method 1: Try standard docker command with full PATH
	if sc.tryDockerCommand("/usr/bin/docker") {
		return true
	}
	
	// Method 2: Try docker command from common locations
	dockerPaths := []string{
		"/usr/local/bin/docker",
		"/bin/docker",
		"/usr/sbin/docker",
		"docker", // fallback to PATH
	}
	
	for _, dockerPath := range dockerPaths {
		if sc.tryDockerCommand(dockerPath) {
			return true
		}
	}
	
	// Method 3: Check if Docker socket exists
	if sc.checkDockerSocket() {
		return true
	}
	
	return false
}

// tryDockerCommand attempts to run docker version command with specific binary path
func (sc *SystemCollector) tryDockerCommand(dockerPath string) bool {
	cmd := exec.Command(dockerPath, "version", "--format", "{{.Server.Version}}")
	
	// Set environment variables for systemd service execution
	cmd.Env = append(os.Environ(),
		"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
	)
	
	err := cmd.Run()
	return err == nil
}

// checkDockerSocket checks if Docker socket is accessible
func (sc *SystemCollector) checkDockerSocket() bool {
	socketPaths := []string{
		"/var/run/docker.sock",
		"/run/docker.sock",
	}
	
	for _, socketPath := range socketPaths {
		if _, err := os.Stat(socketPath); err == nil {
			return true
		}
	}
	
	return false
}

// GetDockerInfo returns comprehensive Docker information
func (sc *SystemCollector) GetDockerInfo() DockerInfo {
	dockerInfo := DockerInfo{
		Available: sc.IsDockerAvailable(),
	}

	if !dockerInfo.Available {
		return dockerInfo
	}

	// Get Docker version
	dockerInfo.Version = sc.getDockerVersion()
	
	// Get container statistics
	dockerInfo.Containers = sc.getDockerContainers()

	return dockerInfo
}

// getDockerVersion gets Docker version with enhanced path detection
func (sc *SystemCollector) getDockerVersion() string {
	dockerPaths := []string{
		"/usr/bin/docker",
		"/usr/local/bin/docker",
		"/bin/docker",
		"docker",
	}
	
	for _, dockerPath := range dockerPaths {
		cmd := exec.Command(dockerPath, "version", "--format", "{{.Server.Version}}")
		cmd.Env = append(os.Environ(),
			"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
		)
		
		output, err := cmd.Output()
		if err == nil {
			return strings.TrimSpace(string(output))
		}
	}
	
	return "unknown"
}

// getDockerContainers gets statistics for all running containers with enhanced path detection
func (sc *SystemCollector) getDockerContainers() []DockerStats {
	var containers []DockerStats
	
	dockerPaths := []string{
		"/usr/bin/docker",
		"/usr/local/bin/docker",
		"/bin/docker",
		"docker",
	}
	
	var cmd *exec.Cmd
	var output []byte
	var err error
	
	// Try different Docker binary paths
	for _, dockerPath := range dockerPaths {
		cmd = exec.Command(dockerPath, "ps", "--format", "{{.ID}}:{{.Names}}:{{.Status}}:{{.RunningFor}}")
		cmd.Env = append(os.Environ(),
			"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
		)
		
		output, err = cmd.Output()
		if err == nil {
			break
		}
	}
	
	if err != nil {
		return containers
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}

		parts := strings.Split(line, ":")
		if len(parts) < 3 {
			continue
		}

		containerID := parts[0]
		containerName := parts[1]
		status := parts[2]
		uptime := ""
		if len(parts) > 3 {
			uptime = parts[3]
		}

		// Get detailed stats for this container
		stats := sc.getContainerStats(containerID, containerName, status, uptime)
		if stats.ID != "" {
			containers = append(containers, stats)
		}
	}

	return containers
}

// getContainerStats gets detailed statistics for a specific container with enhanced Docker path detection
func (sc *SystemCollector) getContainerStats(containerID, containerName, status, uptime string) DockerStats {
	stats := DockerStats{
		ID:     containerID,
		Name:   containerName,
		Status: status,
		Uptime: uptime,
	}

	dockerPaths := []string{
		"/usr/bin/docker",
		"/usr/local/bin/docker",
		"/bin/docker",
		"docker",
	}
	
	var cmd *exec.Cmd
	var output []byte
	var err error
	
	// Try different Docker binary paths for stats command
	for _, dockerPath := range dockerPaths {
		cmd = exec.Command(dockerPath, "stats", "--no-stream", "--format", 
			"{{.CPUPerc}}|{{.MemUsage}}|{{.NetIO}}|{{.BlockIO}}", containerID)
		cmd.Env = append(os.Environ(),
			"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
		)
		
		output, err = cmd.Output()
		if err == nil {
			break
		}
	}
	
	if err != nil {
		// If stats command fails, try to get basic info
		stats.CPUUsage = 0.0
		stats.MemUsage = 0
		stats.MemTotal = 0
		return stats
	}

	statsLine := strings.TrimSpace(string(output))
	if statsLine == "" {
		return stats
	}

	// Parse the stats line
	fields := strings.Split(statsLine, "|")
	
	if len(fields) >= 4 {
		// Parse CPU usage (remove % sign)
		cpuStr := strings.TrimSuffix(fields[0], "%")
		if cpuUsage, err := strconv.ParseFloat(cpuStr, 64); err == nil {
			stats.CPUUsage = cpuUsage
		}

		// Parse memory usage (format: "used / total")
		memUsage := fields[1]
		stats.MemUsage, stats.MemTotal = sc.parseMemoryUsage(memUsage)

		// Parse network I/O (format: "rx / tx")
		netIO := fields[2]
		stats.NetworkRxBytes, stats.NetworkTxBytes = sc.parseNetworkIO(netIO)

		// Parse block I/O for disk usage (format: "read / write")
		blockIO := fields[3]
		diskRead, diskWrite := sc.parseBlockIO(blockIO)
		stats.DiskUsage = diskRead + diskWrite
		stats.DiskTotal = sc.getContainerDiskTotal(containerID)
	}

	// Calculate network speeds (simplified - bytes per second estimate)
	stats.NetworkRxSpeed = stats.NetworkRxBytes / 3600 // Rough hourly average
	stats.NetworkTxSpeed = stats.NetworkTxBytes / 3600 // Rough hourly average

	return stats
}

// parseMemoryUsage parses Docker memory usage string like "1.5GiB / 8GiB"
func (sc *SystemCollector) parseMemoryUsage(memUsage string) (used int64, total int64) {
	parts := strings.Split(memUsage, " / ")
	if len(parts) != 2 {
		return 0, 0
	}

	used = sc.parseDataSize(strings.TrimSpace(parts[0]))
	total = sc.parseDataSize(strings.TrimSpace(parts[1]))
	
	return used, total
}

// parseNetworkIO parses network I/O string like "1.2kB / 3.4kB"
func (sc *SystemCollector) parseNetworkIO(netIO string) (rxBytes int64, txBytes int64) {
	parts := strings.Split(netIO, " / ")
	if len(parts) != 2 {
		return 0, 0
	}

	rxBytes = sc.parseDataSize(strings.TrimSpace(parts[0]))
	txBytes = sc.parseDataSize(strings.TrimSpace(parts[1]))
	
	return rxBytes, txBytes
}

// parseBlockIO parses block I/O string like "1.2MB / 3.4MB"
func (sc *SystemCollector) parseBlockIO(blockIO string) (readBytes int64, writeBytes int64) {
	parts := strings.Split(blockIO, " / ")
	if len(parts) != 2 {
		return 0, 0
	}

	readBytes = sc.parseDataSize(strings.TrimSpace(parts[0]))
	writeBytes = sc.parseDataSize(strings.TrimSpace(parts[1]))
	
	return readBytes, writeBytes
}

// parseDataSize converts data size string to bytes (handles kB, MB, GB, KiB, MiB, GiB)
func (sc *SystemCollector) parseDataSize(sizeStr string) int64 {
	if sizeStr == "" || sizeStr == "0B" {
		return 0
	}
	
	// Use regex to extract number and unit
	re := regexp.MustCompile(`^([0-9.]+)\s*([A-Za-z]+)$`)
	matches := re.FindStringSubmatch(strings.TrimSpace(sizeStr))
	
	if len(matches) != 3 {
		return 0
	}
	
	numStr := matches[1]
	unit := strings.ToLower(matches[2])
	
	size, err := strconv.ParseFloat(numStr, 64)
	if err != nil {
		return 0
	}
	
	var multiplier int64 = 1
	
	switch unit {
	case "kb":
		multiplier = 1000
	case "mb":
		multiplier = 1000 * 1000
	case "gb":
		multiplier = 1000 * 1000 * 1000
	case "tb":
		multiplier = 1000 * 1000 * 1000 * 1000
	case "kib":
		multiplier = 1024
	case "mib":
		multiplier = 1024 * 1024
	case "gib":
		multiplier = 1024 * 1024 * 1024
	case "tib":
		multiplier = 1024 * 1024 * 1024 * 1024
	case "b":
		multiplier = 1
	default:
		multiplier = 1
	}

	return int64(size * float64(multiplier))
}

// getContainerDiskTotal gets container disk total using docker system df with enhanced path detection
func (sc *SystemCollector) getContainerDiskTotal(containerID string) int64 {
	dockerPaths := []string{
		"/usr/bin/docker",
		"/usr/local/bin/docker",
		"/bin/docker",
		"docker",
	}
	
	// Try docker inspect first
	for _, dockerPath := range dockerPaths {
		cmd := exec.Command(dockerPath, "inspect", "--format", "{{.SizeRootFs}}", containerID)
		cmd.Env = append(os.Environ(),
			"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
		)
		
		output, err := cmd.Output()
		if err == nil {
			sizeStr := strings.TrimSpace(string(output))
			if size, err := strconv.ParseInt(sizeStr, 10, 64); err == nil {
				// Add some buffer for container filesystem
				return size + (2 * 1024 * 1024 * 1024) // Add 2GB buffer
			}
		}
	}
	
	// Fallback: try to get from system df
	for _, dockerPath := range dockerPaths {
		cmd := exec.Command(dockerPath, "system", "df", "--format", "table {{.Size}}")
		cmd.Env = append(os.Environ(),
			"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
		)
		
		output, err := cmd.Output()
		if err == nil {
			lines := strings.Split(string(output), "\n")
			if len(lines) > 1 {
				sizeStr := strings.TrimSpace(lines[1])
				if size := sc.parseDataSize(sizeStr); size > 0 {
					return size
				}
			}
		}
	}
	
	return 10 * 1024 * 1024 * 1024 // Default 10GB
}