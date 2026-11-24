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
	active := filepath.Join(root, "work", "startup")
	fixture(t, filepath.Join(active, "config.yaml"), "proxies: []")
	args := []string{"mihomo", "-d", active}
	for _, id := range []string{"selected", "switched"} {
		selectedDir := filepath.Join(root, "work", id)
		selectedPath := fixture(t, filepath.Join(selectedDir, "config.yaml"), "proxies: []")
		fixture(t, filepath.Join(root, "profile.yaml"), "current: "+id)
		s, err := runningSource(args)
		if err != nil || s.path != selectedPath || s.workDir != selectedDir {
			t.Fatalf("after selecting %s: got %+v, %v", id, s, err)
		}
	}
	fixture(t, filepath.Join(root, "profile.yaml"), "current: missing")
	if s, err := runningSource(args); err == nil || s.path != "" {
		t.Fatalf("missing selected config: got %+v, %v", s, err)
	}
	fixture(t, filepath.Join(root, "config.yaml"), "diffWorkDir: false")
	if s, err := runningSource(args); err != nil || s.path != filepath.Join(root, "work", "config.yaml") {
		t.Fatalf("shared config: got %+v, %v", s, err)
	}
	if s, err := runningSource([]string{"mihomo", "-t", "-d", active}); err != nil || s.path != "" {
		t.Fatal("matched config validation process")
	}

}

func TestClashctlRunningConfig(t *testing.T) {
	root := filepath.Join(t.TempDir(), "custom home")
	dir := filepath.Join(root, "resources")
	config := filepath.Join(root, "data", "runtime.yaml")
	if s, err := runningSource([]string{"mihomo", "-d", dir, "-f", config}); err != nil || s.path != "" {
		t.Fatal("matched unrelated core")
	}
	fixture(t, filepath.Join(root, "scripts", "cmd", "clashctl.sh"), "")
	s, err := runningSource([]string{"mihomo", "-d", dir, "-f", config})
	if err != nil || s.path != config || s.workDir != dir || s.client != "clashctl" {
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
