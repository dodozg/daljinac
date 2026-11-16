package agent

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"testing"
	"time"
)

// recorder keeps the messages logged through it.
type recorder struct {
	mu   sync.Mutex
	msgs []string
}

func (r *recorder) Enabled(context.Context, slog.Level) bool { return true }
func (r *recorder) Handle(_ context.Context, rec slog.Record) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.msgs = append(r.msgs, rec.Level.String()+" "+rec.Message)
	return nil
}
func (r *recorder) WithAttrs([]slog.Attr) slog.Handler { return r }
func (r *recorder) WithGroup(string) slog.Handler      { return r }

func TestLoopLogsOutageOnce(t *testing.T) {
	// Three failures, then success: one warning at the start of the outage
	// and one message when reports get through again.
	results := []error{errors.New("down"), errors.New("down"), errors.New("down"), nil, nil}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	calls := 0
	report := func(context.Context) error {
		err := results[calls]
		calls++
		if calls == len(results) {
			cancel()
		}
		return err
	}
	rec := &recorder{}
	done := make(chan struct{})
	go func() { Loop(ctx, slog.New(rec), time.Millisecond, report); close(done) }()
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("Loop did not stop after cancel")
	}

	want := []string{"WARN report failed", "INFO reports are getting through again"}
	if len(rec.msgs) != len(want) || rec.msgs[0] != want[0] || rec.msgs[1] != want[1] {
		t.Fatalf("logged %q, want %q", rec.msgs, want)
	}
}

func TestLoopStopsWhileWaiting(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		Loop(ctx, slog.New(&recorder{}), time.Hour, func(context.Context) error {
			t.Error("report should not run")
			return nil
		})
		close(done)
	}()
	cancel()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Loop did not stop while waiting for the first report")
	}
}
