package solid

import (
	"math"

	"github.com/coreman2200/funtimes-arcaluminis/ledcube/internal/render"
)

// Solid is a tiny renderer that fills the cube with a single color.
// It supports presets and an optional "PulseHz" param that modulates brightness.
type Solid struct {
	name string
	c    render.Color
}

func New(name string, c render.Color) *Solid { return &Solid{name: name, c: c} }

func (s *Solid) Name() string { return s.name }

func (s *Solid) Presets() []string { return []string{"Red", "Green", "Blue", "White", "Black"} }

func (s *Solid) ApplyPreset(name string, u *render.Uniforms) {
	switch name {
	case "Red":
		s.c = render.Color{R: 1, G: 0, B: 0}
	case "Green":
		s.c = render.Color{R: 0, G: 1, B: 0}
	case "Blue":
		s.c = render.Color{R: 0, G: 0, B: 1}
	case "White":
		s.c = render.Color{R: 1, G: 1, B: 1}
	case "Black":
		s.c = render.Color{R: 0, G: 0, B: 0}
	}
}

func (s *Solid) Render(dst []render.Color, _ []render.Vec3, _ render.Dimensions, t float64, u *render.Uniforms, _ *render.Resources) {
	// Optional pulse for testing SetParam path
	scale := float32(1.0)
	if u != nil && u.Params != nil {
		if hz, ok := u.Params["PulseHz"]; ok && hz > 0 {
			scale = float32(0.5 + 0.5*math.Sin(2*math.Pi*hz*t))
		}
	}
	c := render.Color{R: s.c.R * scale, G: s.c.G * scale, B: s.c.B * scale}
	for i := range dst {
		dst[i] = c
	}
}
