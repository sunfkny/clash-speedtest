package speedtester

import (
	"fmt"
	"math"
	"sync"
	"time"
)

type FastCheck struct {
	Window   time.Duration
	PeakMBps float64
}

func DefaultFastCheck() FastCheck {
	return FastCheck{Window: 2 * time.Second, PeakMBps: 1}
}

func (c FastCheck) validate() error {
	if c.Window <= 0 {
		return fmt.Errorf("fast-window must be positive")
	}
	if c.PeakMBps < 0 || math.IsNaN(c.PeakMBps) || math.IsInf(c.PeakMBps, 0) {
		return fmt.Errorf("fast-peak must be finite and nonnegative")
	}
	return nil
}

const peakInterval = 250 * time.Millisecond

// fastProbe counts bytes from concurrent responses in fixed intervals from download start.
type fastProbe struct {
	start time.Time
	check FastCheck

	mu          sync.Mutex
	bucketStart time.Duration
	bucketBytes int64
	peak        float64
}

func newFastProbe(check FastCheck) *fastProbe {
	return &fastProbe{start: time.Now(), check: check}
}

func (p *fastProbe) Write(data []byte) (int, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.record(time.Since(p.start), int64(len(data)))
	return len(data), nil
}

func (p *fastProbe) record(elapsed time.Duration, count int64) {
	if elapsed >= p.check.Window {
		return
	}
	start := elapsed.Truncate(peakInterval)
	if start != p.bucketStart {
		p.peak = max(p.peak, float64(p.bucketBytes)/peakInterval.Seconds())
		p.bucketStart = start
		p.bucketBytes = 0
	}

	p.bucketBytes += count
}

func (p *fastProbe) failure(elapsed time.Duration) string {
	p.mu.Lock()
	defer p.mu.Unlock()
	interval := min(peakInterval, min(elapsed, p.check.Window)-p.bucketStart)
	peak := p.peak
	if interval > 0 {
		peak = max(peak, float64(p.bucketBytes)/interval.Seconds())
	}
	peakMBps := peak / (1024 * 1024)

	if peakMBps < p.check.PeakMBps {
		return fmt.Sprintf("fast check failed: peak %.3f MB/s below %g MB/s", peakMBps, p.check.PeakMBps)
	}
	return ""
}
