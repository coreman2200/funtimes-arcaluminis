package main

import (
	"context"
	"fmt"
	"time"

	"internal/render"
	"internal/sequence"
	"internal/led"

	// fake pieces
	"internal/driver/fake"
	"internal/render/fake/solid"
	"internal/render/fake/grad"
)

// Minimal conductor inline for the sim.
type Conductor struct {
	Eng *render.Engine
	Reg *render.Registry
	Seq *sequence.Player
	cancel context.CancelFunc
}

func NewConductor(eng *render.Engine, reg *render.Registry) *Conductor {
	c := &Conductor{Eng: eng, Reg: reg}
	hooks := sequence.Hooks{
		SetRenderer: func(name, preset string) { _ = eng.SetRenderer(name, preset, reg); fmt.Println("SetRenderer:", name, preset) },
		ArmNext:     func(name, preset string) { _ = eng.ArmNext(name, preset, reg);     fmt.Println("ArmNext:", name, preset) },
		SetCrossfade: func(a float64) { eng.SetCrossfade(a); fmt.Printf("Alpha: %.2f\n", a) },
		SetParam:     eng.SetParam,
		SetBool:      eng.SetBool,
	}
	c.Seq = sequence.NewPlayer(hooks)
	return c
}

func (c *Conductor) Run(ctx context.Context, fps int) {
	if fps <= 0 { fps = 30 }
	tick := time.NewTicker(time.Second / time.Duration(fps))
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
			c.Seq.Tick(1.0/float64(fps))
			_ = c.Eng.RenderOnce(-1)
		}
	}
}

func main() {
	// Registry with fake renderers
	reg := render.NewRegistry()
	reg.Register(solid.New("solid", render.Color{R:1, G:0, B:0}))
	reg.Register(grad.New("grad"))

	// Fake hardware config
	dim := render.Dimensions{X: 5, Y: 5, Z: 5}
	lut := led.BuildLUT(dim, led.Order{}, 0, 0)
	drv := &fake.Driver{}

	// Engine + post
	u := &render.Uniforms{GlobalBrightness: 1, TimeScale: 1, Params: map[string]float64{"OutputGamma":2.2, "ExposureEV":0}, Bools: map[string]bool{}}
	eng, _ := render.NewEngine(dim, lut, drv, mustGet(reg, "solid"), u, &render.Resources{})
	eng.UseFilmicPost()

	// Conductor
	c := NewConductor(eng, reg)

	// Program: solid red -> grad rainbow, crossfade
	prog := sequence.Program{
		Version: "seq.v1",
		Loop:    true,
		Clips: []sequence.Clip{
			{Name:"Red",  Renderer:"solid", Preset:"Red", DurationS:3, XFadeS:1},
			{Name:"Grad", Renderer:"grad",  Preset:"Rainbow", DurationS:3, XFadeS:1},
		},
	}
	_ = c.Seq.Load(prog)
	c.Seq.Start()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go c.Run(ctx, 30)

	// Let it run for 12 seconds
	time.Sleep(12 * time.Second)
}

func mustGet(reg *render.Registry, name string) render.Renderer {
	rr, ok := reg.Get(name)
	if !ok { panic("renderer not in registry: " + name) }
	return rr
}
