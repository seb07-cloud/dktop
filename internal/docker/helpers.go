package docker

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
)

func formatPorts(ports []types.Port) string {
	if len(ports) == 0 {
		return "-"
	}

	var portStrs []string
	for _, p := range ports {
		if p.PublicPort > 0 {
			portStrs = append(portStrs, fmt.Sprintf("%d->%d/%s", p.PublicPort, p.PrivatePort, p.Type))
		} else {
			portStrs = append(portStrs, fmt.Sprintf("%d/%s", p.PrivatePort, p.Type))
		}
	}

	result := strings.Join(portStrs, ", ")
	if len(result) > 30 {
		return result[:27] + "..."
	}
	return result
}

func calculateCPUPercent(stats *container.StatsResponse) float64 {
	cpuDelta := float64(stats.CPUStats.CPUUsage.TotalUsage - stats.PreCPUStats.CPUUsage.TotalUsage)
	systemDelta := float64(stats.CPUStats.SystemUsage - stats.PreCPUStats.SystemUsage)

	if systemDelta > 0 && cpuDelta > 0 {
		cpuCount := float64(stats.CPUStats.OnlineCPUs)
		if cpuCount == 0 {
			cpuCount = float64(len(stats.CPUStats.CPUUsage.PercpuUsage))
		}
		return (cpuDelta / systemDelta) * cpuCount * 100.0
	}
	return 0.0
}

func decodeStats(reader io.Reader, stats *container.StatsResponse) error {
	return json.NewDecoder(reader).Decode(stats)
}

func formatInt(n int) string {
	return fmt.Sprintf("%d", n)
}

func stripLogHeaders(logs string) string {
	lines := strings.Split(logs, "\n")
	var result []string

	for _, line := range lines {
		if len(line) > 8 {
			// Docker multiplexed log format has 8-byte header
			// First byte indicates stream (1=stdout, 2=stderr)
			// Bytes 5-8 indicate length
			result = append(result, line[8:])
		} else if len(line) > 0 {
			result = append(result, line)
		}
	}

	return strings.Join(result, "\n")
}

// FormatBytes formats bytes into human readable format
func FormatBytes(bytes uint64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)

	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.1fGB", float64(bytes)/GB)
	case bytes >= MB:
		return fmt.Sprintf("%.1fMB", float64(bytes)/MB)
	case bytes >= KB:
		return fmt.Sprintf("%.1fKB", float64(bytes)/KB)
	default:
		return fmt.Sprintf("%dB", bytes)
	}
}

// FormatBytesShort formats bytes into short human readable format
func FormatBytesShort(bytes uint64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)

	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.1fG", float64(bytes)/GB)
	case bytes >= MB:
		return fmt.Sprintf("%.0fM", float64(bytes)/MB)
	case bytes >= KB:
		return fmt.Sprintf("%.0fK", float64(bytes)/KB)
	default:
		return fmt.Sprintf("%dB", bytes)
	}
}
