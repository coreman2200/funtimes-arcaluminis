package sequence

import (
	"errors"
	"math"
	"sync"
)

// NewPlayer constructs a Player with provided hooks.
func NewPlayer(h Hooks) *Player {
	return &Player{
		State: Idle,
		hooks: h,
		armedIndex: -1,
	}
}

// Load replaces the current program. Resets time and state to Idle.
func (p *Player) Load(prog Program) error {
	if len(prog.Clips) == 0 {
		return errors.New("program has no clips")
	}
	p.prog = prog
	p.nowS = 0
	p.idx = 0
	p.State = Idle
	p.armed = false
	p.armedIndex = -1
	p.lastAlpha = 0
	return nil
}

// Start moves to Running and primes the first clip.
func (p *Player) Start() {
	if p.State == Running {
		return
	}
	p.State = Running
	clip := p.prog.Clips[p.idx]
	if p.hooks.SetRenderer != nil {
		p.hooks.SetRenderer(clip.Renderer, clip.Preset)
	}
	// reset fade
	if p.hooks.SetCrossfade != nil {
		p.hooks.SetCrossfade(0)
	}
}

// Pause pauses playback.
func (p *Player) Pause() { p.State = Paused }
// Resume resumes playback.
func (p *Player) Resume() { if p.State == Paused { p.State = Running } }
// Stop stops and resets to start.
func (p *Player) Stop() {
	p.State = Idle
	p.nowS = 0
	p.idx = 0
	p.armed = false
	p.armedIndex = -1
	if p.hooks.SetCrossfade != nil {
		p.hooks.SetCrossfade(0)
	}
}

// Seek jumps to absolute program time t. Clamps into [0, totalDur).
func (p *Player) Seek(t float64) {
	if len(p.prog.Clips) == 0 {
		return
	}
	if t < 0 {
		t = 0
	}
	total := p.totalDuration()
	if total > 0 && t >= total {
		// Clamp to just before end
		t = math.Nextafter(total, -1)
	}
	// Find clip index and local time
	acc := 0.0
	idx := 0
	for i, c := range p.prog.Clips {
		if t < acc + c.DurationS {
			idx = i
			break
		}
		acc += c.DurationS
	}
	p.idx = idx
	p.nowS = t
	p.armed = false
	p.armedIndex = -1
	// Switch renderer to this clip
	clip := p.prog.Clips[p.idx]
	if p.hooks.SetRenderer != nil {
		p.hooks.SetRenderer(clip.Renderer, clip.Preset)
	}
	if p.hooks.SetCrossfade != nil {
		p.hooks.SetCrossfade(0)
	}
}

// Tick advances the sequencer by dt seconds and emits control hooks.
func (p *Player) Tick(dt float64) {
	if p.State != Running || len(p.prog.Clips) == 0 {
		return
	}
	if dt <= 0 {
		return
	}
	p.nowS += dt

	clip, localT := p.currentClipAndLocalT()
	// Evaluate params/bools for the active clip
	for name, env := range clip.Params {
		if p.hooks.SetParam != nil {
			p.hooks.SetParam(name, env.Eval(localT))
		}
	}
	for name, env := range clip.Bools {
		if p.hooks.SetBool != nil {
			p.hooks.SetBool(name, env.BoolEval(localT))
		}
	}
	// Crossfade logic
	if clip.XFadeS > 0 {
		remain := clip.DurationS - localT
		if remain <= clip.XFadeS && remain >= 0 {
			// Arm next once
			nextIdx := p.nextIndex()
			if !p.armed && nextIdx != -1 && p.hooks.ArmNext != nil {
				nc := p.prog.Clips[nextIdx]
				p.hooks.ArmNext(nc.Renderer, nc.Preset)
				p.armed = true
				p.armedIndex = nextIdx
			}
			// Alpha 0..1 over [Duration-XFade, Duration]
			alpha := 1.0 - (remain / clip.XFadeS)
			if alpha < 0 {
				alpha = 0
			}
			if alpha > 1 {
				alpha = 1
			}
			if p.hooks.SetCrossfade != nil && alpha != p.lastAlpha {
				p.hooks.SetCrossfade(alpha)
				p.lastAlpha = alpha
			}
		}
	}

	// Clip end?
	if localT >= clip.DurationS {
		p.advanceClip()
	}
}

func (p *Player) currentClipAndLocalT() (Clip, float64) {
	acc := 0.0
	for i := 0; i < p.idx; i++ {
		acc += p.prog.Clips[i].DurationS
	}
	localT := p.nowS - acc
	return p.prog.Clips[p.idx], localT
}

func (p *Player) totalDuration() float64 {
	total := 0.0
	for _, c := range p.prog.Clips {
		total += c.DurationS
	}
	return total
}

func (p *Player) nextIndex() int {
	if len(p.prog.Clips) == 0 {
		return -1
	}
	ni := p.idx + 1
	if ni >= len(p.prog.Clips) {
		if p.prog.Loop {
			return 0
		}
		return -1
	}
	return ni
}

func (p *Player) advanceClip() {
	next := p.nextIndex()
	if next == -1 {
		// End of program
		p.State = Idle
		if p.hooks.SetCrossfade != nil {
			p.hooks.SetCrossfade(0)
		}
		return
	}
	p.idx = next
	// Snap renderer to next and reset crossfade
	clip := p.prog.Clips[p.idx]
	if p.hooks.SetRenderer != nil {
		p.hooks.SetRenderer(clip.Renderer, clip.Preset)
	}
	if p.hooks.SetCrossfade != nil {
		p.hooks.SetCrossfade(0)
	}
	p.armed = false
	p.armedIndex = -1
	p.lastAlpha = 0
}

// --- Lightweight synchronization helpers (optional) ---

type SafePlayer struct {
	mu sync.Mutex
	P  *Player
}

func NewSafePlayer(h Hooks) *SafePlayer {
	return &SafePlayer{P: NewPlayer(h)}
}

func (s *SafePlayer) With(f func(p *Player)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	f(s.P)
}
