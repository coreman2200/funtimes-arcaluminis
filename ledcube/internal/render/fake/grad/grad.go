package grad

import (
	"math"

	"github.com/coreman2200/funtimes-arcaluminis/ledcube/internal/render"
)

// Grad renders a simple spatial gradient with optional time animation.
// Params:
//   - "Speed" (float, default 0): animates hue over time
//   - "Axis"  (0=X,1=Y,2=Z; default 2)
type Grad struct {
	name string
}

func New(name string) *Grad { return &Grad{name: name} }

func (g *Grad) Name() string { return g.name }

func (g *Grad) Presets() []string { return []string{"XZ", "XY", "YZ", "Rainbow"} }

func (g *Grad) ApplyPreset(name string, u *render.Uniforms) {
	if u == nil {
		return
	}
	if u.Params == nil {
		u.Params = map[string]float64{}
	}
	switch name {
	case "XZ":
		u.Params["Axis"] = 1 // vary over XZ -> use Y as brightness (so axis=1 fixes Y)
	case "XY":
		u.Params["Axis"] = 2
	case "YZ":
		u.Params["Axis"] = 0
	case "Rainbow":
		u.Params["Speed"] = 0.1
	}
}

func (g *Grad) Render(dst []render.Color, pLUT []render.Vec3, _ render.Dimensions, t float64, u *render.Uniforms, _ *render.Resources) {
	axis := 2.0
	speed := 0.0
	if u != nil && u.Params != nil {
		if v, ok := u.Params["Axis"]; ok {
			axis = v
		}
		if v, ok := u.Params["Speed"]; ok {
			speed = v
		}
	}
	a := int(axis)
	for i := range dst {
		p := pLUT[i]
		var v float64
		switch a {
		case 0:
			v = p.X
		case 1:
			v = p.Y
		default:
			v = p.Z
		}
		// simple hue-ish rotation
		phase := v*2*math.Pi + t*2*math.Pi*speed
		dst[i] = render.Color{
			R: float32(0.5 + 0.5*math.Sin(phase)),
			G: float32(0.5 + 0.5*math.Sin(phase+2*math.Pi/3)),
			B: float32(0.5 + 0.5*math.Sin(phase+4*math.Pi/3)),
		}
	}
}
