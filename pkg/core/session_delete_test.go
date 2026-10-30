package core

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDeleteSessionFiles(t *testing.T) {
	proj := t.TempDir()
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}

	// session "a": transcript + same-named dir; session "b": must survive
	must(os.WriteFile(filepath.Join(proj, "a.jsonl"), []byte("{}"), 0644))
	must(os.MkdirAll(filepath.Join(proj, "a", "subagents"), 0755))
	must(os.WriteFile(filepath.Join(proj, "a", "subagents", "x.jsonl"), []byte("{}"), 0644))
	must(os.WriteFile(filepath.Join(proj, "b.jsonl"), []byte("{}"), 0644))
	must(os.MkdirAll(filepath.Join(proj, "b", "tool-results"), 0755))
	must(os.MkdirAll(filepath.Join(proj, "memory"), 0755))

	if err := DeleteSessionFiles(Session{FilePath: filepath.Join(proj, "a.jsonl")}); err != nil {
		t.Fatal(err)
	}

	for _, gone := range []string{"a.jsonl", "a"} {
		if _, err := os.Stat(filepath.Join(proj, gone)); !os.IsNotExist(err) {
			t.Errorf("%s should be deleted (err=%v)", gone, err)
		}
	}
	for _, kept := range []string{"b.jsonl", "b", "memory"} {
		if _, err := os.Stat(filepath.Join(proj, kept)); err != nil {
			t.Errorf("%s must be kept: %v", kept, err)
		}
	}
}

func TestDeleteSessionFilesWithoutSiblingDir(t *testing.T) {
	proj := t.TempDir()
	path := filepath.Join(proj, "c.jsonl")
	if err := os.WriteFile(path, []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := DeleteSessionFiles(Session{FilePath: path}); err != nil {
		t.Fatalf("missing sibling dir must not be an error: %v", err)
	}
}

func TestDeleteSessionFilesMissingTranscript(t *testing.T) {
	proj := t.TempDir()
	if err := os.MkdirAll(filepath.Join(proj, "d"), 0755); err != nil {
		t.Fatal(err)
	}
	// transcript absent: report the error and do not touch the directory
	if err := DeleteSessionFiles(Session{FilePath: filepath.Join(proj, "d.jsonl")}); err == nil {
		t.Fatal("expected an error")
	}
	if _, err := os.Stat(filepath.Join(proj, "d")); err != nil {
		t.Fatalf("directory must be kept: %v", err)
	}
}
