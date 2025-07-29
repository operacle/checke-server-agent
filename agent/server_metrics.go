
package agent

import (
	"fmt"
	"log"
	"runtime"
	"time"

	pbClient "monitoring-agent/pocketbase"
)

func (a *Agent) gatherServerMetrics() pbClient.ServerRecord {
	collector := NewSystemCollector()
	
	// Get comprehensive system information
	sysInfo := collector.GetSystemInfo()
	
	// Get real memory data
	ramUsed, ramTotal, _ := collector.GetMemoryUsage()
	
	// Get real disk data
	diskUsed, diskTotal, _ := collector.GetDiskUsage()
	
	// Get real CPU usage with improved accuracy
	cpuUsage := collector.GetCPUUsage()
	
	// Check Docker availability - but don't override PocketBase setting
	dockerAvailable := collector.IsDockerAvailable()
	
	// Format comprehensive system info
	systemInfoString := fmt.Sprintf("%s %s | %s | Kernel: %s | CPU: %s (%d cores) | RAM: %.1f GB | Go %s | IP: %s | Docker: %t", 
		sysInfo.OSName, 
		sysInfo.OSVersion,
		sysInfo.Architecture,
		sysInfo.KernelVersion,
		sysInfo.CPUModel,
		sysInfo.CPUCores,
		float64(sysInfo.TotalRAM)/1024/1024/1024,
		sysInfo.GoVersion,
		sysInfo.IPAddress,
		dockerAvailable,
	)
	
	return pbClient.ServerRecord{
		ID:             a.serverRecord.ID, // Use existing record ID
		ServerID:       a.config.AgentID,
		Name:           a.config.ServerName,
		Hostname:       sysInfo.Hostname, // Use real hostname
		IPAddress:      sysInfo.IPAddress, // Use real IP address
		OSType:         sysInfo.OSType,    // Use real OS type
		Status:         "up",
		Uptime:         a.getUptimeString(),
		RAMTotal:       ramTotal,
		RAMUsed:        ramUsed,
		CPUCores:       runtime.NumCPU(),
		CPUUsage:       cpuUsage,
		DiskTotal:      diskTotal,
		DiskUsed:       diskUsed,
		LastChecked:    pbClient.FlexibleTime{Time: time.Now()},
		ServerToken:    a.config.ServerToken,
		Connection:     "connected",
		SystemInfo:     systemInfoString, // Comprehensive system info
		// Preserve the Docker setting from PocketBase - don't override it
		Docker:         a.serverRecord.Docker,
		Timestamp:      time.Now().Format(time.RFC3339),
		// Preserve the existing check_interval from the server record instead of overwriting it
		CheckInterval:  a.serverRecord.CheckInterval,
	}
}

func (a *Agent) gatherDetailedServerMetrics() pbClient.ServerMetricsRecord {
	collector := NewSystemCollector()
	
	// Get real memory data
	ramUsed, ramTotal, ramPercentage := collector.GetMemoryUsage()
	ramFree := ramTotal - ramUsed
	
	// Get accurate CPU data with improved calculation
	cpuUsage := collector.GetCPUUsage()
	cpuFree := 100.0 - cpuUsage
	
	// Get real disk data
	diskUsed, diskTotal, diskPercentage := collector.GetDiskUsage()
	diskFree := diskTotal - diskUsed
	
	// Get real network data
	networkStats := collector.GetNetworkStats()
	
	// Format values with units and proper precision
	ramTotalStr := fmt.Sprintf("%.2f GB", float64(ramTotal)/1024/1024/1024)
	ramUsedStr := fmt.Sprintf("%.2f GB (%.1f%%)", float64(ramUsed)/1024/1024/1024, ramPercentage)
	ramFreeStr := fmt.Sprintf("%.2f GB", float64(ramFree)/1024/1024/1024)
	
	cpuCoresStr := fmt.Sprintf("%d", runtime.NumCPU())
	cpuUsageStr := fmt.Sprintf("%.2f%%", cpuUsage)
	cpuFreeStr := fmt.Sprintf("%.2f%%", cpuFree)
	
	diskTotalStr := fmt.Sprintf("%.2f GB", float64(diskTotal)/1024/1024/1024)
	diskUsedStr := fmt.Sprintf("%.2f GB (%.1f%%)", float64(diskUsed)/1024/1024/1024, diskPercentage)
	diskFreeStr := fmt.Sprintf("%.2f GB", float64(diskFree)/1024/1024/1024)
	
	return pbClient.ServerMetricsRecord{
		ServerID:        a.config.AgentID,
		Timestamp:       time.Now(),
		RAMTotal:        ramTotalStr,
		RAMUsed:         ramUsedStr,
		RAMFree:         ramFreeStr,
		CPUCores:        cpuCoresStr,
		CPUUsage:        cpuUsageStr,
		CPUFree:         cpuFreeStr,
		DiskTotal:       diskTotalStr,
		DiskUsed:        diskUsedStr,
		DiskFree:        diskFreeStr,
		Status:          "healthy",
		NetworkRxBytes:  int64(networkStats.BytesReceived),
		NetworkTxBytes:  int64(networkStats.BytesSent),
		NetworkRxSpeed:  int64(networkStats.PacketsReceived), // Now contains RX speed (bytes/sec)
		NetworkTxSpeed:  int64(networkStats.PacketsSent),     // Now contains TX speed (bytes/sec)
	}
}

func (a *Agent) sendServerMetrics(serverMetrics pbClient.ServerRecord) error {
	if a.pocketBase == nil {
		return fmt.Errorf("no PocketBase client available")
	}
	
	return a.pocketBase.SaveServerMetrics(serverMetrics)
}

func (a *Agent) sendDetailedServerMetrics(metrics pbClient.ServerMetricsRecord) error {
	if a.pocketBase == nil {
		return fmt.Errorf("no PocketBase client available")
	}
	
	return a.pocketBase.SaveServerMetricsRecord(metrics)
}

func (a *Agent) getUptimeString() string {
	collector := NewSystemCollector()
	uptimeSeconds := collector.GetSystemUptime()
	
	days := uptimeSeconds / 86400
	hours := (uptimeSeconds % 86400) / 3600
	minutes := (uptimeSeconds % 3600) / 60
	
	return fmt.Sprintf("%dd %dh %dm", days, hours, minutes)
}

func (a *Agent) gatherDockerContainers() []pbClient.DockerRecord {
	var dockerRecords []pbClient.DockerRecord
	
	// Check if Docker monitoring is enabled in PocketBase AND Docker is available
	if !a.serverRecord.Docker.Value {
		log.Printf("Docker monitoring is disabled in PocketBase")
		return dockerRecords // Return empty slice if Docker is disabled in PocketBase
	}
	
	collector := NewSystemCollector()
	
	// Check if Docker is actually available on the system
	if !collector.IsDockerAvailable() {
		log.Printf("Docker is not available on system, but monitoring is enabled in PocketBase")
		return dockerRecords
	}
	
	dockerInfo := collector.GetDockerInfo()
	
	if !dockerInfo.Available {
		log.Printf("Docker info indicates Docker is not available")
		return dockerRecords
	}
	
	if len(dockerInfo.Containers) == 0 {
		log.Printf("No Docker containers found")
		return dockerRecords
	}
	
	log.Printf("Found %d Docker containers, collecting data", len(dockerInfo.Containers))
	sysInfo := collector.GetSystemInfo()
	
	for _, container := range dockerInfo.Containers {
		dockerRecord := pbClient.DockerRecord{
			DockerID:       container.ID,
			Name:           container.Name,
			Hostname:       sysInfo.Hostname,
			IPAddress:      sysInfo.IPAddress,
			OSTemplate:     fmt.Sprintf("Docker/%s", dockerInfo.Version),
			Uptime:         container.Uptime,
			RAMTotal:       container.MemTotal,
			RAMUsed:        container.MemUsage,
			CPUCores:       runtime.NumCPU(),
			CPUUsage:       container.CPUUsage,
			DiskTotal:      container.DiskTotal,
			DiskUsed:       container.DiskUsage,
			LastChecked:    pbClient.FlexibleTime{Time: time.Now()},
			Timestamp:      time.Now().Format(time.RFC3339),
			Status:         container.Status,
		}
		
		dockerRecords = append(dockerRecords, dockerRecord)
	}
	
	log.Printf("Prepared %d Docker records for sending", len(dockerRecords))
	return dockerRecords
}

func (a *Agent) gatherDockerMetrics() []pbClient.DockerMetricsRecord {
	var dockerMetrics []pbClient.DockerMetricsRecord
	
	// Check if Docker monitoring is enabled in PocketBase AND Docker is available
	if !a.serverRecord.Docker.Value {
		log.Printf("Docker monitoring is disabled in PocketBase")
		return dockerMetrics // Return empty slice if Docker is disabled in PocketBase
	}
	
	collector := NewSystemCollector()
	
	// Check if Docker is actually available on the system
	if !collector.IsDockerAvailable() {
		log.Printf("Docker is not available on system, but monitoring is enabled in PocketBase")
		return dockerMetrics
	}
	
	dockerInfo := collector.GetDockerInfo()
	
	if !dockerInfo.Available {
		log.Printf("Docker info indicates Docker is not available")
		return dockerMetrics
	}
	
	if len(dockerInfo.Containers) == 0 {
		log.Printf("No Docker containers found for metrics")
		return dockerMetrics
	}
	
	log.Printf("Collecting metrics for %d Docker containers", len(dockerInfo.Containers))
	
	for _, container := range dockerInfo.Containers {
		// Calculate derived values
		ramFree := container.MemTotal - container.MemUsage
		if ramFree < 0 {
			ramFree = 0
		}
		
		cpuFree := 100.0 - container.CPUUsage
		if cpuFree < 0 {
			cpuFree = 0
		}
		
		diskFree := container.DiskTotal - container.DiskUsage
		if diskFree < 0 {
			diskFree = 0
		}
		
		// Calculate percentages safely
		var ramPercentage, diskPercentage float64
		if container.MemTotal > 0 {
			ramPercentage = float64(container.MemUsage) / float64(container.MemTotal) * 100
		}
		if container.DiskTotal > 0 {
			diskPercentage = float64(container.DiskUsage) / float64(container.DiskTotal) * 100
		}
		
		// Format values with units and proper precision
		ramTotalStr := fmt.Sprintf("%.2f GB", float64(container.MemTotal)/1024/1024/1024)
		ramUsedStr := fmt.Sprintf("%.2f GB (%.1f%%)", float64(container.MemUsage)/1024/1024/1024, ramPercentage)
		ramFreeStr := fmt.Sprintf("%.2f GB", float64(ramFree)/1024/1024/1024)
		
		cpuCoresStr := fmt.Sprintf("%d", runtime.NumCPU())
		cpuUsageStr := fmt.Sprintf("%.2f%%", container.CPUUsage)
		cpuFreeStr := fmt.Sprintf("%.2f%%", cpuFree)
		
		diskTotalStr := fmt.Sprintf("%.2f GB", float64(container.DiskTotal)/1024/1024/1024)
		diskUsedStr := fmt.Sprintf("%.2f GB (%.1f%%)", float64(container.DiskUsage)/1024/1024/1024, diskPercentage)
		diskFreeStr := fmt.Sprintf("%.2f GB", float64(diskFree)/1024/1024/1024)
		
		// Create Docker metrics record with real data
		dockerMetric := pbClient.DockerMetricsRecord{
			DockerID:        container.ID,
			Timestamp:       time.Now(),
			RAMTotal:        ramTotalStr,
			RAMUsed:         ramUsedStr,
			RAMFree:         ramFreeStr,
			CPUCores:        cpuCoresStr,
			CPUUsage:        cpuUsageStr,
			CPUFree:         cpuFreeStr,
			DiskTotal:       diskTotalStr,
			DiskUsed:        diskUsedStr,
			DiskFree:        diskFreeStr,
			Status:          container.Status,
			NetworkRxBytes:  container.NetworkRxBytes,
			NetworkTxBytes:  container.NetworkTxBytes,
			NetworkRxSpeed:  container.NetworkRxSpeed,
			NetworkTxSpeed:  container.NetworkTxSpeed,
		}
		
		dockerMetrics = append(dockerMetrics, dockerMetric)
	}
	
	log.Printf("Prepared %d Docker metrics records for sending", len(dockerMetrics))
	return dockerMetrics
}

func (a *Agent) sendDockerRecords(dockerRecords []pbClient.DockerRecord) error {
	if a.pocketBase == nil {
		return fmt.Errorf("no PocketBase client available")
	}
	
	if len(dockerRecords) == 0 {
		log.Printf("No Docker records to send")
		return nil
	}
	
	log.Printf("Sending %d Docker records to PocketBase", len(dockerRecords))
	
	for _, docker := range dockerRecords {
		// Try to find existing Docker record
		existingDocker, err := a.pocketBase.GetDockerByID(docker.DockerID)
		if err != nil {
			// Docker record doesn't exist, create new one
			log.Printf("Creating new Docker record for container %s (%s)", docker.Name, docker.DockerID)
			if err := a.pocketBase.SaveDockerRecord(docker); err != nil {
				log.Printf("Failed to save docker record %s: %v", docker.DockerID, err)
				return fmt.Errorf("failed to save docker record %s: %v", docker.DockerID, err)
			}
			log.Printf("Successfully created Docker record for %s", docker.Name)
		} else {
			// Update existing Docker record
			log.Printf("Updating existing Docker record for container %s (%s)", docker.Name, docker.DockerID)
			if err := a.pocketBase.UpdateDockerRecord(existingDocker.ID, docker); err != nil {
				log.Printf("Failed to update docker record %s: %v", docker.DockerID, err)
				return fmt.Errorf("failed to update docker record %s: %v", docker.DockerID, err)
			}
			log.Printf("Successfully updated Docker record for %s", docker.Name)
		}
	}
	
	log.Printf("Successfully sent all Docker records")
	return nil
}

func (a *Agent) sendDockerMetrics(dockerMetrics []pbClient.DockerMetricsRecord) error {
	if a.pocketBase == nil {
		return fmt.Errorf("no PocketBase client available")
	}
	
	if len(dockerMetrics) == 0 {
		log.Printf("No Docker metrics to send")
		return nil
	}
	
	log.Printf("Sending %d Docker metrics records to PocketBase", len(dockerMetrics))
	
	for _, metric := range dockerMetrics {
		log.Printf("Sending metrics for Docker container %s", metric.DockerID)
		if err := a.pocketBase.SaveDockerMetricsRecord(metric); err != nil {
			log.Printf("Failed to save docker metrics for %s: %v", metric.DockerID, err)
			return fmt.Errorf("failed to save docker metrics for %s: %v", metric.DockerID, err)
		}
		log.Printf("Successfully sent metrics for Docker container %s", metric.DockerID)
	}
	
	log.Printf("Successfully sent all Docker metrics")
	return nil
}