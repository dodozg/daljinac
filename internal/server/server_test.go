package server

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/dodozg/daljinac/internal/metrics"
)

const (
	tokenA = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	tokenB = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
)

func testConfig() Config {
	c := Config{Clients: []Client{{Name: "Firma A", Token: tokenA}, {Name: "Firma B", Token: tokenB}}}
	c.applyDefaults()
	return c
}

type testServer struct {
	*Server
	clock time.Time
}

func newTestServer(t *testing.T) *testServer {
	t.Helper()
	ts := &testServer{clock: time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)}
	ts.Server = New(testConfig(), slog.New(slog.NewTextHandler(io.Discard, nil)))
	ts.now = func() time.Time { return ts.clock }
	return ts
}

func report(host string, disk float64) metrics.Report {
	return metrics.Report{
		DeviceID: "id-" + host, Hostname: host, OS: "linux", Platform: "Ubuntu 24.04",
		CPUPercent: 10, MemTotalBytes: 100, MemUsedBytes: 50,
		Disks: []metrics.Disk{{Mount: "/", TotalBytes: 100, UsedBytes: uint64(disk)}},
	}
}

func (ts *testServer) post(t *testing.T, token string, body any) *http.Response {
	t.Helper()
	var buf bytes.Buffer
	if s, ok := body.(string); ok {
		buf.WriteString(s)
	} else if err := json.NewEncoder(&buf).Encode(body); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/report", &buf)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	ts.AgentHandler().ServeHTTP(rec, req)
	return rec.Result()
}

func (ts *testServer) devices(t *testing.T) []DeviceView {
	t.Helper()
	rec := httptest.NewRecorder()
	ts.ConsoleHandler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/devices", nil))
	var out []DeviceView
	if err := json.NewDecoder(rec.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	return out
}

func TestReportRequiresValidToken(t *testing.T) {
	ts := newTestServer(t)
	for name, token := range map[string]string{"none": "", "wrong": strings.Repeat("c", 40), "prefix": tokenA[:20]} {
		if got := ts.post(t, token, report("pc", 10)).StatusCode; got != http.StatusUnauthorized {
			t.Errorf("%s token: status %d, want 401", name, got)
		}
	}
	if n := len(ts.devices(t)); n != 0 {
		t.Fatalf("unauthorized reports were stored: %d devices", n)
	}
}

func TestReportRejectsBadBodies(t *testing.T) {
	ts := newTestServer(t)
	cases := map[string]struct {
		body any
		want int
	}{
		"not json":  {"{nope", http.StatusBadRequest},
		"invalid":   {metrics.Report{DeviceID: "x"}, http.StatusBadRequest},
		"too large": {`{"device_id":"` + strings.Repeat("x", MaxReportBytes) + `"}`, http.StatusRequestEntityTooLarge},
	}
	for name, c := range cases {
		if got := ts.post(t, tokenA, c.body).StatusCode; got != c.want {
			t.Errorf("%s: status %d, want %d", name, got, c.want)
		}
	}
}

func TestReportsAreSeparatedByClient(t *testing.T) {
	ts := newTestServer(t)
	// The same device ID at two customers must stay two devices.
	for _, tok := range []string{tokenA, tokenB} {
		if got := ts.post(t, tok, report("pc", 10)).StatusCode; got != http.StatusNoContent {
			t.Fatalf("status %d", got)
		}
	}
	got := ts.devices(t)
	if len(got) != 2 || got[0].Client != "Firma A" || got[1].Client != "Firma B" {
		t.Fatalf("got %+v", got)
	}
}

func TestLevelsAndOffline(t *testing.T) {
	ts := newTestServer(t)
	ts.post(t, tokenA, report("full-disk", 97))
	ts.post(t, tokenA, report("healthy", 10))

	byHost := func() map[string]DeviceView {
		m := map[string]DeviceView{}
		for _, d := range ts.devices(t) {
			m[d.Hostname] = d
		}
		return m
	}
	m := byHost()
	if m["full-disk"].Level != "critical" || m["healthy"].Level != "ok" {
		t.Fatalf("levels: %+v", m)
	}

	ts.clock = ts.clock.Add(6 * time.Minute)
	ts.post(t, tokenA, report("healthy", 10))
	m = byHost()
	if m["full-disk"].Online || m["full-disk"].Level != "offline" || !m["healthy"].Online {
		t.Fatalf("after 6 min: %+v", m)
	}
}

func TestDashboardRenders(t *testing.T) {
	ts := newTestServer(t)
	ts.post(t, tokenA, report("pc-<script>", 90))
	rec := httptest.NewRecorder()
	ts.ConsoleHandler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	body := rec.Body.String()
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	for _, want := range []string{"Firma A", "pc-&lt;script&gt;", `class="num warning"`} {
		if !strings.Contains(body, want) {
			t.Errorf("dashboard missing %q", want)
		}
	}
	if strings.Contains(body, "pc-<script>") {
		t.Error("hostname was not escaped")
	}
}

func TestMachinesPlural(t *testing.T) {
	for n, want := range map[int]string{
		1: "1 stroj", 2: "2 stroja", 4: "4 stroja", 5: "5 strojeva", 11: "11 strojeva",
		12: "12 strojeva", 21: "21 stroj", 22: "22 stroja", 111: "111 strojeva", 500: "500 strojeva",
	} {
		if got := machines(n); got != want {
			t.Errorf("machines(%d) = %q, want %q", n, got, want)
		}
	}
}

func TestConfigValidation(t *testing.T) {
	if err := testConfig().Validate(); err != nil {
		t.Fatalf("valid config rejected: %v", err)
	}
	cases := map[string]func(*Config){
		"public console": func(c *Config) { c.ConsoleListen = "0.0.0.0:8080" },
		"short token":    func(c *Config) { c.Clients[0].Token = "short" },
		"shared token":   func(c *Config) { c.Clients[1].Token = tokenA },
		"duplicate name": func(c *Config) { c.Clients[1].Name = "Firma A" },
		"no clients":     func(c *Config) { c.Clients = nil },
		"offline short":  func(c *Config) { c.OfflineAfter.Duration = time.Second },
	}
	for name, mutate := range cases {
		c := testConfig()
		mutate(&c)
		if err := c.Validate(); err == nil {
			t.Errorf("%s: expected an error", name)
		}
	}
}
