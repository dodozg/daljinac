package agent

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
	"time"

	"github.com/shirou/gopsutil/v4/disk"

	"github.com/dodozg/daljinac/internal/metrics"
)

// TestCollectOnThisMachine runs the real collector. CI runs it on Windows,
// macOS and Linux, which is the point: it proves gopsutil works on each.
func TestCollectOnThisMachine(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	r, err := Collect(ctx, 200*time.Millisecond, "test")
	if err != nil {
		t.Fatal(err)
	}
	if err := r.Validate(); err != nil {
		t.Fatalf("collected report is invalid: %v\n%+v", err, r)
	}
	if r.MemTotalBytes == 0 || len(r.Disks) == 0 {
		t.Fatalf("expected memory and at least one disk: %+v", r)
	}
	t.Logf("%s (%s): cpu %.0f %%, mem %.0f %%, disks %v", r.Hostname, r.Platform, r.CPUPercent, r.MemPercent(), r.Disks)
}

func mounts(ps []disk.PartitionStat) []string {
	var out []string
	for _, p := range ps {
		out = append(out, p.Mountpoint)
	}
	return out
}

func TestFilterPartitions(t *testing.T) {
	cases := []struct {
		goos  string
		parts []disk.PartitionStat
		want  []string
	}{
		{"linux", []disk.PartitionStat{
			{Device: "/dev/sda1", Mountpoint: "/", Fstype: "ext4"},
			{Device: "/dev/sda1", Mountpoint: "/var/lib/docker", Fstype: "ext4"}, // bind mount of the same device
			{Device: "tmpfs", Mountpoint: "/run", Fstype: "tmpfs"},
			{Device: "/dev/loop3", Mountpoint: "/snap/core/1", Fstype: "squashfs"},
			{Device: "/dev/sda2", Mountpoint: "/boot/efi", Fstype: "vfat"},
			{Device: "/dev/sdb1", Mountpoint: "/data", Fstype: "xfs"},
		}, []string{"/", "/data"}},
		{"darwin", []disk.PartitionStat{
			{Device: "/dev/disk3s1s1", Mountpoint: "/", Fstype: "apfs"},
			{Device: "/dev/disk3s6", Mountpoint: "/System/Volumes/VM", Fstype: "apfs"},
			{Device: "/dev/disk3s2", Mountpoint: "/System/Volumes/Preboot", Fstype: "apfs"},
			{Device: "/dev/disk3s5", Mountpoint: "/System/Volumes/Data", Fstype: "apfs"},
			{Device: "devfs", Mountpoint: "/dev", Fstype: "devfs"},
		}, []string{"/", "/System/Volumes/Data"}},
		{"windows", []disk.PartitionStat{
			{Device: "C:", Mountpoint: "C:", Fstype: "NTFS"},
			{Device: "D:", Mountpoint: "D:", Fstype: "NTFS"},
		}, []string{"C:", "D:"}},
	}
	for _, c := range cases {
		if got := mounts(filterPartitions(c.goos, c.parts)); !slices.Equal(got, c.want) {
			t.Errorf("%s: got %v, want %v", c.goos, got, c.want)
		}
	}
}

func TestReportURL(t *testing.T) {
	ok := map[string]string{
		"https://remote.example.hr":        "https://remote.example.hr/api/v1/report",
		"https://remote.example.hr/agent/": "https://remote.example.hr/agent/api/v1/report",
		"http://127.0.0.1:8081":            "http://127.0.0.1:8081/api/v1/report",
		"http://localhost:8081":            "http://localhost:8081/api/v1/report",
	}
	for in, want := range ok {
		if got, err := ReportURL(in); err != nil || got != want {
			t.Errorf("ReportURL(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
	for _, bad := range []string{"http://remote.example.hr", "ftp://x", "remote.example.hr", "https://"} {
		if _, err := ReportURL(bad); err == nil {
			t.Errorf("ReportURL(%q) should fail", bad)
		}
	}
}

func TestSend(t *testing.T) {
	var got metrics.Report
	var auth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth = r.Header.Get("Authorization")
		_ = json.NewDecoder(r.Body).Decode(&got)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	url, err := ReportURL(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	want := metrics.Report{DeviceID: "d1", Hostname: "pc"}
	if err := Send(context.Background(), srv.Client(), url, "tok", want); err != nil {
		t.Fatal(err)
	}
	if auth != "Bearer tok" || got.DeviceID != "d1" {
		t.Fatalf("auth %q, report %+v", auth, got)
	}

	fail := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	}))
	defer fail.Close()
	url, _ = ReportURL(fail.URL)
	if err := Send(context.Background(), fail.Client(), url, "tok", want); err == nil {
		t.Fatal("expected an error on 401")
	}
}
