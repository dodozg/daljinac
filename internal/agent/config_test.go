package agent

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const testToken = "0123456789abcdef0123456789abcdef"

func TestConfigValidate(t *testing.T) {
	good := Config{Server: "https://remote.example.hr", Token: testToken}
	if err := good.Validate(); err != nil {
		t.Fatalf("valid config rejected: %v", err)
	}
	bad := map[string]Config{
		"no server":      {Token: testToken},
		"no token":       {Server: good.Server},
		"plain http":     {Server: "http://remote.example.hr", Token: testToken},
		"space in token": {Server: good.Server, Token: "abc def"},
		"newline":        {Server: good.Server, Token: testToken + "\n"},
	}
	for name, c := range bad {
		if err := c.Validate(); err == nil {
			t.Errorf("%s: expected an error", name)
		}
	}
}

func writeFile(t *testing.T, content string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "agent.json")
	if err := os.WriteFile(p, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestLoadConfig(t *testing.T) {
	p := writeFile(t, `{"server": "https://remote.example.hr", "token": "`+testToken+`"}`)
	c, err := LoadConfig(p)
	if err != nil || c.Server != "https://remote.example.hr" || c.Token != testToken {
		t.Fatalf("got %+v, %v", c, err)
	}

	if _, err := LoadConfig(filepath.Join(t.TempDir(), "missing.json")); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("missing file: got %v, want fs.ErrNotExist", err)
	}
	if _, err := LoadConfig(writeFile(t, `{"sever": "https://x"}`)); err == nil {
		t.Error("a misspelled field should be an error")
	}
	if _, err := LoadConfig(writeFile(t, `{"server": `)); err == nil {
		t.Error("broken JSON should be an error")
	}
	if _, err := LoadConfig(writeFile(t, strings.Repeat(" ", maxConfigSize+1)+"{}")); err == nil {
		t.Error("an oversized file should be an error")
	}
}

func TestResolve(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing.json")
	fromFile := writeFile(t, `{"server": "https://file.example.hr", "token": "filetoken"}`)

	cases := []struct {
		name, path, server, token string
		want                      Config
		wantErr                   bool
	}{
		{"file only", fromFile, "", "", Config{"https://file.example.hr", "filetoken"}, false},
		{"overrides win", fromFile, "https://flag.example.hr", "envtoken", Config{"https://flag.example.hr", "envtoken"}, false},
		{"token from file, server overridden", fromFile, "https://flag.example.hr", "", Config{"https://flag.example.hr", "filetoken"}, false},
		{"no file, overrides complete", missing, "http://127.0.0.1:8081", "envtoken", Config{"http://127.0.0.1:8081", "envtoken"}, false},
		{"no file, token missing", missing, "https://flag.example.hr", "", Config{}, true},
		{"unsafe override", fromFile, "http://remote.example.hr", "", Config{}, true},
		{"broken file is not ignored", writeFile(t, "{"), "https://flag.example.hr", "envtoken", Config{}, true},
	}
	for _, c := range cases {
		got, err := Resolve(c.path, c.server, c.token)
		if (err != nil) != c.wantErr || got != c.want {
			t.Errorf("%s: got %+v, %v", c.name, got, err)
		}
	}
}

func TestWriteConfig(t *testing.T) {
	requirePrivateWrites(t)
	path := filepath.Join(t.TempDir(), "Daljinac", "agent.json")
	want := Config{Server: "https://remote.example.hr", Token: testToken}
	if err := WriteConfig(path, want); err != nil {
		t.Fatal(err)
	}
	// Overwriting must work too: that is how an update changes the token.
	want.Token = "another" + testToken
	if err := WriteConfig(path, want); err != nil {
		t.Fatal(err)
	}
	got, err := LoadConfig(path)
	if err != nil || got != want {
		t.Fatalf("read back %+v, %v; want %+v", got, err, want)
	}
	entries, _ := os.ReadDir(filepath.Dir(path))
	if len(entries) != 1 {
		t.Errorf("temporary files left behind: %v", entries)
	}
	checkPrivate(t, filepath.Dir(path), path)

	if err := WriteConfig(path, Config{Server: "http://remote.example.hr", Token: testToken}); err == nil {
		t.Error("an invalid config must not be written")
	}
}
