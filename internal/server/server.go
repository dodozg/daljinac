// Package server receives usage reports from agents and shows them to
// technicians on a console page.
package server

import (
	"crypto/subtle"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/dodozg/daljinac/internal/metrics"
)

// MaxReportBytes caps the size of one report request body.
const MaxReportBytes = 64 << 10

//go:embed dashboard.html
var dashboardHTML string

var dashboardTmpl = template.Must(template.New("dashboard").Funcs(template.FuncMap{
	"pct":      func(v float64) string { return fmt.Sprintf("%.0f %%", v) },
	"gb":       formatGB,
	"machines": machines,
}).Parse(dashboardHTML))

func formatGB(b uint64) string {
	gb := float64(b) / (1 << 30)
	if gb < 10 {
		return fmt.Sprintf("%.1f GB", gb)
	}
	return fmt.Sprintf("%.0f GB", gb)
}

// machines writes "N strojeva" with the right Croatian plural form.
func machines(n int) string {
	switch {
	case n%10 == 1 && n%100 != 11:
		return fmt.Sprintf("%d stroj", n)
	case n%10 >= 2 && n%10 <= 4 && (n%100 < 12 || n%100 > 14):
		return fmt.Sprintf("%d stroja", n)
	default:
		return fmt.Sprintf("%d strojeva", n)
	}
}

// Server holds the state shared by the agent and console handlers.
type Server struct {
	cfg   Config
	store *Store
	log   *slog.Logger
	now   func() time.Time
}

func New(cfg Config, log *slog.Logger) *Server {
	return &Server{cfg: cfg, store: NewStore(), log: log, now: time.Now}
}

// AgentHandler serves the endpoints agents talk to.
func (s *Server) AgentHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })
	mux.HandleFunc("POST /api/v1/report", s.handleReport)
	return mux
}

// ConsoleHandler serves the technician console.
func (s *Server) ConsoleHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", s.handleDashboard)
	mux.HandleFunc("GET /api/v1/devices", s.handleDevices)
	return mux
}

// clientForToken returns the client owning the bearer token. Every token is
// compared in constant time so response timing does not reveal valid prefixes.
func (s *Server) clientForToken(r *http.Request) (string, bool) {
	auth := r.Header.Get("Authorization")
	token, ok := strings.CutPrefix(auth, "Bearer ")
	if !ok || token == "" {
		return "", false
	}
	found := ""
	for _, c := range s.cfg.Clients {
		if subtle.ConstantTimeCompare([]byte(token), []byte(c.Token)) == 1 {
			found = c.Name
		}
	}
	return found, found != ""
}

func (s *Server) handleReport(w http.ResponseWriter, r *http.Request) {
	client, ok := s.clientForToken(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	var rep metrics.Report
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, MaxReportBytes))
	if err := dec.Decode(&rep); err != nil {
		var tooBig *http.MaxBytesError
		if errors.As(err, &tooBig) {
			http.Error(w, "report too large", http.StatusRequestEntityTooLarge)
			return
		}
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	if err := rep.Validate(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	s.store.Put(client, rep, s.now())
	w.WriteHeader(http.StatusNoContent)
}

// DeviceView is a device as the console shows it.
type DeviceView struct {
	Client        string             `json:"client"`
	DeviceID      string             `json:"device_id"`
	Hostname      string             `json:"hostname"`
	OS            string             `json:"os"`
	Platform      string             `json:"platform"`
	Online        bool               `json:"online"`
	LastSeen      time.Time          `json:"last_seen"`
	CPUPercent    float64            `json:"cpu_percent"`
	MemPercent    float64            `json:"mem_percent"`
	MemTotalBytes uint64             `json:"mem_total_bytes"`
	Disk          *DiskView          `json:"fullest_disk,omitempty"`
	Assessment    metrics.Assessment `json:"-"`
	Level         string             `json:"level"` // ok, warning, critical, offline
}

type DiskView struct {
	Mount       string  `json:"mount"`
	UsedPercent float64 `json:"used_percent"`
	TotalBytes  uint64  `json:"total_bytes"`
}

// ClientView groups the devices of one customer.
type ClientView struct {
	Name     string
	Devices  []DeviceView
	Problems int // devices that are offline or at warning/critical level
}

func (s *Server) views() []DeviceView {
	now := s.now()
	devices := s.store.List()
	out := make([]DeviceView, 0, len(devices))
	for _, d := range devices {
		r := d.Report
		v := DeviceView{
			Client:        d.Client,
			DeviceID:      r.DeviceID,
			Hostname:      r.Hostname,
			OS:            r.OS,
			Platform:      r.Platform,
			Online:        now.Sub(d.ReceivedAt) <= s.cfg.OfflineAfter.Duration,
			LastSeen:      d.ReceivedAt,
			CPUPercent:    r.CPUPercent,
			MemPercent:    r.MemPercent(),
			MemTotalBytes: r.MemTotalBytes,
			Assessment:    metrics.Assess(r, *s.cfg.Thresholds),
		}
		if disk, ok := r.FullestDisk(); ok {
			v.Disk = &DiskView{Mount: disk.Mount, UsedPercent: disk.UsedPercent(), TotalBytes: disk.TotalBytes}
		}
		v.Level = v.Assessment.Overall.String()
		if !v.Online {
			v.Level = "offline"
		}
		out = append(out, v)
	}
	return out
}

func groupByClient(devices []DeviceView) []ClientView {
	var out []ClientView
	for _, d := range devices { // already sorted by client
		if len(out) == 0 || out[len(out)-1].Name != d.Client {
			out = append(out, ClientView{Name: d.Client})
		}
		c := &out[len(out)-1]
		c.Devices = append(c.Devices, d)
		if d.Level != "ok" {
			c.Problems++
		}
	}
	return out
}

func (s *Server) handleDevices(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(s.views()); err != nil {
		s.log.Error("encode devices", "err", err)
	}
}

func (s *Server) handleDashboard(w http.ResponseWriter, _ *http.Request) {
	data := struct {
		Clients    []ClientView
		Generated  time.Time
		Thresholds metrics.Thresholds
	}{groupByClient(s.views()), s.now(), *s.cfg.Thresholds}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := dashboardTmpl.Execute(w, data); err != nil {
		s.log.Error("render dashboard", "err", err)
	}
}
