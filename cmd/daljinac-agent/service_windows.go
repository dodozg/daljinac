package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/eventlog"
	"golang.org/x/sys/windows/svc/mgr"

	"github.com/dodozg/daljinac/internal/agent"
)

const (
	serviceName        = "DaljinacAgent"
	serviceDisplayName = "Daljinac Agent"
	// Shown to technicians in services.msc, hence in Croatian.
	serviceDescription = "Svake minute šalje zauzeće CPU-a, RAM-a i diskova Daljinac serveru."
	exeName            = "daljinac-agent.exe"
)

func isService() bool {
	ok, err := svc.IsWindowsService()
	return err == nil && ok
}

// runService is the entry point when the service manager starts the agent.
// Logs go to the Windows event log (Application, source DaljinacAgent).
func runService(configPath string, interval time.Duration) {
	var log *slog.Logger
	if el, err := eventlog.Open(serviceName); err == nil {
		defer el.Close()
		log = slog.New(newEventLogHandler(el))
	} else {
		log = slog.New(slog.NewTextHandler(io.Discard, nil))
	}

	h := &serviceHandler{run: func(ctx context.Context) error {
		// Only the config file counts here; stray environment variables on
		// the machine must not redirect the service.
		cfg, err := agent.Resolve(configPath, "", "")
		if err != nil {
			return err
		}
		report, err := newReporter(cfg)
		if err != nil {
			return err
		}
		log.Info("daljinac-agent service started", "version", version, "server", cfg.Server, "interval", interval)
		agent.Loop(ctx, log, interval, report)
		return nil
	}}
	if err := svc.Run(serviceName, h); err != nil {
		log.Error("service failed", "err", err)
	}
	if h.err != nil {
		log.Error("service stopped", "err", h.err)
	}
}

// serviceHandler adapts a long-running function to the service manager:
// it runs until Stop or Shutdown, and reports a failure if the function
// returns on its own, so the recovery settings restart the service.
type serviceHandler struct {
	run func(context.Context) error
	err error // why run returned early, if it did
}

const exitRunFailed = 1

func (h *serviceHandler) Execute(_ []string, req <-chan svc.ChangeRequest, status chan<- svc.Status) (bool, uint32) {
	status <- svc.Status{State: svc.StartPending}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- h.run(ctx) }()

	const accepts = svc.AcceptStop | svc.AcceptShutdown
	status <- svc.Status{State: svc.Running, Accepts: accepts}
	for {
		select {
		case err := <-done:
			if err == nil {
				err = errors.New("stopped unexpectedly")
			}
			h.err = err
			return true, exitRunFailed
		case c := <-req:
			switch c.Cmd {
			case svc.Interrogate:
				status <- c.CurrentStatus
			case svc.Stop, svc.Shutdown:
				status <- svc.Status{State: svc.StopPending}
				cancel()
				select {
				case <-done:
				case <-time.After(20 * time.Second): // a report in flight; give up on it
				}
				return false, 0
			}
		}
	}
}

func installService(log *slog.Logger, configPath, server, token string) error {
	if err := requireAdmin(); err != nil {
		return err
	}
	// On an update the existing config is kept unless new values are given.
	cfg, err := agent.Resolve(configPath, server, token)
	if err != nil {
		return fmt.Errorf("%w; give -server and set DALJINAC_TOKEN", err)
	}
	configPath, err = filepath.Abs(configPath)
	if err != nil {
		return err
	}

	m, err := mgr.Connect()
	if err != nil {
		return err
	}
	defer m.Disconnect()

	s, err := m.OpenService(serviceName)
	if err == nil {
		defer s.Close()
		// Stop first: a running service holds its exe, which is replaced below.
		if err := stopAndWait(s); err != nil {
			return err
		}
	} else {
		s = nil
	}

	exe, err := copySelf()
	if err != nil {
		return err
	}
	if err := agent.WriteConfig(configPath, cfg); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	log.Info("config written", "path", configPath)

	c := mgr.Config{
		DisplayName:      serviceDisplayName,
		Description:      serviceDescription,
		StartType:        mgr.StartAutomatic,
		DelayedAutoStart: true, // let the machine finish booting first
		ErrorControl:     mgr.ErrorNormal,
	}
	args := []string{"-config", configPath}
	if s == nil {
		s, err = m.CreateService(serviceName, exe, c, args...)
		if err != nil {
			return fmt.Errorf("create service: %w", err)
		}
		defer s.Close()
	} else {
		// Start from the current config: UpdateConfig sends every field,
		// and zero values (e.g. ServiceType 0) would be rejected.
		cur, err := s.Config()
		if err != nil {
			return fmt.Errorf("read service config: %w", err)
		}
		cur.BinaryPathName = commandLine(exe, args)
		cur.DisplayName = c.DisplayName
		cur.Description = c.Description
		cur.StartType = c.StartType
		cur.DelayedAutoStart = c.DelayedAutoStart
		cur.ErrorControl = c.ErrorControl
		cur.ServiceStartName = "LocalSystem"
		cur.Password = ""
		if err := s.UpdateConfig(cur); err != nil {
			return fmt.Errorf("update service: %w", err)
		}
	}

	// Restart after a crash or an early exit: twice after a minute, then
	// every five minutes. The failure count resets after a day.
	if err := s.SetRecoveryActions([]mgr.RecoveryAction{
		{Type: mgr.ServiceRestart, Delay: time.Minute},
		{Type: mgr.ServiceRestart, Delay: time.Minute},
		{Type: mgr.ServiceRestart, Delay: 5 * time.Minute},
	}, uint32((24 * time.Hour).Seconds())); err != nil {
		return fmt.Errorf("set recovery: %w", err)
	}
	if err := s.SetRecoveryActionsOnNonCrashFailures(true); err != nil {
		return fmt.Errorf("set recovery: %w", err)
	}

	_ = eventlog.Remove(serviceName) // re-register cleanly on an update
	if err := eventlog.InstallAsEventCreate(serviceName, eventlog.Error|eventlog.Warning|eventlog.Info); err != nil {
		return fmt.Errorf("register event log source: %w", err)
	}

	if err := s.Start(); err != nil {
		return fmt.Errorf("start service: %w", err)
	}
	if err := waitFor(s, svc.Running, 30*time.Second); err != nil {
		return fmt.Errorf("%w; see Event Viewer > Windows Logs > Application, source %s", err, serviceName)
	}
	log.Info("service installed and running", "name", serviceName, "exe", exe, "server", cfg.Server)
	return nil
}

func uninstallService(log *slog.Logger, configPath string) error {
	if err := requireAdmin(); err != nil {
		return err
	}
	m, err := mgr.Connect()
	if err != nil {
		return err
	}
	defer m.Disconnect()

	if s, err := m.OpenService(serviceName); err == nil {
		if err := stopAndWait(s); err != nil {
			s.Close()
			return err
		}
		err = s.Delete()
		s.Close()
		if err != nil {
			return fmt.Errorf("delete service: %w", err)
		}
		log.Info("service removed", "name", serviceName)
	} else {
		log.Info("service is not installed", "name", serviceName)
	}
	_ = eventlog.Remove(serviceName)

	// The config holds the client's token, so it goes. Only our own file is
	// removed, and its directory only if nothing else is in it.
	if err := os.Remove(configPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove config: %w", err)
	}
	_ = os.Remove(filepath.Dir(configPath))

	exe := installedExe()
	if err := os.Remove(exe); err != nil && !errors.Is(err, os.ErrNotExist) {
		// Windows cannot delete a running exe, e.g. when uninstalling with
		// the installed copy itself.
		log.Warn("could not remove the program; delete it by hand", "path", exe, "err", err)
	} else {
		_ = os.Remove(filepath.Dir(exe))
	}
	log.Info("uninstalled")
	return nil
}

func requireAdmin() error {
	if !windows.GetCurrentProcessToken().IsElevated() {
		return errors.New("run this from a terminal opened as administrator")
	}
	return nil
}

func installedExe() string {
	base := os.Getenv("ProgramFiles")
	if base == "" {
		base = `C:\Program Files`
	}
	return filepath.Join(base, "Daljinac", exeName)
}

// copySelf copies the running exe to Program Files, where only
// administrators can change it, so the downloaded file can be deleted.
func copySelf() (string, error) {
	src, err := os.Executable()
	if err != nil {
		return "", err
	}
	dst := installedExe()
	if strings.EqualFold(filepath.Clean(src), filepath.Clean(dst)) {
		return dst, nil // reinstalling with the installed copy
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return "", err
	}
	in, err := os.Open(src)
	if err != nil {
		return "", err
	}
	defer in.Close()
	tmp := dst + ".new"
	out, err := os.Create(tmp)
	if err != nil {
		return "", err
	}
	_, err = io.Copy(out, in)
	if cerr := out.Close(); err == nil {
		err = cerr
	}
	if err == nil {
		err = os.Rename(tmp, dst)
	}
	if err != nil {
		os.Remove(tmp)
		return "", fmt.Errorf("copy program to %s: %w", dst, err)
	}
	return dst, nil
}

func commandLine(exe string, args []string) string {
	parts := []string{syscall.EscapeArg(exe)}
	for _, a := range args {
		parts = append(parts, syscall.EscapeArg(a))
	}
	return strings.Join(parts, " ")
}

func stopAndWait(s *mgr.Service) error {
	st, err := s.Query()
	if err != nil {
		return err
	}
	if st.State == svc.Stopped {
		return nil
	}
	if st.State != svc.StopPending {
		if _, err := s.Control(svc.Stop); err != nil {
			return fmt.Errorf("stop service: %w", err)
		}
	}
	return waitFor(s, svc.Stopped, 30*time.Second)
}

func waitFor(s *mgr.Service, want svc.State, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for {
		st, err := s.Query()
		if err != nil {
			return err
		}
		if st.State == want {
			return nil
		}
		if want == svc.Running && st.State == svc.Stopped {
			return errors.New("service stopped right after starting")
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("service did not reach state %d in %s (now %d)", want, timeout, st.State)
		}
		time.Sleep(300 * time.Millisecond)
	}
}
