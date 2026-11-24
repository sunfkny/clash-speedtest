package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func fixture(t *testing.T, path, value string) string {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(value), 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestPartyDiscovery(t *testing.T) {
	root := filepath.Join(t.TempDir(), "Party data")
	fixture(t, filepath.Join(root, "profile.yaml"), "current: selected\n")
	for _, tc := range []struct {
		separate string
		sub      string
	}{{"false", ""}, {"true", "selected"}} {
		fixture(t, filepath.Join(root, "config.yaml"), "diffWorkDir: "+tc.separate)
		dir := filepath.Join(root, "work", tc.sub)
		fixture(t, filepath.Join(dir, "config.yaml"), "proxies: []")
		s, err := partySnapshot(root)
		if err != nil || s.workDir != dir {
			t.Fatalf("snapshot = %+v, %v", s, err)
		}
	}
	active := filepath.Join(root, "work", "running")
	s := runningSource([]string{"mihomo", "-d", active})
	if s.workDir != active || s.path != filepath.Join(active, "config.yaml") {
		t.Fatalf("running config = %+v", s)
	}
	if s := runningSource([]string{"mihomo", "-t", "-d", active}); s.path != "" {
		t.Fatal("matched config validation process")
	}
}

func TestClashctlRunningConfig(t *testing.T) {
	root := filepath.Join(t.TempDir(), "custom home")
	dir := filepath.Join(root, "resources")
	config := filepath.Join(root, "data", "runtime.yaml")
	if s := runningSource([]string{"mihomo", "-d", dir, "-f", config}); s.path != "" {
		t.Fatal("matched unrelated core")
	}
	fixture(t, filepath.Join(root, "scripts", "cmd", "clashctl.sh"), "")
	s := runningSource([]string{"mihomo", "-d", dir, "-f", config})
	if s.path != config || s.workDir != dir || s.client != "clashctl" {
		t.Fatalf("source = %+v", s)
	}
}

func TestSelectConfig(t *testing.T) {
	candidates := sources{{"clashctl", "/b", "/resources"}, {"Clash Party", "/a", "/work"}}
	var prompt bytes.Buffer
	if _, err := chooseConfig(candidates, strings.NewReader(""), &prompt, false); err == nil || !strings.Contains(prompt.String(), "/b") {
		t.Fatal("expected noninteractive choices")
	}
	s, err := chooseConfig(candidates, strings.NewReader("2\n"), &prompt, true)
	if err != nil || s.path != "/b" {
		t.Fatalf("selected %+v, %v", s, err)
	}
}
