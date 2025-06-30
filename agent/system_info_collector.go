
package agent

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// getSystemInfo returns comprehensive system information
func (sc *SystemCollector) getSystemInfo() SystemInfo {
	hostname, _ := os.Hostname()
	
	info := SystemInfo{
		Hostname:     hostname,
		Architecture: runtime.GOARCH,
		CPUCores:     runtime.NumCPU(),
		GoVersion:    runtime.Version(),
		Platform:     runtime.GOOS,
		IPAddress:    sc.getRealIPAddress(),
		OSType:       sc.getOSType(),
	}
	
	// Get OS information from /etc/os-release
	if osInfo := sc.getOSInfo(); osInfo != nil {
		info.OSName = osInfo["NAME"]
		info.OSVersion = osInfo["VERSION"]
	}
	
	// Get kernel version
	if kernelVersion := sc.getKernelVersion(); kernelVersion != "" {
		info.KernelVersion = kernelVersion
	}
	
	// Get CPU model information
	if cpuModel := sc.getCPUModel(); cpuModel != "" {
		info.CPUModel = cpuModel
	}
	
	// Get total RAM
	if memInfo, err := sc.getMemInfo(); err == nil {
		info.TotalRAM = memInfo["MemTotal"]
	}
	
	return info
}

// getRealHostname returns the actual system hostname
func (sc *SystemCollector) getRealHostname() string {
	hostname, err := os.Hostname()
	if err != nil {
		return "unknown"
	}
	return hostname
}

// getRealIPAddress returns the actual system IP address
func (sc *SystemCollector) getRealIPAddress() string {
	// Try to get the IP address from network interfaces
	interfaces, err := net.Interfaces()
	if err != nil {
		return "unknown"
	}

	for _, iface := range interfaces {
		// Skip loopback and down interfaces
		if iface.Flags&net.FlagLoopback != 0 || iface.Flags&net.FlagUp == 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}

			// Return first non-loopback IPv4 address
			if ip != nil && !ip.IsLoopback() && ip.To4() != nil {
				return ip.String()
			}
		}
	}

	return "unknown"
}

// getOSType returns the operating system type
func (sc *SystemCollector) getOSType() string {
	switch runtime.GOOS {
	case "linux":
		return "Linux"
	case "darwin":
		return "macOS"
	case "windows":
		return "Windows"
	case "freebsd":
		return "FreeBSD"
	case "openbsd":
		return "OpenBSD"
	case "netbsd":
		return "NetBSD"
	default:
		return runtime.GOOS
	}
}

// getOSInfo reads OS information from /etc/os-release
func (sc *SystemCollector) getOSInfo() map[string]string {
	file, err := os.Open("/etc/os-release")
	if err != nil {
		// Try alternative location
		file, err = os.Open("/usr/lib/os-release")
		if err != nil {
			return nil
		}
	}
	defer file.Close()

	osInfo := make(map[string]string)
	scanner := bufio.NewScanner(file)
	
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "=") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				value := strings.Trim(strings.TrimSpace(parts[1]), `"`)
				osInfo[key] = value
			}
		}
	}

	return osInfo
}

// getKernelVersion reads kernel version from /proc/version
func (sc *SystemCollector) getKernelVersion() string {
	file, err := os.Open("/proc/version")
	if err != nil {
		return ""
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	if scanner.Scan() {
		version := scanner.Text()
		// Extract version number from "Linux version x.x.x"
		if strings.HasPrefix(version, "Linux version ") {
			parts := strings.Fields(version)
			if len(parts) >= 3 {
				return parts[2]
			}
		}
	}

	return ""
}

// getCPUModel reads CPU model from /proc/cpuinfo
func (sc *SystemCollector) getCPUModel() string {
	file, err := os.Open("/proc/cpuinfo")
	if err != nil {
		return ""
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "model name") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				return strings.TrimSpace(parts[1])
			}
		}
	}

	return ""
}

// getSystemUptime returns system uptime in seconds
func (sc *SystemCollector) getSystemUptime() int64 {
	uptime, err := sc.getUptime()
	if err != nil {
		// Fallback to a placeholder
		return int64(time.Since(time.Now().Add(-24 * time.Hour)).Seconds())
	}
	return uptime
}

// getUptime reads system uptime from /proc/uptime
func (sc *SystemCollector) getUptime() (int64, error) {
	file, err := os.Open("/proc/uptime")
	if err != nil {
		return 0, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	if scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) >= 1 {
			uptime, err := strconv.ParseFloat(fields[0], 64)
			if err == nil {
				return int64(uptime), nil
			}
		}
	}

	return 0, fmt.Errorf("failed to parse uptime")
}