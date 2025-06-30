
package agent

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// getCPUUsage returns real CPU usage percentage with proper timing and multiple samples
func (sc *SystemCollector) getCPUUsage() float64 {
	// Take multiple samples for more accurate measurement
	const sampleCount = 3
	const sampleInterval = 100 * time.Millisecond
	
	var totalUsage float64
	validSamples := 0
	
	for i := 0; i < sampleCount; i++ {
		usage := sc.getSingleCPUUsage()
		if usage >= 0 && usage <= 100 {
			totalUsage += usage
			validSamples++
		}
		
		if i < sampleCount-1 {
			time.Sleep(sampleInterval)
		}
	}
	
	if validSamples == 0 {
		return 0.0
	}
	
	avgUsage := totalUsage / float64(validSamples)
	
	// Round to 2 decimal places
	return float64(int(avgUsage*100)) / 100
}

// getSingleCPUUsage gets a single CPU usage sample
func (sc *SystemCollector) getSingleCPUUsage() float64 {
	currentStats, err := sc.getCPUStats()
	if err != nil {
		return 0.0
	}

	now := time.Now()

	// If this is the first call, initialize and wait for next sample
	if !sc.initialized || sc.lastCPUStats.Total == 0 {
		sc.lastCPUStats = currentStats
		sc.lastCPUTime = now
		sc.initialized = true
		
		// Wait a bit and take another sample
		time.Sleep(200 * time.Millisecond)
		
		newStats, err := sc.getCPUStats()
		if err != nil {
			return 0.0
		}
		
		return sc.calculateCPUPercentage(currentStats, newStats)
	}

	// Calculate time difference
	timeDiff := now.Sub(sc.lastCPUTime)
	if timeDiff < 50*time.Millisecond {
		// Too little time has passed, return previous calculation
		return 0.0
	}

	cpuUsage := sc.calculateCPUPercentage(sc.lastCPUStats, currentStats)
	
	// Update last stats and time
	sc.lastCPUStats = currentStats
	sc.lastCPUTime = now

	return cpuUsage
}

// calculateCPUPercentage calculates CPU usage percentage between two CPU stat snapshots
func (sc *SystemCollector) calculateCPUPercentage(prev, curr CPUStats) float64 {
	// Calculate differences
	prevIdle := prev.Idle + prev.IOWait
	currIdle := curr.Idle + curr.IOWait
	
	prevNonIdle := prev.User + prev.Nice + prev.System + prev.IRQ + prev.SoftIRQ + prev.Steal
	currNonIdle := curr.User + curr.Nice + curr.System + curr.IRQ + curr.SoftIRQ + curr.Steal
	
	prevTotal := prevIdle + prevNonIdle
	currTotal := currIdle + currNonIdle
	
	totalDiff := currTotal - prevTotal
	idleDiff := currIdle - prevIdle

	if totalDiff == 0 {
		return 0.0
	}

	cpuUsage := (float64(totalDiff - idleDiff) / float64(totalDiff)) * 100.0
	
	// Ensure reasonable bounds
	if cpuUsage < 0 {
		return 0.0
	}
	if cpuUsage > 100 {
		return 100.0
	}

	return cpuUsage
}

// getCPUStats reads CPU stats from /proc/stat with better error handling
func (sc *SystemCollector) getCPUStats() (CPUStats, error) {
	file, err := os.Open("/proc/stat")
	if err != nil {
		return CPUStats{}, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "cpu ") {
			fields := strings.Fields(line)
			if len(fields) < 8 {
				continue
			}

			stats := CPUStats{}
			stats.User, _ = strconv.ParseUint(fields[1], 10, 64)
			stats.Nice, _ = strconv.ParseUint(fields[2], 10, 64)
			stats.System, _ = strconv.ParseUint(fields[3], 10, 64)
			stats.Idle, _ = strconv.ParseUint(fields[4], 10, 64)
			stats.IOWait, _ = strconv.ParseUint(fields[5], 10, 64)
			stats.IRQ, _ = strconv.ParseUint(fields[6], 10, 64)
			stats.SoftIRQ, _ = strconv.ParseUint(fields[7], 10, 64)
			if len(fields) > 8 {
				stats.Steal, _ = strconv.ParseUint(fields[8], 10, 64)
			}
			if len(fields) > 9 {
				stats.Guest, _ = strconv.ParseUint(fields[9], 10, 64)
			}
			
			// Calculate total
			stats.Total = stats.User + stats.Nice + stats.System + stats.Idle + 
						  stats.IOWait + stats.IRQ + stats.SoftIRQ + stats.Steal + stats.Guest

			return stats, nil
		}
	}

	return CPUStats{}, fmt.Errorf("cpu stats not found")
}

// getTotalCPUTime calculates total CPU time
func (sc *SystemCollector) getTotalCPUTime(stats CPUStats) uint64 {
	return stats.Total
}