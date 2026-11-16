package main

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"sync"
)

// eventWriter is the part of *eventlog.Log the handler uses.
type eventWriter interface {
	Info(eid uint32, msg string) error
	Warning(eid uint32, msg string) error
	Error(eid uint32, msg string) error
}

// eventID is the one ID we use; EventCreate's message file accepts 1–1000.
const eventID = 1

// eventLogHandler formats records like slog's text handler and writes each
// one as an event of the matching level.
type eventLogHandler struct {
	out   eventWriter
	text  slog.Handler // writes into buf
	buf   *bytes.Buffer
	mu    *sync.Mutex // guards buf, shared by handlers derived via With*
	level slog.Leveler
}

func newEventLogHandler(out eventWriter) *eventLogHandler {
	buf := new(bytes.Buffer)
	return &eventLogHandler{
		out: out,
		// The event already carries the time and level.
		text: slog.NewTextHandler(buf, &slog.HandlerOptions{ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if len(groups) == 0 && (a.Key == slog.TimeKey || a.Key == slog.LevelKey) {
				return slog.Attr{}
			}
			return a
		}}),
		buf:   buf,
		mu:    new(sync.Mutex),
		level: slog.LevelInfo,
	}
}

func (h *eventLogHandler) Enabled(_ context.Context, l slog.Level) bool {
	return l >= h.level.Level()
}

func (h *eventLogHandler) Handle(ctx context.Context, r slog.Record) error {
	h.mu.Lock()
	h.buf.Reset()
	err := h.text.Handle(ctx, r)
	msg := strings.TrimSpace(h.buf.String())
	h.mu.Unlock()
	if err != nil {
		return err
	}
	switch {
	case r.Level >= slog.LevelError:
		return h.out.Error(eventID, msg)
	case r.Level >= slog.LevelWarn:
		return h.out.Warning(eventID, msg)
	default:
		return h.out.Info(eventID, msg)
	}
}

func (h *eventLogHandler) WithAttrs(as []slog.Attr) slog.Handler {
	c := *h
	c.text = h.text.WithAttrs(as)
	return &c
}

func (h *eventLogHandler) WithGroup(name string) slog.Handler {
	c := *h
	c.text = h.text.WithGroup(name)
	return &c
}
