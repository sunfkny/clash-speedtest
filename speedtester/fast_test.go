package speedtester

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/metacubex/mihomo/adapter"
	"github.com/metacubex/mihomo/adapter/outbound"
)

func TestFastModeCancelsStalledDownload(t *testing.T) {
	canceled := make(chan struct{}, 2)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodHead {
			return
		}
		w.WriteHeader(http.StatusOK)
		w.(http.Flusher).Flush()
		<-r.Context().Done()
		canceled <- struct{}{}
	}))
	defer server.Close()
	st, err := New(&Config{ServerURL: server.URL, Mode: SpeedModeFast, DownloadSize: 1024 * 1024, Concurrent: 2, MaxLatency: time.Second, Timeout: 4 * time.Second, FastCheck: &FastCheck{Window: 300 * time.Millisecond}})
	if err != nil {
		t.Fatal(err)
	}
	proxy := &CProxy{Proxy: adapter.NewProxy(outbound.NewDirect())}
	result := st.testProxy(context.Background(), "stalled", proxy)
	if !strings.Contains(result.DownloadError, "未收到下载数据") || result.DownloadSpeed != 0 {
		t.Fatalf("unexpected download result: %+v", result)
	}
	for range 2 {
		select {
		case <-canceled:
		case <-time.After(time.Second):
			t.Fatal("download connection was not canceled")
		}
	}
}

func TestFastModeKeepsSlowDownloads(t *testing.T) {
	const chunk = 16
	data := make([]byte, chunk)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodHead {
			return
		}
		w.Write(data)
		w.(http.Flusher).Flush()
		time.Sleep(500 * time.Millisecond)
		w.Write(data)
	}))
	defer server.Close()
	st, err := New(&Config{ServerURL: server.URL + "/file", Mode: SpeedModeFast, DownloadSize: 2 * chunk, Concurrent: 1, MaxLatency: time.Second, Timeout: 3 * time.Second, FastCheck: &FastCheck{Window: 400 * time.Millisecond}})
	if err != nil {
		t.Fatal(err)
	}
	result := st.testProxy(context.Background(), "burst", &CProxy{Proxy: adapter.NewProxy(outbound.NewDirect())})
	if result.DownloadError != "" || result.DownloadSize != 2*chunk {
		t.Fatalf("unexpected download result: %+v", result)
	}
}
