// Package metrics defines the usage report an agent sends to the server and
// the rules that turn raw numbers into an OK / warning / critical level.
package metrics

import (
	"errors"
	"fmt"
)

// Limits that keep a single report small and bounded.
const (
	MaxDeviceIDLen = 128
	MaxHostnameLen = 255
	MaxTextLen     = 255
	MaxMountLen    = 512
	MaxDisks       = 64
)

// Disk is the usage of one mounted volume.
type Disk struct {
	Mount      string `json:"mount"`
	TotalBytes uint64 `json:"total_bytes"`
	UsedBytes  uint64 `json:"used_bytes"`
}

// UsedPercent returns how full the disk is, 0–100.
func (d Disk) UsedPercent() float64 {
	if d.TotalBytes == 0 {
		return 0
	}
	return float64(d.UsedBytes) / float64(d.TotalBytes) * 100
}

// Report is one snapshot of a machine's usage, sent by the agent.
type Report struct {
	DeviceID      string  `json:"device_id"`
	Hostname      string  `json:"hostname"`
	OS            string  `json:"os"`       // windows, darwin, linux
	Platform      string  `json:"platform"` // human-readable OS name and version
	CPUPercent    float64 `json:"cpu_percent"`
	MemTotalBytes uint64  `json:"mem_total_bytes"`
	MemUsedBytes  uint64  `json:"mem_used_bytes"`
	Disks         []Disk  `json:"disks"`
	UptimeSeconds uint64  `json:"uptime_seconds"`
	AgentVersion  string  `json:"agent_version"`
}

// MemPercent returns memory usage, 0–100.
func (r Report) MemPercent() float64 {
	if r.MemTotalBytes == 0 {
		return 0
	}
	return float64(r.MemUsedBytes) / float64(r.MemTotalBytes) * 100
}

// FullestDisk returns the disk with the highest usage percentage.
func (r Report) FullestDisk() (Disk, bool) {
	var best Disk
	found := false
	for _, d := range r.Disks {
		if !found || d.UsedPercent() > best.UsedPercent() {
			best, found = d, true
		}
	}
	return best, found
}

// Validate rejects reports that are malformed or implausible.
func (r Report) Validate() error {
	switch {
	case r.DeviceID == "":
		return errors.New("device_id is required")
	case len(r.DeviceID) > MaxDeviceIDLen:
		return fmt.Errorf("device_id longer than %d", MaxDeviceIDLen)
	case r.Hostname == "":
		return errors.New("hostname is required")
	case len(r.Hostname) > MaxHostnameLen:
		return fmt.Errorf("hostname longer than %d", MaxHostnameLen)
	case len(r.OS) > MaxTextLen || len(r.Platform) > MaxTextLen || len(r.AgentVersion) > MaxTextLen:
		return fmt.Errorf("os, platform or agent_version longer than %d", MaxTextLen)
	case r.CPUPercent < 0 || r.CPUPercent > 100:
		return fmt.Errorf("cpu_percent %v outside 0–100", r.CPUPercent)
	case r.MemUsedBytes > r.MemTotalBytes:
		return errors.New("mem_used_bytes greater than mem_total_bytes")
	case len(r.Disks) > MaxDisks:
		return fmt.Errorf("more than %d disks", MaxDisks)
	}
	for i, d := range r.Disks {
		if d.Mount == "" || len(d.Mount) > MaxMountLen {
			return fmt.Errorf("disk %d: mount empty or longer than %d", i, MaxMountLen)
		}
		if d.UsedBytes > d.TotalBytes {
			return fmt.Errorf("disk %d (%s): used greater than total", i, d.Mount)
		}
	}
	return nil
}

// Level is how worried a technician should be about a value.
type Level int

const (
	OK Level = iota
	Warning
	Critical
)

func (l Level) String() string {
	switch l {
	case Warning:
		return "warning"
	case Critical:
		return "critical"
	default:
		return "ok"
	}
}

// Threshold holds the percentages at which a value becomes a warning or critical.
type Threshold struct {
	Warn float64 `json:"warn"`
	Crit float64 `json:"crit"`
}

// Classify returns the level for a percentage.
func (t Threshold) Classify(pct float64) Level {
	switch {
	case pct >= t.Crit:
		return Critical
	case pct >= t.Warn:
		return Warning
	default:
		return OK
	}
}

// Thresholds groups the limits for CPU, memory and disk.
type Thresholds struct {
	CPU  Threshold `json:"cpu"`
	Mem  Threshold `json:"mem"`
	Disk Threshold `json:"disk"`
}

// DefaultThresholds are the limits used unless the server config overrides them.
var DefaultThresholds = Thresholds{
	CPU:  Threshold{Warn: 75, Crit: 90},
	Mem:  Threshold{Warn: 85, Crit: 95},
	Disk: Threshold{Warn: 85, Crit: 95},
}

// Validate checks that warn < crit and both lie in 0–100.
func (t Thresholds) Validate() error {
	for name, th := range map[string]Threshold{"cpu": t.CPU, "mem": t.Mem, "disk": t.Disk} {
		if th.Warn <= 0 || th.Crit > 100 || th.Warn >= th.Crit {
			return fmt.Errorf("threshold %s: need 0 < warn < crit <= 100, got warn=%v crit=%v", name, th.Warn, th.Crit)
		}
	}
	return nil
}

// Assessment is the level of each measured value and the worst of them.
type Assessment struct {
	CPU, Mem, Disk, Overall Level
}

// Assess classifies a report against the thresholds.
func Assess(r Report, t Thresholds) Assessment {
	a := Assessment{
		CPU: t.CPU.Classify(r.CPUPercent),
		Mem: t.Mem.Classify(r.MemPercent()),
	}
	if d, ok := r.FullestDisk(); ok {
		a.Disk = t.Disk.Classify(d.UsedPercent())
	}
	a.Overall = max(a.CPU, a.Mem, a.Disk)
	return a
}
