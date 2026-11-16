//go:build !windows

package agent

import (
	"os"
	"testing"
)

func requirePrivateWrites(*testing.T) {}

func checkPrivate(t *testing.T, dir, file string) {
	t.Helper()
	for p, want := range map[string]os.FileMode{dir: 0o700, file: 0o600} {
		fi, err := os.Stat(p)
		if err != nil {
			t.Fatal(err)
		}
		if got := fi.Mode().Perm(); got != want {
			t.Errorf("%s: mode %o, want %o", p, got, want)
		}
	}
}
