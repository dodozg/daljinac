package server

import (
	"cmp"
	"slices"
	"sync"
	"time"

	"github.com/dodozg/daljinac/internal/metrics"
)

// Device is the latest report from one machine.
type Device struct {
	Client     string
	Report     metrics.Report
	ReceivedAt time.Time
}

// Store keeps the latest report per device in memory. Agents report every
// minute, so after a restart the picture is complete again within a minute.
type Store struct {
	mu      sync.RWMutex
	devices map[string]Device
}

func NewStore() *Store { return &Store{devices: map[string]Device{}} }

// Put records a report. Devices are keyed by client and device ID, so two
// customers can never overwrite each other's machines.
func (s *Store) Put(client string, r metrics.Report, now time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.devices[client+"\x00"+r.DeviceID] = Device{Client: client, Report: r, ReceivedAt: now}
}

// List returns all devices sorted by client, then hostname.
func (s *Store) List() []Device {
	s.mu.RLock()
	out := make([]Device, 0, len(s.devices))
	for _, d := range s.devices {
		out = append(out, d)
	}
	s.mu.RUnlock()
	slices.SortFunc(out, func(a, b Device) int {
		return cmp.Or(
			cmp.Compare(a.Client, b.Client),
			cmp.Compare(a.Report.Hostname, b.Report.Hostname),
			cmp.Compare(a.Report.DeviceID, b.Report.DeviceID),
		)
	})
	return out
}
