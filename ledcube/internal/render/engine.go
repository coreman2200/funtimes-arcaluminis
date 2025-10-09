package render

import (
	"errors"
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

	// Render active
	if e.RActive != nil {
		e.RActive.Render(e.BufA, e.LUT, e.Dim, t, e.UActive, e.Rsrc)
	}

	// Render next if fading
	if e.fading && e.RNext != nil {
		e.RNext.Render(e.BufB, e.LUT, e.Dim, t, e.UNext, e.Rsrc)
		// Mix A/B by alpha into Out
		Mix(e.Out, e.BufA, e.BufB, e.alpha)
	} else {
		copy(e.Out, e.BufA)
	}

	// Post
	postStart := time.Now()
	if e.post.ToneMap != nil {
		e.post.ToneMap(e.Out)
	}
	if e.post.Limiter != nil {
		e.post.Limiter(e.Out, e.UActive)
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
	if e.UNext == nil {
		// clone a shallow copy of uniforms
		e.UNext = &Uniforms{
			GlobalBrightness: e.UActive.GlobalBrightness,
			TimeScale:        e.UActive.TimeScale,
			SunDir:           e.UActive.SunDir,
			MoonDir:          e.UActive.MoonDir,
			Params:           map[string]float64{},
			Bools:            map[string]bool{},
		}
		for k, v := range e.UActive.Params {
			e.UNext.Params[k] = v
		}
		for k, v := range e.UActive.Bools {
			e.UNext.Bools[k] = v
		}
	}
	if preset != "" {
		rr.ApplyPreset(preset, e.UNext)
	}
	e.fading = true
	return nil
}

// SetCrossfade sets mix alpha 0..1 and enables/disables fading.
func (e *Engine) SetCrossfade(alpha float64) {
	if alpha <= 0 {
		e.alpha = 0
		e.fading = false
	} else if alpha >= 1 {
		e.alpha = 1
		e.fading = false
		// promote next -> active
		if e.RNext != nil {
			e.RActive = e.RNext
			e.UActive = e.UNext
		}
		e.RNext = nil
		// leave UNEXT as last copy; caller may reuse
	} else {
		e.alpha = alpha
		e.fading = true
	}
}

// SetParam updates active uniforms.
func (e *Engine) SetParam(name string, v float64) {
	if e.UActive == nil {
		return
	}
	if e.UActive.Params == nil {
		e.UActive.Params = map[string]float64{}
	}
	e.UActive.Params[name] = v
}

// SetBool updates active uniforms.
func (e *Engine) SetBool(name string, b bool) {
	if e.UActive == nil {
		return
	}
	if e.UActive.Bools == nil {
		e.UActive.Bools = map[string]bool{}
	}
	e.UActive.Bools[name] = b
}
