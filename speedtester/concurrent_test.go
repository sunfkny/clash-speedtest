package speedtester

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/metacubex/mihomo/adapter"
	"github.com/metacubex/mihomo/adapter/outbound"
)

func TestStalledNodeDoesNotBlockOtherNodes(t *testing.T) {
	var downloads, probes atomic.Int32
	canceled := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodHead {
			probes.Add(1)
			return
		}
		if downloads.Add(1) == 1 {
			<-r.Context().Done()
			close(canceled)
			return
		}
		fmt.Fprint(w, "ok")
	}))
	defer server.Close()

	st, err := New(&Config{ServerURL: server.URL + "/file", NodeConcurrent: 2,
		Concurrent: 1, DownloadSize: 2, MaxLatency: time.Second, Timeout: 8 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	nodes := make(map[string]*CProxy)
	for i := range 5 {
		nodes[fmt.Sprint(i)] = &CProxy{Proxy: adapter.NewProxy(outbound.NewDirect())}
	}
	seen := make(map[string]bool)
	st.TestProxiesUntil(nodes, func(result *Result) bool {
		if result.DownloadSize != 2 || seen[result.ProxyName] {
			t.Errorf("unexpected result: %+v", result)
		}
		seen[result.ProxyName] = true
		return len(seen) < 2
	})
	if len(seen) != 2 || downloads.Load() != 3 || probes.Load() != 0 {
		t.Fatalf("got %d results, %d downloads, %d probes", len(seen), downloads.Load(), probes.Load())
	}
	select {
	case <-canceled:
	case <-time.After(time.Second):
		t.Fatal("early stop did not cancel the stalled download")
	}
}
