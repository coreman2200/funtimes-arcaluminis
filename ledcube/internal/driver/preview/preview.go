package preview

import (
	"context"
	"encoding/base64"
	"sync"
	"time"

	"github.com/coreman2200/funtimes-arcaluminis/ledcube/internal/render"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type Driver struct {
	ctx      context.Context
	dim      render.Dimensions
	throttle time.Duration
	lastEmit time.Time
	mu       sync.Mutex
}

func New(ctx context.Context, dim render.Dimensions) *Driver {
	return &Driver{
		ctx:      ctx,
		dim:      dim,
		throttle: 50 * time.Millisecond, // ~20 FPS to UI
	}
}

func clamp255(x float32) byte {
	if x <= 0 {
		return 0
	}
	if x >= 1 {
		return 255
	}
	return byte(x * 255.0)
}

func (d *Driver) Write(buf []render.Color) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	now := time.Now()
	if d.lastEmit.Add(d.throttle).After(now) {
		return nil // throttle UI updates
	}
	d.lastEmit = now

	rgb := make([]byte, len(buf)*3)
	for i := range buf {
		rgb[i*3+0] = clamp255(buf[i].R)
		rgb[i*3+1] = clamp255(buf[i].G)
		rgb[i*3+2] = clamp255(buf[i].B)
	}
	payload := map[string]any{
		"x": d.dim.X, "y": d.dim.Y, "z": d.dim.Z,
		"rgb": base64.StdEncoding.EncodeToString(rgb),
	}
	runtime.EventsEmit(d.ctx, "preview:frame", payload)
	return nil
}
