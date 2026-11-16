package main

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"time"

	"golang.org/x/sys/windows/svc"
)

// execute runs the handler as the service manager would and returns the
// states it reported and its exit values.
func execute(t *testing.T, h *serviceHandler, send func(chan<- svc.ChangeRequest)) ([]svc.State, bool, uint32) {
	t.Helper()
	req := make(chan svc.ChangeRequest)
	status := make(chan svc.Status, 10)
	type result struct {
		specific bool
		code     uint32
	}
	res := make(chan result, 1)
	go func() {
		s, c := h.Execute(nil, req, status)
		res <- result{s, c}
	}()
	send(req)
	var r result
	select {
	case r = <-res:
	case <-time.After(10 * time.Second):
		t.Fatal("Execute did not return")
	}
	close(status)
	var states []svc.State
	for s := range status {
		states = append(states, s.State)
	}
	return states, r.specific, r.code
}

func TestServiceStops(t *testing.T) {
	stopped := false
	h := &serviceHandler{run: func(ctx context.Context) error {
		<-ctx.Done()
		stopped = true
		return nil
	}}
	states, specific, code := execute(t, h, func(req chan<- svc.ChangeRequest) {
		req <- svc.ChangeRequest{Cmd: svc.Stop}
	})
	want := []svc.State{svc.StartPending, svc.Running, svc.StopPending}
	if len(states) != len(want) || states[0] != want[0] || states[1] != want[1] || states[2] != want[2] {
		t.Errorf("states %v, want %v", states, want)
	}
	if specific || code != 0 {
		t.Errorf("exit %v/%d, want a clean stop", specific, code)
	}
	if !stopped {
		t.Error("run was not cancelled")
	}
}

func TestServiceReportsEarlyExit(t *testing.T) {
	// A missing config makes run return at once; the service must report a
	// failure, so Windows restarts it per the recovery settings.
	h := &serviceHandler{run: func(context.Context) error { return errors.New("no config") }}
	_, specific, code := execute(t, h, func(chan<- svc.ChangeRequest) {})
	if !specific || code != exitRunFailed {
		t.Errorf("exit %v/%d, want a service-specific failure", specific, code)
	}
	if h.err == nil || h.err.Error() != "no config" {
		t.Errorf("err %v", h.err)
	}
}

type fakeEvents struct{ got []string }

func (f *fakeEvents) Info(_ uint32, m string) error    { f.got = append(f.got, "I "+m); return nil }
func (f *fakeEvents) Warning(_ uint32, m string) error { f.got = append(f.got, "W "+m); return nil }
func (f *fakeEvents) Error(_ uint32, m string) error   { f.got = append(f.got, "E "+m); return nil }

func TestEventLogHandler(t *testing.T) {
	f := &fakeEvents{}
	log := slog.New(newEventLogHandler(f)).With("version", "v1")
	log.Debug("hidden")
	log.Info("started", "server", "https://remote.example.hr")
	log.Warn("report failed", "err", "timeout")
	log.Error("stopped")

	want := []string{
		`I msg=started version=v1 server=https://remote.example.hr`,
		`W msg="report failed" version=v1 err=timeout`,
		`E msg=stopped version=v1`,
	}
	if strings.Join(f.got, "\n") != strings.Join(want, "\n") {
		t.Errorf("got\n%s\nwant\n%s", strings.Join(f.got, "\n"), strings.Join(want, "\n"))
	}
}

func TestCommandLine(t *testing.T) {
	got := commandLine(`C:\Program Files\Daljinac\daljinac-agent.exe`, []string{"-config", `C:\ProgramData\Daljinac\agent.json`})
	want := `"C:\Program Files\Daljinac\daljinac-agent.exe" -config C:\ProgramData\Daljinac\agent.json`
	if got != want {
		t.Errorf("got %s\nwant %s", got, want)
	}
}
