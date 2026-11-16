package agent

import (
	"strings"
	"testing"

	"golang.org/x/sys/windows"
)

// requirePrivateWrites skips when not elevated: making Administrators the
// owner needs an elevated process. The installer always runs elevated, and
// so do the GitHub Windows runners, where this test does run.
func requirePrivateWrites(t *testing.T) {
	t.Helper()
	if !windows.GetCurrentProcessToken().IsElevated() {
		t.Skip("needs an elevated process")
	}
}

func checkPrivate(t *testing.T, paths ...string) {
	t.Helper()
	for _, p := range paths {
		sd, err := windows.GetNamedSecurityInfo(p, windows.SE_FILE_OBJECT,
			windows.OWNER_SECURITY_INFORMATION|windows.DACL_SECURITY_INFORMATION)
		if err != nil {
			t.Fatal(err)
		}
		sddl := sd.String()
		owner, _, _ := sd.Owner()
		if !owner.IsWellKnown(windows.WinBuiltinAdministratorsSid) {
			t.Errorf("%s: owner is not Administrators: %s", p, sddl)
		}
		ctl, _, _ := sd.Control()
		if ctl&windows.SE_DACL_PROTECTED == 0 {
			t.Errorf("%s: permissions are inherited from the parent: %s", p, sddl)
		}
		// Every access entry must be for SYSTEM or Administrators.
		dacl := sddl[strings.Index(sddl, "D:"):]
		for _, ace := range strings.Split(dacl, "(")[1:] {
			ace = strings.TrimSuffix(ace, ")")
			if !strings.HasSuffix(ace, ";;;SY") && !strings.HasSuffix(ace, ";;;BA") {
				t.Errorf("%s: unexpected access entry (%s) in %s", p, ace, sddl)
			}
		}
	}
}
