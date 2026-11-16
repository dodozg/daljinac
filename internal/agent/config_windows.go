package agent

import (
	"os"
	"path/filepath"

	"golang.org/x/sys/windows"
)

// privateSDDL: owned by Administrators; full access for SYSTEM and
// Administrators only; inherited permissions from the parent are cut off
// (P), so the default "Users can read" on ProgramData does not apply.
// OICI makes files created in the directory inherit the same rule.
const privateSDDL = "O:BAD:P(A;OICI;FA;;;SY)(A;OICI;FA;;;BA)"

// DefaultConfigPath is where the installer writes the config and where the
// service reads it.
func DefaultConfigPath() string {
	base := os.Getenv("ProgramData")
	if base == "" {
		base = `C:\ProgramData`
	}
	return filepath.Join(base, "Daljinac", "agent.json")
}

func makePrivateDir(dir string) error {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	// The directory may already exist, possibly created by an ordinary user
	// (anyone may create folders in ProgramData), so the owner is reset too.
	return restrict(dir)
}

func makePrivateFile(path string) error { return restrict(path) }

func restrict(path string) error {
	sd, err := windows.SecurityDescriptorFromString(privateSDDL)
	if err != nil {
		return err
	}
	owner, _, err := sd.Owner()
	if err != nil {
		return err
	}
	dacl, _, err := sd.DACL()
	if err != nil {
		return err
	}
	return windows.SetNamedSecurityInfo(path, windows.SE_FILE_OBJECT,
		windows.OWNER_SECURITY_INFORMATION|windows.DACL_SECURITY_INFORMATION|windows.PROTECTED_DACL_SECURITY_INFORMATION,
		owner, nil, dacl, nil)
}
