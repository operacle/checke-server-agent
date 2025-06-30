
package agent

import (
	"time"
)

// SystemCollector provides real system metrics
type SystemCollector struct {
	lastCPUStats     CPUStats
	lastNetworkStats NetworkStats
	lastNetworkTime  time.Time
	lastCPUTime      time.Time
	initialized      bool
}

type CPUStats struct {
	User   uint64
	Nice   uint64
	System uint64
	Idle   uint64
	IOWait uint64
	IRQ    uint64
	SoftIRQ uint64
	Steal  uint64
	Guest  uint64
	Total  uint64 // Add total for easier calculation
}

type SystemInfo struct {
	Hostname        string
	OSName          string
	OSVersion       string
	KernelVersion   string
	Architecture    string
	CPUModel        string
	CPUCores        int
	TotalRAM        int64
	GoVersion       string
	Platform        string
	IPAddress       string
	OSType          string
}

func NewSystemCollector() *SystemCollector {
	return &SystemCollector{}
}

// GetSystemInfo returns comprehensive system information
func (sc *SystemCollector) GetSystemInfo() SystemInfo {
	return sc.getSystemInfo()
}

// GetRealHostname returns the actual system hostname
func (sc *SystemCollector) GetRealHostname() string {
	return sc.getRealHostname()
}

// GetCPUUsage returns real CPU usage percentage with proper timing and multiple samples
func (sc *SystemCollector) GetCPUUsage() float64 {
	return sc.getCPUUsage()
}

// GetMemoryUsage returns memory usage in bytes and percentage
func (sc *SystemCollector) GetMemoryUsage() (used int64, total int64, percentage float64) {
	return sc.getMemoryUsage()
}

// GetDiskUsage returns disk usage for root filesystem
func (sc *SystemCollector) GetDiskUsage() (used int64, total int64, percentage float64) {
	return sc.getDiskUsage()
}

// GetNetworkStats returns real network statistics
func (sc *SystemCollector) GetNetworkStats() NetworkStats {
	return sc.getNetworkStats()
}

// GetSystemUptime returns system uptime in seconds
func (sc *SystemCollector) GetSystemUptime() int64 {
	return sc.getSystemUptime()
}