package render

import (
	"errors"
	"sync"
	"time"
)

// Driver abstracts the LED transport (SPI, etc.).
type Driver interface {
	Write([]Color) error
}

// Engine renders frames using an active Renderer, optional next Renderer for crossfades,
// applies post-processing, then writes to the driver.
type Engine struct {
	Dim  Dimensions
	LUT  []Vec3
	Drv  Driver
	Rsrc *Resources
	mu   sync.RWMutex

	// active + next renderer and uniforms
	RActive Renderer
	RNext   Renderer
	UActive *Uniforms
	UNext   *Uniforms

	// framebuffers
	BufA []Color // active
	BufB []Color // next (during crossfade)
	Out  []Color // mixed + post

	// crossfade
	alpha  float64 // 0..1
	fading bool

	// timing
	t0 time.Time

	// post
	post PostPipeline

	// metrics (last durations in ms)
	Last struct {
		RenderMS float64
		PostMS   float64
		TotalMS  float64
	}
}

// PostPipeline groups post stages; all are optional.
type PostPipeline struct {
	ToneMap func([]Color)
	Limiter func([]Color, *Uniforms)
}

func (e *Engine) SnapshotUniforms() *Uniforms { return e.snapshotActive() }

// NewEngine allocates buffers and returns an Engine with defaults wired.
func NewEngine(dim Dimensions, lut []Vec3, drv Driver, r Renderer, u *Uniforms, rsrc *Resources) (*Engine, error) {
	if dim.X*dim.Y*dim.Z == 0 {
		return nil, errors.New("invalid dimensions")
	}
	n := dim.X * dim.Y * dim.Z
	e := &Engine{
		Dim:     dim,
		LUT:     lut,
		Drv:     drv,
		Rsrc:    rsrc,
		RActive: r,
		UActive: u,
		BufA:    make([]Color, n),
		BufB:    make([]Color, n),
		Out:     make([]Color, n),
		alpha:   0,
		fading:  false,
		post: PostPipeline{
			ToneMap: DefaultToneMap,
			Limiter: DefaultLimiter,
		},
		t0: time.Now(),
	}
	return e, nil
}

// Now returns seconds since engine start, scaled by TimeScale.
func (e *Engine) Now() float64 {
	scale := 1.0
	if e.UActive != nil && e.UActive.TimeScale != 0 {
		scale = e.UActive.TimeScale
	}
	return time.Since(e.t0).Seconds() * scale
}

// RenderOnce renders a single frame at absolute time t (seconds).
// If t < 0, it uses Engine.Now().
func (e *Engine) RenderOnce(t float64) error {
	if t < 0 {
		t = e.Now()
	}
	start := time.Now()

	// Read-only uniforms for this frame
	uA := e.snapshotActive()
	uN := e.snapshotNext()

	// Render active
	// --- Render & mix ---
	if e.RActive != nil {
		e.RActive.Render(e.BufA, e.LUT, e.Dim, t, uA, e.Rsrc)
	}

	if e.RNext != nil {
		// Only render B if we actually need it (alpha in (0,1))
		if e.alpha > 0 && e.alpha < 1 {
			e.RNext.Render(e.BufB, e.LUT, e.Dim, t, uN, e.Rsrc)
			Mix(e.Out, e.BufA, e.BufB, e.alpha)
		} else if e.alpha >= 1 {
			// promote B -> A immediately at frame boundary
			e.RActive = e.RNext
			e.RNext = nil
			e.fading = false
			e.alpha = 0
			copy(e.Out, e.BufA) // will be overwritten next frame by new A
			// clear BufB to avoid ghosts
			for i := range e.BufB {
				e.BufB[i] = Color{}
			}
		} else {
			// alpha == 0: ignore B
			copy(e.Out, e.BufA)
		}
	} else {
		// no B armed, render A only
		copy(e.Out, e.BufA)
	}

	// --- Post ---
	postStart := time.Now()
	if e.post.ToneMap != nil {
		e.post.ToneMap(e.Out) // exposure+tonemap+gamma for preview path
	}
	preview := false
	if uA != nil && uA.Params != nil {
		if uA.Params["PreviewMode"] > 0.5 || uA.Params["PreviewBypass"] > 0.5 {
			preview = true
		}
	}
	if !preview && e.post.Limiter != nil {
		e.post.Limiter(e.Out, uA)
	}
	e.Last.PostMS = float64(time.Since(postStart).Microseconds()) / 1000.0

	// Write
	if e.Drv != nil {
		if err := e.Drv.Write(e.Out); err != nil {
			return err
		}
	}

	// Metrics
	e.Last.RenderMS = float64(time.Since(start).Microseconds()) / 1000.0
	e.Last.TotalMS = e.Last.RenderMS
	return nil
}

func (e *Engine) UseFilmicPost() {
	e.SetPost(PostPipeline{
		ToneMap: func(buf []Color) { FilmicToneMap(buf, e.UActive) },
		Limiter: DefaultLimiter,
	})
}

func (e *Engine) SetPost(p PostPipeline) { e.post = p }

// snapshotActive copies UActive (maps included) so renderers read a stable view
func (e *Engine) snapshotActive() *Uniforms {
	e.mu.RLock()
	defer e.mu.RUnlock()
	if e.UActive == nil {
		return &Uniforms{Params: map[string]float64{}, Bools: map[string]bool{}}
	}
	out := &Uniforms{
		GlobalBrightness: e.UActive.GlobalBrightness,
		TimeScale:        e.UActive.TimeScale,
		SunDir:           e.UActive.SunDir,
		MoonDir:          e.UActive.MoonDir,
		Params:           map[string]float64{},
		Bools:            map[string]bool{},
	}
	for k, v := range e.UActive.Params {
		out.Params[k] = v
	}
	for k, v := range e.UActive.Bools {
		out.Bools[k] = v
	}
	return out
}

func (e *Engine) snapshotNext() *Uniforms {
	e.mu.RLock()
	defer e.mu.RUnlock()
	if e.UNext == nil {
		return &Uniforms{Params: map[string]float64{}, Bools: map[string]bool{}}
	}
	out := &Uniforms{
		GlobalBrightness: e.UNext.GlobalBrightness,
		TimeScale:        e.UNext.TimeScale,
		SunDir:           e.UNext.SunDir,
		MoonDir:          e.UNext.MoonDir,
		Params:           map[string]float64{},
		Bools:            map[string]bool{},
	}
	for k, v := range e.UNext.Params {
		out.Params[k] = v
	}
	for k, v := range e.UNext.Bools {
		out.Bools[k] = v
	}
	return out
}

// ---- Hooks that match Sequencer expectations ----

// SetRenderer becomes the active renderer immediately.
// If preset != "", ApplyPreset is called on the renderer with UActive.
func (e *Engine) SetRenderer(name string, preset string, reg *Registry) error {
	if reg == nil {
		return errors.New("registry is nil")
	}
	rr, ok := reg.Get(name)
	if !ok {
		return errors.New("renderer not found: " + name)
	}
	e.RActive = rr
	if preset != "" {
		rr.ApplyPreset(preset, e.UActive)
	}
	// reset fade
	e.fading = false
	e.alpha = 0
	return nil
}

// ArmNext prepares the next renderer for crossfade.
func (e *Engine) ArmNext(name string, preset string, reg *Registry) error {
	if reg == nil {
		return errors.New("registry is nil")
	}
	rr, ok := reg.Get(name)
	if !ok {
		return errors.New("renderer not found: " + name)
	}
	e.RNext = rr

	e.mu.RLock()
	ua := e.UActive
	e.mu.RUnlock()

	if e.UNext == nil {
		e.UNext = &Uniforms{
			GlobalBrightness: ua.GlobalBrightness,
			TimeScale:        ua.TimeScale,
			SunDir:           ua.SunDir,
			MoonDir:          ua.MoonDir,
			Params:           map[string]float64{},
			Bools:            map[string]bool{},
		}
		if ua.Params != nil {
			for k, v := range ua.Params {
				e.UNext.Params[k] = v
			}
		}
		if ua.Bools != nil {
			for k, v := range ua.Bools {
				e.UNext.Bools[k] = v
			}
		}
	}
	if preset != "" {
		rr.ApplyPreset(preset, e.UNext)
	}
	e.fading = true
	return nil
}

// SetCrossfade sets mix alpha 0..1 and enables/disables fading.
// SetCrossfade clamps alpha and toggles fading appropriately.
// If there is no RNext armed, alpha changes are ignored.
func (e *Engine) SetCrossfade(a float64) {
	if a < 0 {
		a = 0
	} else if a > 1 {
		a = 1
	}
	// no next renderer => nothing to fade to; ignore
	if e.RNext == nil {
		e.fading = false
		e.alpha = 0
		return
	}
	e.alpha = a
	// only mix when we truly are in-between
	e.fading = (a > 0 && a < 1)
}

// SetParam updates active uniforms.
func (e *Engine) SetParam(name string, v float64) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.UActive == nil {
		e.UActive = &Uniforms{}
	}
	if e.UActive.Params == nil {
		e.UActive.Params = map[string]float64{}
	}
	e.UActive.Params[name] = v
}

func (e *Engine) SetBool(name string, b bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.UActive == nil {
		e.UActive = &Uniforms{}
	}
	if e.UActive.Bools == nil {
		e.UActive.Bools = map[string]bool{}
	}
	e.UActive.Bools[name] = b
}
