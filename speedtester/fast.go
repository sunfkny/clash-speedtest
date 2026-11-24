package speedtester

import (
	"fmt"

	"sync/atomic"
	"time"
)

type FastCheck struct {
	Window time.Duration
}

func DefaultFastCheck() FastCheck {
	return FastCheck{Window: 2 * time.Second}
}

func (c FastCheck) validate() error {
	if c.Window <= 0 {
		return fmt.Errorf("fast-window must be positive")
	}
	return nil
}

type fastProbe struct {
	start    time.Time
	check    FastCheck
	received atomic.Bool
}

func newFastProbe(check FastCheck) *fastProbe { return &fastProbe{start: time.Now(), check: check} }

func (p *fastProbe) Write(data []byte) (int, error) {
	if len(data) > 0 {
		p.received.Store(true)
	}
	return len(data), nil
}
