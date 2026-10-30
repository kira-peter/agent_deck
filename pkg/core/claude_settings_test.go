package core

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestClaudeSettingsKeepsUnknownKeysAndOrder(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	src := `{"env":{"A":"1","B":"<x>"},"zzz":{"x":[1,2]},"permissions":{"allow":["Bash(ls)"],"defaultMode":"plan"},"model":"m"}`
	if err := os.WriteFile(path, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}

	s, err := LoadClaudeSettings(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Set("permissions.defaultMode", "acceptEdits"); err != nil {
		t.Fatal(err)
	}
	if err := s.Set("autoCompact", true); err != nil {
		t.Fatal(err)
	}
	if err := s.Unset("model"); err != nil {
		t.Fatal(err)
	}
	if err := s.Save(); err != nil {
		t.Fatal(err)
	}

	got, err := LoadClaudeSettings(path)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"env", "zzz", "permissions", "autoCompact"}; !reflect.DeepEqual(got.Keys(), want) {
		t.Fatalf("keys = %v, want %v", got.Keys(), want)
	}
	if raw, _ := got.Lookup("permissions.defaultMode"); string(raw) != `"acceptEdits"` {
		t.Fatalf("defaultMode = %s", raw)
	}
	if want := []string{"Bash(ls)"}; !reflect.DeepEqual(got.StringList("permissions.allow"), want) {
		t.Fatalf("allow = %v", got.StringList("permissions.allow"))
	}
	if raw, _ := got.Lookup("zzz.x"); string(raw) != `[1,2]` && !strings.Contains(string(raw), "2") {
		t.Fatalf("zzz.x = %s", raw)
	}
	if raw, _ := got.Lookup("env.B"); string(raw) != `"<x>"` {
		t.Fatalf("env.B = %s (HTML must not be escaped)", raw)
	}
	if _, ok := got.Lookup("model"); ok {
		t.Fatal("model should have been removed")
	}
}

func TestClaudeSettingsMissingFile(t *testing.T) {
	s, err := LoadClaudeSettings(filepath.Join(t.TempDir(), "none.json"))
	if err != nil || len(s.Keys()) != 0 {
		t.Fatalf("missing file: keys=%v err=%v", s.Keys(), err)
	}
}
