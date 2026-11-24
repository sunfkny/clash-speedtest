package main

import (
	"path/filepath"
	"testing"

	"github.com/faceair/clash-speedtest/speedtester"
)

func TestInstalledClashctlLoadsProvider(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	for _, root := range []string{filepath.Join(home, ".clashctl"), filepath.Join(home, "custom install")} {
		if filepath.Base(root) != ".clashctl" {
			t.Setenv("CLASHCTL_HOME", root)
		} else {
			t.Setenv("CLASHCTL_HOME", "")
		}
		fixture(t, filepath.Join(root, "scripts", "cmd", "clashctl.sh"), "")
		config := fixture(t, filepath.Join(root, "data", "runtime.yaml"), "proxy-providers:\n  local: {type: file, path: nodes.yaml}\n")
		fixture(t, filepath.Join(root, "resources", "nodes.yaml"), "proxies:\n  - {name: test, type: socks5, server: 127.0.0.1, port: 1080}\n")
		found, err := installedConfigs()
		if err != nil || len(found) != 1 || found[0].path != config {
			t.Fatalf("found %+v, %v", found, err)
		}
		tester, err := speedtester.New(&speedtester.Config{ConfigPaths: found[0].path, WorkDir: found[0].workDir, ServerURL: "http://localhost/file"})
		if err != nil {
			t.Fatal(err)
		}
		proxies, err := tester.LoadProxies()
		if err != nil || proxies["[local] test"] == nil {
			t.Fatalf("missing node from relative provider: %v", err)
		}
	}
}
