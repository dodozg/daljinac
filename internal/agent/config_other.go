//go:build !windows

package agent

import "os"

// DefaultConfigPath is where the config lives on macOS and Linux. Installing
// there is planned for the Mac and Linux phase; the agent already reads it.
func DefaultConfigPath() string { return "/etc/daljinac/agent.json" }

func makePrivateDir(dir string) error {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	return os.Chmod(dir, 0o700)
}

func makePrivateFile(path string) error { return os.Chmod(path, 0o600) }
