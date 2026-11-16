package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/dodozg/daljinac/internal/metrics"
)

// MinTokenLen is the shortest enrollment token the server accepts.
const MinTokenLen = 32

// Client is one customer company. Agents installed at that company
// authenticate with its enrollment token.
type Client struct {
	Name  string `json:"name"`
	Token string `json:"token"`
}

// Config is the server configuration, read from a JSON file.
type Config struct {
	// AgentListen is where agents send reports. Put it behind a TLS reverse proxy.
	AgentListen string `json:"agent_listen"`
	// ConsoleListen serves the technician console. It must be a loopback
	// address until the console has its own login with 2FA.
	ConsoleListen string `json:"console_listen"`
	// OfflineAfter is how long without a report before a device shows as offline.
	OfflineAfter Duration            `json:"offline_after"`
	Thresholds   *metrics.Thresholds `json:"thresholds,omitempty"`
	Clients      []Client            `json:"clients"`
}

// Duration is a time.Duration written as a string such as "5m" in JSON.
type Duration struct{ time.Duration }

func (d *Duration) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return fmt.Errorf("duration must be a string like \"5m\": %w", err)
	}
	v, err := time.ParseDuration(s)
	if err != nil {
		return err
	}
	d.Duration = v
	return nil
}

func (d Duration) MarshalJSON() ([]byte, error) { return json.Marshal(d.String()) }

// LoadConfig reads, defaults and validates a config file.
func LoadConfig(path string) (Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}
	var c Config
	if err := json.Unmarshal(b, &c); err != nil {
		return Config{}, fmt.Errorf("%s: %w", path, err)
	}
	c.applyDefaults()
	if err := c.Validate(); err != nil {
		return Config{}, fmt.Errorf("%s: %w", path, err)
	}
	return c, nil
}

func (c *Config) applyDefaults() {
	if c.AgentListen == "" {
		c.AgentListen = "127.0.0.1:8081"
	}
	if c.ConsoleListen == "" {
		c.ConsoleListen = "127.0.0.1:8080"
	}
	if c.OfflineAfter.Duration == 0 {
		c.OfflineAfter.Duration = 5 * time.Minute
	}
	if c.Thresholds == nil {
		t := metrics.DefaultThresholds
		c.Thresholds = &t
	}
}

// Validate checks the config for mistakes that would make the server unsafe or useless.
func (c Config) Validate() error {
	if !isLoopback(c.ConsoleListen) {
		return fmt.Errorf("console_listen %q must be a loopback address (127.0.0.1 or ::1) until the console has a login; reach it through an SSH tunnel", c.ConsoleListen)
	}
	if _, _, err := net.SplitHostPort(c.AgentListen); err != nil {
		return fmt.Errorf("agent_listen: %w", err)
	}
	if c.OfflineAfter.Duration < time.Minute {
		return errors.New("offline_after must be at least 1m")
	}
	if err := c.Thresholds.Validate(); err != nil {
		return err
	}
	if len(c.Clients) == 0 {
		return errors.New("at least one client is required")
	}
	names, tokens := map[string]bool{}, map[string]bool{}
	for i, cl := range c.Clients {
		switch {
		case cl.Name == "":
			return fmt.Errorf("client %d: name is required", i)
		case names[cl.Name]:
			return fmt.Errorf("client %q listed twice", cl.Name)
		case len(cl.Token) < MinTokenLen:
			return fmt.Errorf("client %q: token must be at least %d characters (generate one with: openssl rand -hex 32)", cl.Name, MinTokenLen)
		case tokens[cl.Token]:
			return fmt.Errorf("client %q: token is shared with another client", cl.Name)
		}
		names[cl.Name], tokens[cl.Token] = true, true
	}
	return nil
}

func isLoopback(addr string) bool {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return false
	}
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
