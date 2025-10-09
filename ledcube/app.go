package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/coreman2200/funtimes-arcaluminis/ledcube/internal/app"
	"github.com/coreman2200/funtimes-arcaluminis/ledcube/internal/driver/preview"
	"github.com/coreman2200/funtimes-arcaluminis/ledcube/internal/render"
	"github.com/coreman2200/funtimes-arcaluminis/ledcube/internal/sequence"

	grad "github.com/coreman2200/funtimes-arcaluminis/ledcube/internal/render/fake/grad"
	solid "github.com/coreman2200/funtimes-arcaluminis/ledcube/internal/render/fake/solid"
)

type App struct{ core *app.Core }

func NewApp() *App { return &App{} }

// === Wails-callable methods ===

func (a *App) ListRenderers() []string {
	if a.core == nil || a.core.Reg == nil {
		return nil
	}
	return a.core.Reg.List()
}

func (a *App) SetRenderer(name, preset string) error {
	log.Printf("SetRenderer(%s,%s)\n", name, preset)
	if a.core == nil {
		return fmt.Errorf("core not ready")
	}
	return a.core.Eng.SetRenderer(name, preset, a.core.Reg)
}

func (a *App) ArmNext(name, preset string) error {
	log.Printf("ArmNext(%s,%s)\n", name, preset)
	if a.core == nil {
		return fmt.Errorf("core not ready")
	}
	return a.core.Eng.ArmNext(name, preset, a.core.Reg)
}

func (a *App) SeqCmd(cmd string) error {
	log.Println("SeqCmd:", cmd)
	if a.core == nil {
		return fmt.Errorf("core not ready")
	}
	switch cmd {
	case "start":
		a.core.Seq.Start()
	case "stop":
		a.core.Seq.Stop()
	case "pause":
		a.core.Seq.Pause()
	case "resume":
		a.core.Seq.Resume()
	default:
		return fmt.Errorf("unknown cmd: %s", cmd)
	}
	return nil
}

func (a *App) LoadProgram(jsonStr string) error {
	log.Println("LoadProgram JSON len:", len(jsonStr))
	if a.core == nil {
		return fmt.Errorf("core not ready")
	}
	var p sequence.Program
	if err := json.Unmarshal([]byte(jsonStr), &p); err != nil {
		return err
	}
	return a.core.Seq.Load(p)
}

func (a *App) SetParam(key string, val float64) {
	log.Printf("SetParam(%s=%v)\n", key, val)
	if a.core != nil {
		a.core.Eng.SetParam(key, val)
	}
}

func (a *App) SetBool(key string, val bool) {
	log.Printf("SetBool(%s=%v)\n", key, val)
	if a.core != nil {
		a.core.Eng.SetBool(key, val)
	}
}

// Convenience for your “Tests” menu
func (a *App) RunTest(name string) (string, error) {
	log.Println("RunTest:", name)
	if a.core == nil {
		return "", fmt.Errorf("core not ready")
	}
	switch name {
	case "SolidRed":
		_ = a.core.Eng.SetRenderer("solid", "Red", a.core.Reg)
	case "SolidWhite":
		_ = a.core.Eng.SetRenderer("solid", "White", a.core.Reg)
	case "GradRainbow":
		_ = a.core.Eng.SetRenderer("grad", "Rainbow", a.core.Reg)
	case "ProgramDemo":
		_ = a.core.Seq.Load(sequence.Program{
			Version: "seq.v1", Loop: true,
			Clips: []sequence.Clip{
				{Name: "Red", Renderer: "solid", Preset: "Red", DurationS: 3, XFadeS: 1},
				{Name: "Grad", Renderer: "grad", Preset: "Rainbow", DurationS: 3, XFadeS: 1},
			},
		})
		a.core.Seq.Start()
	default:
		return "", fmt.Errorf("unknown test: %s", name)
	}
	return "ok", nil
}

func (a *App) startup(ctx context.Context) {
	// Dimensions: adjust to match your cube
	dim := render.Dimensions{X: 5, Y: 26, Z: 5}

	// Preview driver for desktop
	drv := preview.New(ctx, dim)

	uniforms := &render.Uniforms{
		GlobalBrightness: 0.8,
		TimeScale:        1.0,
		Params: map[string]float64{
			"ExposureEV": 0, "OutputGamma": 2.2,
			"LEDChan_mA": 20, "Budget_mA": 3000,
			"LimiterKnee": 0.9, "WhiteCap": 2.2,
		},
		Bools: map[string]bool{},
	}

	registrar := func(reg *render.Registry) {
		reg.Register(solid.New("solid", render.Color{R: 1})) // 🔴 solid red
		reg.Register(grad.New("grad"))                       // 🌈 gradient
	}

	core, err := app.InitCore(ctx, app.HWConfig{
		Dim: dim,
		Drv: drv, // 👈 no LEDs required
		// TODO: wire your Order/Pitch/Gap as needed for BuildLUT
	}, "solid", uniforms, &render.Resources{}, registrar)
	if err != nil {
		panic(err)
	}
	a.core = core

	// quick visual program using your fake renderers
	_ = a.core.Seq.Load(sequence.Program{
		Version: "seq.v1", Loop: true,
		Clips: []sequence.Clip{
			{Name: "Red", Renderer: "solid", Preset: "Red", DurationS: 3, XFadeS: 1},
			{Name: "Grad", Renderer: "grad", Preset: "Rainbow", DurationS: 3, XFadeS: 1},
		},
	})
	a.core.Seq.Start()
}

func (a *App) shutdown(ctx context.Context) {
	if a.core != nil {
		a.core.Seq.Stop()
	}
}
