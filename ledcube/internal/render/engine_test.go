package render

import "testing"

// fakeRenderer writes a constant color for testing.
type fakeRenderer struct {
	name    string
	r, g, b float32
}

func (f *fakeRenderer) Name() string                         { return f.name }
func (f *fakeRenderer) Presets() []string                    { return []string{"default"} }
func (f *fakeRenderer) ApplyPreset(name string, u *Uniforms) {}
func (f *fakeRenderer) Render(dst []Color, pLUT []Vec3, dim Dimensions, t float64, u *Uniforms, r *Resources) {
	for i := range dst {
		dst[i] = Color{f.r, f.g, f.b}
	}
}

// fakeDriver captures the last frame written.
type fakeDriver struct {
	last []Color
}

func (d *fakeDriver) Write(buf []Color) error {
	d.last = make([]Color, len(buf))
	copy(d.last, buf)
	return nil
}

func TestMixAlpha(t *testing.T) {
	n := 10
	a := make([]Color, n)
	b := make([]Color, n)
	dst := make([]Color, n)
	for i := 0; i < n; i++ {
		a[i] = Color{1, 0, 0} // red
		b[i] = Color{0, 0, 1} // blue
	}
	Mix(dst, a, b, 0.5)
	if dst[0].R < 0.49 || dst[0].R > 0.51 || dst[0].B < 0.49 || dst[0].B > 0.51 {
		t.Fatalf("expected ~purple at alpha=0.5, got %#v", dst[0])
	}
}

func TestEngineRenderOnceAndCrossfade(t *testing.T) {
	dim := Dimensions{X: 1, Y: 1, Z: 1}
	lut := []Vec3{{0.5, 0.5, 0.5}}
	drv := &fakeDriver{}
	reg := NewRegistry()
	ra := &fakeRenderer{name: "A", r: 1, g: 0, b: 0}
	rb := &fakeRenderer{name: "B", r: 0, g: 0, b: 1}
	reg.Register(ra)
	reg.Register(rb)

	u := &Uniforms{GlobalBrightness: 1.0, TimeScale: 1.0, Params: map[string]float64{}, Bools: map[string]bool{}}
	e, err := NewEngine(dim, lut, drv, ra, u, &Resources{})
	if err != nil {
		t.Fatalf("engine: %v", err)
	}
	// Disable tone mapping for deterministic tests.
	e.SetPost(PostPipeline{})

	// Active A, render once
	if err := e.RenderOnce(-1); err != nil {
		t.Fatalf("render: %v", err)
	}
	if drv.last[0].R < 0.99 || drv.last[0].B > 0.01 {
		t.Fatalf("expected red frame, got %#v", drv.last[0])
	}

	// Arm B and fade to 50%
	if err := e.ArmNext("B", "default", reg); err != nil {
		t.Fatalf("arm: %v", err)
	}
	e.SetCrossfade(0.5)
	if err := e.RenderOnce(-1); err != nil {
		t.Fatalf("render 2: %v", err)
	}
	if drv.last[0].R < 0.49 || drv.last[0].R > 0.51 || drv.last[0].B < 0.49 || drv.last[0].B > 0.51 {
		t.Fatalf("expected purple during fade, got %#v", drv.last[0])
	}

	// Complete fade
	e.SetCrossfade(1.0)
	if err := e.RenderOnce(-1); err != nil {
		t.Fatalf("render 3: %v", err)
	}
	if drv.last[0].B < 0.99 || drv.last[0].R > 0.01 {
		t.Fatalf("expected blue frame after complete fade, got %#v", drv.last[0])
	}
}
