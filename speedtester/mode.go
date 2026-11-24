package speedtester

import (
	"fmt"
	"strings"
)

type SpeedMode string

const (
	SpeedModeTTFB     SpeedMode = "ttfb"
	SpeedModeFast     SpeedMode = "fast"
	SpeedModeDownload SpeedMode = "download"
	SpeedModeFull     SpeedMode = "full"
)

func ParseSpeedMode(value string) (SpeedMode, error) {
	mode := SpeedMode(strings.ToLower(strings.TrimSpace(value)))
	switch mode {
	case SpeedModeFast, SpeedModeTTFB, SpeedModeDownload, SpeedModeFull:
		return mode, nil
	default:
		return "", fmt.Errorf("unsupported speed mode %q", value)
	}
}

func (m SpeedMode) IsFast() bool { return m == SpeedModeFast }

func (m SpeedMode) LatencyOnly() bool {
	return m == SpeedModeTTFB
}

func (m SpeedMode) UploadEnabled() bool {
	return m == SpeedModeFull
}
