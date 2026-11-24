package speedtester

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
)

const proxyFile = "proxies:\n  - {name: node, type: socks5, server: 127.0.0.1, port: 1080}\n"

func TestProviderWorkDir(t *testing.T) {
	root := t.TempDir()
	data := filepath.Join(root, "data,one")
	resources := filepath.Join(root, "resources")
	for _, dir := range []string{data, resources} {
		if err := os.Mkdir(dir, 0755); err != nil {
			t.Fatal(err)
		}
	}
	file := filepath.Join(data, "runtime.yaml")
	if err := os.WriteFile(filepath.Join(resources, "nodes.yaml"), []byte(proxyFile), 0600); err != nil {
		t.Fatal(err)
	}
	for _, vehicle := range []string{"type: file", "type: http\n    url: http://127.0.0.1:1/unreachable", "type: inline\n    payload: [{name: node, type: socks5, server: 127.0.0.1, port: 1080}]"} {
		content := "proxy-providers:\n  local:\n    " + vehicle + "\n    path: nodes.yaml\n    filter: ^node$\n    override:\n      additional-prefix: pre-\n      udp: true\n"
		if err := os.WriteFile(file, []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
		st := SpeedTester{config: &Config{ConfigPaths: file, WorkDir: resources}}
		nodes, err := st.LoadProxies()
		if err != nil {
			t.Fatal(err)
		}
		n := nodes["[local] pre-node"]
		if n == nil || n.Config["name"] != "pre-node" || n.Config["udp"] != true {
			t.Fatalf("provider %s did not resolve with its overrides", vehicle)
		}
	}
}

func TestRemoteProvider(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		if r.Header.Get("X-Subscription") != "test" {
			w.WriteHeader(403)
			return
		}
		w.Write([]byte(proxyFile))
	}))
	defer server.Close()
	st := SpeedTester{config: &Config{}}
	nodes, err := st.providerNodes("remote", map[string]any{"type": "http", "url": server.URL, "header": map[string]any{"X-Subscription": []string{"test"}}}, "")
	if err != nil || len(nodes) != 1 || requests.Load() != 1 {
		t.Fatalf("got %d nodes, %d requests, %v", len(nodes), requests.Load(), err)
	}
	if _, err := st.providerNodes("local", map[string]any{"type": "file", "path": "nodes.yaml"}, ""); err == nil {
		t.Fatal("remote config used a local file provider")
	}
}
