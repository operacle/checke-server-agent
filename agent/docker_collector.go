
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

// IsDockerAvailable checks if Docker service is running with enhanced detection
func (sc *SystemCollector) IsDockerAvailable() bool {
	// First check if Docker socket exists
	if !sc.checkDockerSocket() {
		return false
	}
	
	// Try multiple approaches to detect Docker command availability
	dockerPaths := []string{
		"/usr/bin/docker",
		"/usr/local/bin/docker",
		"/bin/docker",
		"/usr/sbin/docker",
		"docker", // fallback to PATH
	}
	
	for _, dockerPath := range dockerPaths {
		if err := sc.tryDockerCommand(dockerPath); err == nil {
			return true
		}
	}
	
	return false
}

// tryDockerCommand attempts to run docker version command with specific binary path
func (sc *SystemCollector) tryDockerCommand(dockerPath string) error {
	cmd := exec.Command(dockerPath, "version", "--format", "{{.Server.Version}}")
	
	// Set environment variables for systemd service execution
	cmd.Env = append(os.Environ(),
		"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
	)
	
	_, err := cmd.CombinedOutput()
	return err
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

// getDockerVersion gets Docker version with enhanced path detection and better error handling
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
			version := strings.TrimSpace(string(output))
			return version
		}
	}
	
	return "permission_denied"
}

// getDockerContainers gets statistics for all running containers with improved error handling
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
	
	// Try different Docker binary paths to list containers
	for _, dockerPath := range dockerPaths {
		cmd = exec.Command(dockerPath, "ps", "--all", "--format", "{{.ID}}\t{{.Names}}\t{{.Status}}\t{{.RunningFor}}")
		cmd.Env = append(os.Environ(),
			"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
		)
		
		output, err = cmd.CombinedOutput()
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

		parts := strings.Split(line, "\t")
		if len(parts) < 3 {
			continue
		}

		containerID := strings.TrimSpace(parts[0])
		containerName := strings.TrimSpace(parts[1])
		status := strings.TrimSpace(parts[2])
		uptime := ""
		if len(parts) > 3 {
			uptime = strings.TrimSpace(parts[3])
		}

		// Get detailed stats for this container
		stats := sc.getContainerStats(containerID, containerName, status, uptime)
		if stats.ID != "" {
			containers = append(containers, stats)
		}
	}

	return containers
}

// getContainerStats gets detailed statistics for a specific container with better error handling
func (sc *SystemCollector) getContainerStats(containerID, containerName, status, uptime string) DockerStats {
	stats := DockerStats{
		ID:     containerID,
		Name:   containerName,
		Status: status,
		Uptime: uptime,
	}

	// Skip stats collection for stopped containers
	if !strings.Contains(strings.ToLower(status), "up") {
		stats.CPUUsage = 0.0
		stats.MemUsage = 0
		stats.MemTotal = 1024 * 1024 * 1024 // 1GB default
		stats.DiskUsage = 0
		stats.DiskTotal = 10 * 1024 * 1024 * 1024 // 10GB default
		return stats
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
			"{{.CPUPerc}}\t{{.MemUsage}}\t{{.NetIO}}\t{{.BlockIO}}", containerID)
		cmd.Env = append(os.Environ(),
			"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
		)
		
		output, err = cmd.CombinedOutput()
		if err == nil {
			break
		}
	}
	
	if err != nil {
		// Set default values when stats collection fails
		stats.CPUUsage = 0.0
		stats.MemUsage = 512 * 1024 * 1024 // 512MB default
		stats.MemTotal = 2 * 1024 * 1024 * 1024 // 2GB default
		stats.DiskUsage = 1024 * 1024 * 1024 // 1GB default
		stats.DiskTotal = 10 * 1024 * 1024 * 1024 // 10GB default
		return stats
	}

	statsLine := strings.TrimSpace(string(output))
	if statsLine == "" {
		return stats
	}

	// Parse the stats line
	fields := strings.Split(statsLine, "\t")
	
	if len(fields) >= 4 {
		// Parse CPU usage (remove % sign)
		cpuStr := strings.TrimSuffix(strings.TrimSpace(fields[0]), "%")
		if cpuUsage, err := strconv.ParseFloat(cpuStr, 64); err == nil {
			stats.CPUUsage = cpuUsage
		}

		// Parse memory usage (format: "used / total")
		memUsage := strings.TrimSpace(fields[1])
		stats.MemUsage, stats.MemTotal = sc.parseMemoryUsage(memUsage)

		// Parse network I/O (format: "rx / tx")
		netIO := strings.TrimSpace(fields[2])
		stats.NetworkRxBytes, stats.NetworkTxBytes = sc.parseNetworkIO(netIO)

		// Parse block I/O for disk usage (format: "read / write")
		blockIO := strings.TrimSpace(fields[3])
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
		return 512 * 1024 * 1024, 2 * 1024 * 1024 * 1024 // Default 512MB / 2GB
	}

	used = sc.parseDataSize(strings.TrimSpace(parts[0]))
	total = sc.parseDataSize(strings.TrimSpace(parts[1]))
	
	// Ensure we have reasonable values
	if used == 0 {
		used = 512 * 1024 * 1024 // 512MB default
	}
	if total == 0 {
		total = 2 * 1024 * 1024 * 1024 // 2GB default
	}
	
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
	if sizeStr == "" || sizeStr == "0B" || sizeStr == "0" {
		return 0
	}
	
	// Use regex to extract number and unit
	re := regexp.MustCompile(`^([0-9.]+)\s*([A-Za-z]*)$`)
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
	case "kb", "k":
		multiplier = 1000
	case "mb", "m":
		multiplier = 1000 * 1000
	case "gb", "g":
		multiplier = 1000 * 1000 * 1000
	case "tb", "t":
		multiplier = 1000 * 1000 * 1000 * 1000
	case "kib":
		multiplier = 1024
	case "mib":
		multiplier = 1024 * 1024
	case "gib":
		multiplier = 1024 * 1024 * 1024
	case "tib":
		multiplier = 1024 * 1024 * 1024 * 1024
	case "b", "":
		multiplier = 1
	default:
		multiplier = 1
	}

	result := int64(size * float64(multiplier))
	return result
}

// getContainerDiskTotal gets container disk total using docker system df
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
			if size, err := strconv.ParseInt(sizeStr, 10, 64); err == nil && size > 0 {
				// Add some buffer for container filesystem
				totalSize := size + (2 * 1024 * 1024 * 1024) // Add 2GB buffer
				return totalSize
			}
		}
	}
	
	// Fallback: return default size
	defaultSize := int64(10 * 1024 * 1024 * 1024) // Default 10GB
	return defaultSize
}