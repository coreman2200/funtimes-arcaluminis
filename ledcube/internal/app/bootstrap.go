package app

import (
	"context"
	"fmt"
	"time"

	"github.com/coreman2200/funtimes-arcaluminis/ledcube/internal/led"
	"github.com/coreman2200/funtimes-arcaluminis/ledcube/internal/render"
	"github.com/coreman2200/funtimes-arcaluminis/ledcube/internal/sequence"
)

type Core struct {
	Eng    *render.Engine
	Reg    *render.Registry
	Seq    *sequence.Player
	cancel context.CancelFunc
}

type HWConfig struct {
	Dim     render.Dimensions
	Order   led.Order
	PitchMM float64
	GapMM   float64
	Drv     render.Driver
}

func applyPostDefaults(eng *render.Engine) {
	for k, v := range map[string]float64{
		"Budget_mA":   3000,
		"LEDChan_mA":  20,
		"LimiterKnee": 0.9,
		"WhiteCap":    2.2,
		"ExposureEV":  0,
		"OutputGamma": 2.2,
	} {
		eng.SetParam(k, v)
	}
}

func InitCore(ctx context.Context, hw HWConfig, startRenderer string, uniforms *render.Uniforms, resources *render.Resources) (*Core, error) {
	// 1) Registry (register your real renderers elsewhere & import here)
	reg := render.NewRegistry()
	// reg.Register(ocean.New())
	// reg.Register(warp.New())
	rr, ok := reg.Get(startRenderer)
	if !ok {
		// Fallback to any renderer in the registry
		names := reg.List()
		if len(names) == 0 {
			return nil, fmt.Errorf("no renderers registered")
		}
		rr, _ = reg.Get(names[0])
	}

	// 2) LUT from your physical layout
	lut := led.BuildLUT(hw.Dim, hw.Order, hw.PitchMM, hw.GapMM)

	// 3) Engine
	eng, err := render.NewEngine(hw.Dim, lut, hw.Drv, rr, uniforms, resources)
	if err != nil {
		return nil, err
	}

	// 4) Post pipeline (filmic + limiter) — see next section
	eng.UseFilmicPost() // <— tiny helper (shown below)
	// Safer than touching the map directly:
	applyPostDefaults(eng)

	// 5) Sequencer wiring (hooks → engine)
	hooks := sequence.Hooks{
		SetRenderer:  func(name, preset string) { _ = eng.SetRenderer(name, preset, reg) },
		ArmNext:      func(name, preset string) { _ = eng.ArmNext(name, preset, reg) },
		SetCrossfade: func(a float64) { eng.SetCrossfade(a) },
		SetParam:     eng.SetParam,
		SetBool:      eng.SetBool,
	}
	seq := sequence.NewPlayer(hooks)

	// 6) Frame/timeline loop
	ctx, cancel := context.WithCancel(ctx)
	go func() {
		tick := time.NewTicker(time.Second / 60)
		defer tick.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-tick.C:
				seq.Tick(1.0 / 60.0)
				_ = eng.RenderOnce(-1) // uses eng.Now()
			}
		}
	}()

	return &Core{Eng: eng, Reg: reg, Seq: seq, cancel: cancel}, nil
}
