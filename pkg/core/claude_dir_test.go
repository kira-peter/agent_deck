package core

import (
	"runtime"
	"testing"
)

func TestEncodePath(t *testing.T) {
	cases := map[string]string{
		`D:\jarvis\claude-manager`:          "D--jarvis-claude-manager",
		`D:\StarCore\config\config_tool`:    "D--StarCore-config-config-tool",
		`D:\sg2\afk-tool-cfg-generator-sg2`: "D--sg2-afk-tool-cfg-generator-sg2",
		`C:\Users\TU`:                       "C--Users-TU",
	}
	for in, want := range cases {
		if got := EncodePath(in); got != want {
			t.Errorf("EncodePath(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestEncodeDirSegment(t *testing.T) {
	cases := map[string]string{
		"config_tool":       "config-tool",
		"ls-server":         "ls-server",
		"my project":        "my-project",
		"100-star":          "100-star",
		"firewareditconfig": "firewareditconfig",
	}
	for in, want := range cases {
		if got := encodeDirSegment(in); got != want {
			t.Errorf("encodeDirSegment(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestDecodePathFallback(t *testing.T) {
	want := "/StarCore/config/config/tool"
	if runtime.GOOS == "windows" {
		want = `D:\StarCore\config\config\tool`
	}
	if got := DecodePath("D--StarCore-config-config-tool"); got != want {
		t.Errorf("DecodePath = %q, want %q", got, want)
	}
}
