// internal/app/conductor.go
package app

import (
	"time"

	"github.com/coreman2200/funtimes-arcaluminis/ledcube/internal/render"
	"github.com/coreman2200/funtimes-arcaluminis/ledcube/internal/sequence"
)

type Conductor struct {
	Eng *render.Engine
	Reg *render.Registry
	Seq *sequence.Player
}

func NewConductor(eng *render.Engine, reg *render.Registry) *Conductor {
	c := &Conductor{Eng: eng, Reg: reg}

	hooks := sequence.Hooks{
		SetRenderer:  func(name, preset string) { _ = eng.SetRenderer(name, preset, reg) },
		ArmNext:      func(name, preset string) { _ = eng.ArmNext(name, preset, reg) },
		SetCrossfade: func(a float64) { eng.SetCrossfade(a) },
		SetParam:     func(k string, v float64) { eng.SetParam(k, v) },
		SetBool:      func(k string, b bool) { eng.SetBool(k, b) },
	}
	c.Seq = sequence.NewPlayer(hooks)
	return c
}

func (c *Conductor) Run(fps int) {
	if fps <= 0 {
		fps = 60
	}
	dt := time.Second / time.Duration(fps)
	ticker := time.NewTicker(dt)
	defer ticker.Stop()
	for range ticker.C {
		c.Seq.Tick(dt.Seconds()) // drive timeline & crossfades
		_ = c.Eng.RenderOnce(-1) // render current frame (uses Eng.Now())
	}
}
