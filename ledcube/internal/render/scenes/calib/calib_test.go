package calib

import (
	"fmt"
	"testing"

	"github.com/coreman2200/funtimes-arcaluminis/ledcube/internal/render"
)

func colorAt(buf []render.Color, dim render.Dimensions, x, y, z int) render.Color {
	idx := z*int(dim.Y)*int(dim.X) + y*int(dim.X) + x
	return buf[idx]
}

func TestPanelChanSweepSceneLinear(t *testing.T) {
	dim := render.Dimensions{X: 5, Y: 5, Z: 3}
	dst := make([]render.Color, dim.X*dim.Y*dim.Z)
	r := New("calib")
	u := &render.Uniforms{Params: map[string]float64{}, Bools: map[string]bool{}}

	r.ApplyPreset("PanelChanSweep", u)
	r.Render(dst, nil, dim, 0, u, nil)

	bottomRed := colorAt(dst, dim, 0, 0, 0)
	topRed := colorAt(dst, dim, 0, int(dim.Y)-1, 0)
	bottomGreen := colorAt(dst, dim, 0, 0, 1)
	bottomBlue := colorAt(dst, dim, 0, 0, 2)

	t.Logf("scene-linear bottom front panel (red): %+v", bottomRed)
	t.Logf("scene-linear top front panel (should trend white): %+v", topRed)
	t.Logf("scene-linear bottom second panel (green): %+v", bottomGreen)
	t.Logf("scene-linear bottom third panel (blue): %+v", bottomBlue)

	if bottomRed.R < 0.3 {
		t.Fatalf("expected visible red at (0,0,0); got %+v", bottomRed)
	}
	if bottomGreen.G < 0.3 {
		t.Fatalf("expected visible green at (0,0,1); got %+v", bottomGreen)
	}
	if bottomBlue.B < 0.3 {
		t.Fatalf("expected visible blue at (0,0,2); got %+v", bottomBlue)
	}
}

func TestPanelChanSweepPreviewPost(t *testing.T) {
	dim := render.Dimensions{X: 5, Y: 5, Z: 3}
	dst := make([]render.Color, dim.X*dim.Y*dim.Z)
	r := New("calib")
	u := &render.Uniforms{Params: map[string]float64{
		"PreviewMode": 1,
		"ExposureEV":  0,
		"OutputGamma": 2.2,
	}, Bools: map[string]bool{}}

	r.ApplyPreset("PanelChanSweep", u)
	r.Render(dst, nil, dim, 0, u, nil)

	post := make([]render.Color, len(dst))
	copy(post, dst)
	render.FilmicToneMap(post, u)

	bottomRed := colorAt(post, dim, 0, 0, 0)
	bottomGreen := colorAt(post, dim, 0, 0, 1)
	bottomBlue := colorAt(post, dim, 0, 0, 2)

	t.Logf("post-tonemap bottom front panel (red): %+v", bottomRed)
	t.Logf("post-tonemap bottom second panel (green): %+v", bottomGreen)
	t.Logf("post-tonemap bottom third panel (blue): %+v", bottomBlue)

	if bottomRed.R < 0.3 {
		t.Fatalf("expected red dominance after post; got %+v", bottomRed)
	}
	if bottomGreen.G < 0.3 {
		t.Fatalf("expected green dominance after post; got %+v", bottomGreen)
	}
	if bottomBlue.B < 0.3 {
		t.Fatalf("expected blue dominance after post; got %+v", bottomBlue)
	}
}

func TestPanelChanSweepColorAtlas(t *testing.T) {
	dim := render.Dimensions{X: 5, Y: 5, Z: 3}
	linear := make([]render.Color, dim.X*dim.Y*dim.Z)
	r := New("calib")
	u := &render.Uniforms{Params: map[string]float64{"PreviewMode": 1, "ExposureEV": 0, "OutputGamma": 2.2}, Bools: map[string]bool{}}
	r.ApplyPreset("PanelChanSweep", u)
	r.Render(linear, nil, dim, 0, u, nil)

	preview := make([]render.Color, len(linear))
	copy(preview, linear)
	render.FilmicToneMap(preview, u)

	// Log and verify monotonic fade left->right on bottom row
	prevLin := float32(2)
	prevPre := float32(2)
	var diffLin []float32
	var diffPre []float32
	for x := 0; x < int(dim.X); x++ {
		lin := colorAt(linear, dim, x, 0, 0)
		pre := colorAt(preview, dim, x, 0, 0)
		t.Logf("bottom row x=%d linear=%+v preview=%+v", x, lin, pre)
		if lin.R > prevLin+1e-4 {
			t.Fatalf("bottom row linear not monotonic at x=%d: %.4f -> %.4f", x, prevLin, lin.R)
		}
		if pre.R > prevPre+1e-4 {
			t.Fatalf("bottom row preview not monotonic at x=%d: %.4f -> %.4f", x, prevPre, pre.R)
		}
		if x > 0 {
			diffLin = append(diffLin, prevLin-lin.R)
			diffPre = append(diffPre, prevPre-pre.R)
		}
		prevLin = lin.R
		prevPre = pre.R
	}
	t.Logf("bottom row linear diffs: %v", diffLin)
	t.Logf("bottom row preview diffs: %v", diffPre)

	for _, s := range []struct {
		label   string
		x, y, z int
	}{
		{"panel0_bottom_left", 0, 0, 0},
		{"panel0_bottom_right", int(dim.X) - 1, 0, 0},
		{"panel0_mid", 2, 2, 0},
		{"panel0_top_left", 0, int(dim.Y) - 1, 0},
		{"panel0_top_right", int(dim.X) - 1, int(dim.Y) - 1, 0},
		{"panel1_bottom_left", 0, 0, 1},
		{"panel1_mid", 2, 2, 1},
		{"panel2_bottom_left", 0, 0, 2},
		{"panel2_mid", 2, 2, 2},
	} {
		lin := colorAt(linear, dim, s.x, s.y, s.z)
		pre := colorAt(preview, dim, s.x, s.y, s.z)
		addr := fmt.Sprintf("(x=%d,y=%d,z=%d)", s.x, s.y, s.z)
		t.Logf("%s %s linear=%+v preview=%+v", s.label, addr, lin, pre)
	}

	topLeft := colorAt(preview, dim, 0, int(dim.Y)-1, 0)
	topRight := colorAt(preview, dim, int(dim.X)-1, int(dim.Y)-1, 0)
	bottomRight := colorAt(preview, dim, int(dim.X)-1, 0, 0)

	if topLeft.R < 0.9 || topLeft.G < 0.9 || topLeft.B < 0.9 {
		t.Fatalf("expected near-white at top-left, got %+v", topLeft)
	}
	if topRight.R < 0.9 || topRight.G < 0.9 || topRight.B < 0.9 {
		t.Fatalf("expected near-white at top-right, got %+v", topRight)
	}
	if bottomRight.R > 0.05 || bottomRight.G > 0.05 || bottomRight.B > 0.05 {
		t.Fatalf("expected near-black at bottom-right, got %+v", bottomRight)
	}
}
