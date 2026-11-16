package agent

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// Config is what the agent needs to report: where to and as which client.
// It lives in a file readable only by administrators (see WriteConfig),
// because the token lets anyone send reports in the client's name.
type Config struct {
	Server string `json:"server"`
	Token  string `json:"token"`
}

// maxConfigSize keeps a wrong path (say, a log file) from being read whole.
const maxConfigSize = 64 << 10

// Validate checks that the config is complete and the server URL is safe to
// send the token to.
func (c Config) Validate() error {
	if c.Server == "" {
		return errors.New("server URL is missing")
	}
	if _, err := ReportURL(c.Server); err != nil {
		return err
	}
	if c.Token == "" {
		return errors.New("token is missing")
	}
	if strings.ContainsFunc(c.Token, func(r rune) bool { return r <= ' ' || r == 0x7f }) {
		return errors.New("token contains spaces or control characters")
	}
	return nil
}

// LoadConfig reads a config file. A missing file returns an error that
// matches fs.ErrNotExist, so callers can treat it as "not installed".
func LoadConfig(path string) (Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return Config{}, err
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, maxConfigSize+1))
	if err != nil {
		return Config{}, fmt.Errorf("%s: %w", path, err)
	}
	if len(data) > maxConfigSize {
		return Config{}, fmt.Errorf("%s: larger than %d bytes", path, maxConfigSize)
	}
	var c Config
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields() // a typo in a field name should fail, not be ignored
	if err := dec.Decode(&c); err != nil {
		return Config{}, fmt.Errorf("%s: %w", path, err)
	}
	return c, nil
}

// WriteConfig writes the config so that only administrators and the system
// (the agent service) can read it. It replaces an existing file atomically.
func WriteConfig(path string, c Config) error {
	if err := c.Validate(); err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')

	dir := filepath.Dir(path)
	if err := makePrivateDir(dir); err != nil {
		return fmt.Errorf("create %s: %w", dir, err)
	}
	tmp, err := os.CreateTemp(dir, ".agent-*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name()) // no-op after a successful rename
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := makePrivateFile(tmp.Name()); err != nil {
		return fmt.Errorf("restrict %s: %w", tmp.Name(), err)
	}
	return os.Rename(tmp.Name(), path)
}

// Resolve combines the config file with overrides from the command line or
// environment, which are used for trials and development. A missing file is
// fine as long as the overrides complete the config.
func Resolve(path, server, token string) (Config, error) {
	c, err := LoadConfig(path)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return Config{}, err
	}
	missing := err != nil
	if server != "" {
		c.Server = server
	}
	if token != "" {
		c.Token = token
	}
	if err := c.Validate(); err != nil {
		if missing {
			return Config{}, fmt.Errorf("%w (config file %s does not exist)", err, path)
		}
		return Config{}, fmt.Errorf("%s: %w", path, err)
	}
	return c, nil
}
