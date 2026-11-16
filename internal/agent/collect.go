// Package agent collects a machine's usage and sends it to the server.
package agent

import (
	"context"
	"fmt"
	"runtime"
	"strings"
	"time"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/host"
	"github.com/shirou/gopsutil/v4/mem"

	"github.com/dodozg/daljinac/internal/metrics"
)

// Collect takes one usage snapshot. It blocks for about cpuSample while
// measuring CPU load.
func Collect(ctx context.Context, cpuSample time.Duration, agentVersion string) (metrics.Report, error) {
	info, err := host.InfoWithContext(ctx)
	if err != nil {
		return metrics.Report{}, fmt.Errorf("host info: %w", err)
	}
	vm, err := mem.VirtualMemoryWithContext(ctx)
	if err != nil {
		return metrics.Report{}, fmt.Errorf("memory: %w", err)
	}
	cpus, err := cpu.PercentWithContext(ctx, cpuSample, false)
	if err != nil || len(cpus) == 0 {
		return metrics.Report{}, fmt.Errorf("cpu: %w", err)
	}
	parts, err := disk.PartitionsWithContext(ctx, false)
	if err != nil {
		return metrics.Report{}, fmt.Errorf("partitions: %w", err)
	}

	var disks []metrics.Disk
	for _, p := range filterPartitions(runtime.GOOS, parts) {
		u, err := disk.UsageWithContext(ctx, p.Mountpoint)
		if err != nil || u.Total == 0 {
			continue // unreadable or empty volume, e.g. a card reader without a card
		}
		disks = append(disks, metrics.Disk{Mount: p.Mountpoint, TotalBytes: u.Total, UsedBytes: u.Used})
		if len(disks) == metrics.MaxDisks {
			break
		}
	}

	deviceID := info.HostID
	if deviceID == "" {
		deviceID = info.Hostname // last resort; HostID is missing only on unusual systems
	}
	used := vm.Total - vm.Available // what is not available to new programs
	if vm.Available > vm.Total {
		used = vm.Used
	}

	return metrics.Report{
		DeviceID:      deviceID,
		Hostname:      info.Hostname,
		OS:            runtime.GOOS,
		Platform:      strings.TrimSpace(info.Platform + " " + info.PlatformVersion),
		CPUPercent:    min(max(cpus[0], 0), 100),
		MemTotalBytes: vm.Total,
		MemUsedBytes:  used,
		Disks:         disks,
		UptimeSeconds: info.Uptime,
		AgentVersion:  agentVersion,
	}, nil
}

// Filesystems that never hold user data worth watching.
var pseudoFS = map[string]bool{
	"tmpfs": true, "devtmpfs": true, "squashfs": true, "overlay": true, "proc": true,
	"sysfs": true, "cgroup": true, "cgroup2": true, "devfs": true, "autofs": true,
	"nsfs": true, "tracefs": true, "debugfs": true, "efivarfs": true, "ramfs": true,
	"fuse.snapfuse": true, "nullfs": true,
}

// filterPartitions keeps the volumes a technician cares about: real disks,
// each counted once. On macOS the APFS system volumes share one container,
// so only "/" and the data volume are kept.
func filterPartitions(goos string, parts []disk.PartitionStat) []disk.PartitionStat {
	seen := map[string]bool{}
	var out []disk.PartitionStat
	for _, p := range parts {
		if pseudoFS[strings.ToLower(p.Fstype)] || p.Mountpoint == "" {
			continue
		}
		switch goos {
		case "darwin":
			if p.Mountpoint != "/" && strings.HasPrefix(p.Mountpoint, "/System/Volumes/") &&
				p.Mountpoint != "/System/Volumes/Data" {
				continue
			}
		case "linux":
			if strings.HasPrefix(p.Mountpoint, "/snap/") || strings.HasPrefix(p.Mountpoint, "/boot/efi") {
				continue
			}
		}
		key := p.Device
		if key == "" || goos == "windows" {
			key = p.Mountpoint
		}
		if seen[key] {
			continue // bind mounts and duplicate entries of the same device
		}
		seen[key] = true
		out = append(out, p)
	}
	return out
}
