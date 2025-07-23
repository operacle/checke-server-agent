
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
	
	// Check Docker availability only if Docker is enabled in PocketBase
	var dockerAvailable bool
	if a.serverRecord.Docker.Value {
		dockerAvailable = collector.IsDockerAvailable()
	} else {
		dockerAvailable = false
	}
	
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
		Docker:         pbClient.FlexibleBool{Value: dockerAvailable},   // Set Docker availability based on PocketBase setting
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
	
	// Check if Docker monitoring is enabled in PocketBase before proceeding
	if !a.serverRecord.Docker.Value {
		return dockerRecords // Return empty slice if Docker is disabled
	}
	
	collector := NewSystemCollector()
	dockerInfo := collector.GetDockerInfo()
	
	if !dockerInfo.Available {
		return dockerRecords
	}
	
	if len(dockerInfo.Containers) == 0 {
		return dockerRecords
	}
	
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
	
	return dockerRecords
}

func (a *Agent) gatherDockerMetrics() []pbClient.DockerMetricsRecord {
	var dockerMetrics []pbClient.DockerMetricsRecord
	
	// Check if Docker monitoring is enabled in PocketBase before proceeding
	if !a.serverRecord.Docker.Value {
		return dockerMetrics // Return empty slice if Docker is disabled
	}
	
	collector := NewSystemCollector()
	dockerInfo := collector.GetDockerInfo()
	
	if !dockerInfo.Available {
		return dockerMetrics
	}
	
	if len(dockerInfo.Containers) == 0 {
		return dockerMetrics
	}
	
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
	
	return dockerMetrics
}

func (a *Agent) sendDockerRecords(dockerRecords []pbClient.DockerRecord) error {
	if a.pocketBase == nil {
		return fmt.Errorf("no PB client available")
	}
	
	if len(dockerRecords) == 0 {
		return nil
	}
	
	for _, docker := range dockerRecords {
		// Try to find existing Docker record
		existingDocker, err := a.pocketBase.GetDockerByID(docker.DockerID)
		if err != nil {
			// Docker record doesn't exist, create new one
			if err := a.pocketBase.SaveDockerRecord(docker); err != nil {
				log.Printf("Failed to save docker record %s: %v", docker.DockerID, err)
				return fmt.Errorf("failed to save docker record %s: %v", docker.DockerID, err)
			}
		} else {
			// Update existing Docker record
			if err := a.pocketBase.UpdateDockerRecord(existingDocker.ID, docker); err != nil {
				log.Printf("Failed to update docker record %s: %v", docker.DockerID, err)
				return fmt.Errorf("failed to update docker record %s: %v", docker.DockerID, err)
			}
		}
	}
	
	return nil
}

func (a *Agent) sendDockerMetrics(dockerMetrics []pbClient.DockerMetricsRecord) error {
	if a.pocketBase == nil {
		return fmt.Errorf("no PocketBase client available")
	}
	
	if len(dockerMetrics) == 0 {
		return nil
	}
	
	for _, metric := range dockerMetrics {
		if err := a.pocketBase.SaveDockerMetricsRecord(metric); err != nil {
			log.Printf("Failed to save docker metrics for %s: %v", metric.DockerID, err)
			return fmt.Errorf("failed to save docker metrics for %s: %v", metric.DockerID, err)
		}
	}
	
	return nil
}