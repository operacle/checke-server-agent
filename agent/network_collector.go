
package agent

import (
	"bufio"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

// getNetworkStats returns real network statistics for the main physical interface
func (sc *SystemCollector) getNetworkStats() NetworkStats {
	currentStats, err := sc.getNetworkInfo()
	if err != nil {
		// Return placeholder values if unable to get real network stats
		return NetworkStats{
			BytesReceived: 0,
			BytesSent:     0,
			PacketsReceived: 0,
			PacketsSent:   0,
		}
	}

	now := time.Now()
	
	// Calculate speed if we have previous data
	var rxSpeed, txSpeed uint64
	if !sc.lastNetworkTime.IsZero() {
		timeDiff := now.Sub(sc.lastNetworkTime).Seconds()
		if timeDiff > 0 {
			rxDiff := currentStats.BytesReceived - sc.lastNetworkStats.BytesReceived
			txDiff := currentStats.BytesSent - sc.lastNetworkStats.BytesSent
			
			rxSpeed = uint64(float64(rxDiff) / timeDiff)
			txSpeed = uint64(float64(txDiff) / timeDiff)
		}
	}

	// Update last stats
	sc.lastNetworkStats = NetworkStats{
		BytesReceived: currentStats.BytesReceived,
		BytesSent:     currentStats.BytesSent,
		PacketsReceived: currentStats.PacketsReceived,
		PacketsSent:   currentStats.PacketsSent,
	}
	sc.lastNetworkTime = now

	// Return current stats with calculated speeds
	return NetworkStats{
		BytesReceived:   currentStats.BytesReceived,
		BytesSent:       currentStats.BytesSent,
		PacketsReceived: rxSpeed, // Use calculated RX speed
		PacketsSent:     txSpeed, // Use calculated TX speed
	}
}

// getMainNetworkInterface identifies the main physical network interface
func (sc *SystemCollector) getMainNetworkInterface() string {
	// Get default route interface
	if iface := sc.getDefaultRouteInterface(); iface != "" {
		return iface
	}
	
	// Fallback: find first active physical interface
	interfaces, err := net.Interfaces()
	if err != nil {
		return ""
	}
	
	// Priority order for common physical interface names
	priorities := []string{"eth0", "ens", "enp", "eno", "em", "bond", "br"}
	
	for _, priority := range priorities {
		for _, iface := range interfaces {
			// Skip loopback, virtual, and down interfaces
			if iface.Flags&net.FlagLoopback != 0 || iface.Flags&net.FlagUp == 0 {
				continue
			}
			
			// Check if interface name starts with priority prefix
			if strings.HasPrefix(iface.Name, priority) {
				// Verify it has an IP address
				if sc.hasValidIPAddress(iface) {
					return iface.Name
				}
			}
		}
	}
	
	// Last resort: return first active non-loopback interface
	for _, iface := range interfaces {
		if iface.Flags&net.FlagLoopback == 0 && iface.Flags&net.FlagUp != 0 {
			if sc.hasValidIPAddress(iface) {
				return iface.Name
			}
		}
	}
	
	return ""
}

// getDefaultRouteInterface gets the interface used for the default route
func (sc *SystemCollector) getDefaultRouteInterface() string {
	file, err := os.Open("/proc/net/route")
	if err != nil {
		return ""
	}
	defer file.Close()
	
	scanner := bufio.NewScanner(file)
	scanner.Scan() // Skip header
	
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) >= 2 {
			// Check if this is the default route (destination 00000000)
			if fields[1] == "00000000" {
				return fields[0]
			}
		}
	}
	
	return ""
}

// hasValidIPAddress checks if interface has a valid IP address
func (sc *SystemCollector) hasValidIPAddress(iface net.Interface) bool {
	addrs, err := iface.Addrs()
	if err != nil {
		return false
	}
	
	for _, addr := range addrs {
		var ip net.IP
		switch v := addr.(type) {
		case *net.IPNet:
			ip = v.IP
		case *net.IPAddr:
			ip = v.IP
		}
		
		if ip != nil && !ip.IsLoopback() && ip.To4() != nil {
			return true
		}
	}
	
	return false
}

// getNetworkInfo reads network statistics from /proc/net/dev for the main interface only
func (sc *SystemCollector) getNetworkInfo() (NetworkStats, error) {
	file, err := os.Open("/proc/net/dev")
	if err != nil {
		return NetworkStats{}, err
	}
	defer file.Close()

	// Get the main network interface
	mainInterface := sc.getMainNetworkInterface()
	if mainInterface == "" {
		// Fallback to old behavior if we can't determine main interface
		return sc.getNetworkInfoAllInterfaces()
	}

	scanner := bufio.NewScanner(file)
	
	// Skip header lines
	scanner.Scan()
	scanner.Scan()
	
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		
		if len(fields) < 10 {
			continue
		}
		
		// Get interface name
		interfaceName := strings.TrimSuffix(fields[0], ":")
		
		// Only process the main interface
		if interfaceName != mainInterface {
			continue
		}
		
		// Parse RX bytes (field 1), RX packets (field 2), TX bytes (field 9), TX packets (field 10)
		rxBytes, err1 := strconv.ParseUint(fields[1], 10, 64)
		rxPackets, err2 := strconv.ParseUint(fields[2], 10, 64)
		txBytes, err3 := strconv.ParseUint(fields[9], 10, 64)
		txPackets, err4 := strconv.ParseUint(fields[10], 10, 64)
		
		if err1 == nil && err2 == nil && err3 == nil && err4 == nil {
			return NetworkStats{
				BytesReceived:   rxBytes,
				BytesSent:       txBytes,
				PacketsReceived: rxPackets,
				PacketsSent:     txPackets,
			}, nil
		}
	}

	return NetworkStats{}, scanner.Err()
}

// getNetworkInfoAllInterfaces is the fallback method that aggregates all interfaces
func (sc *SystemCollector) getNetworkInfoAllInterfaces() (NetworkStats, error) {
	file, err := os.Open("/proc/net/dev")
	if err != nil {
		return NetworkStats{}, err
	}
	defer file.Close()

	var totalRx, totalTx, totalRxPackets, totalTxPackets uint64
	scanner := bufio.NewScanner(file)
	
	// Skip header lines
	scanner.Scan()
	scanner.Scan()
	
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		
		if len(fields) < 10 {
			continue
		}
		
		// Skip loopback interface
		interfaceName := strings.TrimSuffix(fields[0], ":")
		if interfaceName == "lo" {
			continue
		}
		
		// Parse RX bytes (field 1), RX packets (field 2), TX bytes (field 9), TX packets (field 10)
		rxBytes, err1 := strconv.ParseUint(fields[1], 10, 64)
		rxPackets, err2 := strconv.ParseUint(fields[2], 10, 64)
		txBytes, err3 := strconv.ParseUint(fields[9], 10, 64)
		txPackets, err4 := strconv.ParseUint(fields[10], 10, 64)
		
		if err1 == nil && err2 == nil && err3 == nil && err4 == nil {
			totalRx += rxBytes
			totalRxPackets += rxPackets
			totalTx += txBytes
			totalTxPackets += txPackets
		}
	}

	return NetworkStats{
		BytesReceived:   totalRx,
		BytesSent:       totalTx,
		PacketsReceived: totalRxPackets,
		PacketsSent:     totalTxPackets,
	}, scanner.Err()
}