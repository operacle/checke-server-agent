
package agent

import (
	"bufio"
	"os"
	"runtime"
	"strconv"
	"strings"
)

// getMemoryUsage returns memory usage in bytes and percentage
func (sc *SystemCollector) getMemoryUsage() (used int64, total int64, percentage float64) {
	memInfo, err := sc.getMemInfo()
	if err != nil {
		// Fallback to Go runtime memory stats
		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		return int64(m.Alloc), int64(m.Sys), float64(m.Alloc)/float64(m.Sys)*100
	}

	used = memInfo["MemTotal"] - memInfo["MemAvailable"]
	total = memInfo["MemTotal"]
	
	if total > 0 {
		percentage = float64(used) / float64(total) * 100.0
	}

	return used, total, percentage
}

// getMemInfo reads memory information from /proc/meminfo
func (sc *SystemCollector) getMemInfo() (map[string]int64, error) {
	file, err := os.Open("/proc/meminfo")
	if err != nil {
		return nil, err
	}
	defer file.Close()

	memInfo := make(map[string]int64)
	scanner := bufio.NewScanner(file)
	
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) >= 2 {
			key := strings.TrimSuffix(fields[0], ":")
			value, err := strconv.ParseInt(fields[1], 10, 64)
			if err == nil {
				// Convert from KB to bytes
				memInfo[key] = value * 1024
			}
		}
	}

	return memInfo, scanner.Err()
}