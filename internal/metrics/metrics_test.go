package metrics

import (
	"strings"
	"testing"
)

func validReport() Report {
	return Report{
		DeviceID:      "abc-123",
		Hostname:      "pc-01",
		OS:            "windows",
		Platform:      "Windows 11 Pro",
		CPUPercent:    12.5,
		MemTotalBytes: 16 << 30,
		MemUsedBytes:  8 << 30,
		Disks: []Disk{
			{Mount: "C:", TotalBytes: 500 << 30, UsedBytes: 250 << 30},
			{Mount: "D:", TotalBytes: 1000 << 30, UsedBytes: 900 << 30},
		},
	}
}

func TestValidate(t *testing.T) {
	if err := validReport().Validate(); err != nil {
		t.Fatalf("valid report rejected: %v", err)
	}

	cases := map[string]func(*Report){
		"missing device id":   func(r *Report) { r.DeviceID = "" },
		"long device id":      func(r *Report) { r.DeviceID = strings.Repeat("x", MaxDeviceIDLen+1) },
		"missing hostname":    func(r *Report) { r.Hostname = "" },
		"cpu over 100":        func(r *Report) { r.CPUPercent = 101 },
		"cpu negative":        func(r *Report) { r.CPUPercent = -1 },
		"mem used over total": func(r *Report) { r.MemUsedBytes = r.MemTotalBytes + 1 },
		"disk used over total": func(r *Report) {
			r.Disks[0].UsedBytes = r.Disks[0].TotalBytes + 1
		},
		"empty mount":    func(r *Report) { r.Disks[0].Mount = "" },
		"too many disks": func(r *Report) { r.Disks = make([]Disk, MaxDisks+1) },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			r := validReport()
			mutate(&r)
			if err := r.Validate(); err == nil {
				t.Fatal("expected an error")
			}
		})
	}
}

func TestPercentagesHandleZeroTotals(t *testing.T) {
	if got := (Disk{}).UsedPercent(); got != 0 {
		t.Fatalf("empty disk: got %v", got)
	}
	if got := (Report{}).MemPercent(); got != 0 {
		t.Fatalf("no memory info: got %v", got)
	}
}

func TestFullestDisk(t *testing.T) {
	d, ok := validReport().FullestDisk()
	if !ok || d.Mount != "D:" {
		t.Fatalf("got %+v, %v; want D:", d, ok)
	}
	if _, ok := (Report{}).FullestDisk(); ok {
		t.Fatal("report without disks should have no fullest disk")
	}
}

func TestClassify(t *testing.T) {
	th := Threshold{Warn: 85, Crit: 95}
	for pct, want := range map[float64]Level{0: OK, 84.9: OK, 85: Warning, 94.9: Warning, 95: Critical, 100: Critical} {
		if got := th.Classify(pct); got != want {
			t.Errorf("Classify(%v) = %v, want %v", pct, got, want)
		}
	}
}

func TestAssessTakesWorstLevel(t *testing.T) {
	a := Assess(validReport(), DefaultThresholds) // D: is 90 % full
	if a.CPU != OK || a.Mem != OK || a.Disk != Warning || a.Overall != Warning {
		t.Fatalf("got %+v", a)
	}

	r := validReport()
	r.CPUPercent = 99
	if got := Assess(r, DefaultThresholds).Overall; got != Critical {
		t.Fatalf("overall = %v, want critical", got)
	}
}

func TestThresholdsValidate(t *testing.T) {
	if err := DefaultThresholds.Validate(); err != nil {
		t.Fatalf("defaults rejected: %v", err)
	}
	bad := DefaultThresholds
	bad.Disk = Threshold{Warn: 95, Crit: 85}
	if err := bad.Validate(); err == nil {
		t.Fatal("warn >= crit should be rejected")
	}
}
