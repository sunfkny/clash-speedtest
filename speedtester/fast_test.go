package speedtester

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/metacubex/mihomo/adapter"
	"github.com/metacubex/mihomo/adapter/outbound"
)

func TestFastPeakWithinWindow(t *testing.T) {
	probe := newFastProbe(DefaultFastCheck())
	// Two connections contribute to the same 250 ms interval.
	probe.record(time.Second, 128*1024)
	probe.record(time.Second, 128*1024-1)

	if probe.failure(2*time.Second) == "" {
		t.Fatal("sub-threshold peak passed")
	}
	beforeDeadline := newFastProbe(DefaultFastCheck())
	beforeDeadline.record(2*time.Second, 1024*1024)
	if beforeDeadline.failure(2*time.Second) == "" {
		t.Fatal("bytes after the observation window counted toward peak")
	}
	probe.record(time.Second+time.Millisecond, 1)
	probe.record(1500*time.Millisecond, 1)
	if reason := probe.failure(2 * time.Second); reason != "" {
		t.Fatalf("peak at threshold failed: %s", reason)
	}
}

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
	st, err := New(&Config{ServerURL: server.URL, Mode: SpeedModeFast, DownloadSize: 1024 * 1024, Concurrent: 2, MaxLatency: time.Second, Timeout: 4 * time.Second, FastCheck: &FastCheck{Window: 300 * time.Millisecond, PeakMBps: 1}})
	if err != nil {
		t.Fatal(err)
	}
	proxy := &CProxy{Proxy: adapter.NewProxy(outbound.NewDirect())}
	result := st.testProxy("stalled", proxy)
	if !strings.Contains(result.DownloadError, "fast check failed") || result.DownloadSpeed != 0 {
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

func TestFastModeContinuesAfterPeak(t *testing.T) {
	const chunk = 512 * 1024
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
	st, err := New(&Config{ServerURL: server.URL + "/file", Mode: SpeedModeFast, DownloadSize: 2 * chunk, Concurrent: 1, MaxLatency: time.Second, Timeout: 3 * time.Second, FastCheck: &FastCheck{Window: 400 * time.Millisecond, PeakMBps: 1}})
	if err != nil {
		t.Fatal(err)
	}
	result := st.testProxy("burst", &CProxy{Proxy: adapter.NewProxy(outbound.NewDirect())})
	if result.DownloadError != "" || result.DownloadSize != 2*chunk {
		t.Fatalf("unexpected download result: %+v", result)
	}
}
