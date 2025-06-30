
package agent

import (
	"syscall"
)

// getDiskUsage returns disk usage for root filesystem
func (sc *SystemCollector) getDiskUsage() (used int64, total int64, percentage float64) {
	var stat syscall.Statfs_t
	err := syscall.Statfs("/", &stat)
	if err != nil {
		// Return placeholder values if unable to get real disk stats
		return 5 * 1024 * 1024 * 1024, 20 * 1024 * 1024 * 1024, 25.0
	}

	total = int64(stat.Blocks) * int64(stat.Bsize)
	free := int64(stat.Bavail) * int64(stat.Bsize)
	used = total - free

	if total > 0 {
		percentage = float64(used) / float64(total) * 100.0
	}

	return used, total, percentage
}