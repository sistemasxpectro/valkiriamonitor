package metrics

import (
	"fmt"
	"os"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"
)

type SystemStats struct {
	CPUUsage    float64
	RAMUsed     uint64
	RAMTotal    uint64
	RAMUsage    float64
	DiskUsed    uint64
	DiskTotal   uint64
	DiskUsage   float64
	UptimeHours float64
}

func GetStats() (*SystemStats, error) {
	stats := &SystemStats{}

	// CPU
	cpuPercent, err := cpu.Percent(0, false)
	if err != nil {
		return nil, fmt.Errorf("error obteniendo uso de CPU: %w", err)
	}
	if len(cpuPercent) > 0 {
		stats.CPUUsage = cpuPercent[0]
	}

	// RAM
	vMem, err := mem.VirtualMemory()
	if err != nil {
		return nil, fmt.Errorf("error obteniendo memoria: %w", err)
	}
	stats.RAMUsed = vMem.Used
	stats.RAMTotal = vMem.Total
	stats.RAMUsage = vMem.UsedPercent

	rootPath := os.Getenv("HOST_ROOTFS")
	if rootPath == "" {
		rootPath = "/"
	}

	// Disco (partición raíz)
	dUsage, err := disk.Usage(rootPath)
	if err != nil {
		// Retornamos error con wrap
		return nil, fmt.Errorf("error obteniendo uso de disco en %s: %w", rootPath, err)
	}
	stats.DiskUsed = dUsage.Used
	stats.DiskTotal = dUsage.Total
	stats.DiskUsage = dUsage.UsedPercent

	// Uptime
	hInfo, err := host.Info()
	if err != nil {
		return nil, fmt.Errorf("error obteniendo información del host: %w", err)
	}
	stats.UptimeHours = float64(hInfo.Uptime) / 3600.0

	return stats, nil
}
