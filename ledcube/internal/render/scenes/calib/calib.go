package calib

import (
	"math"

	"github.com/coreman2200/funtimes-arcaluminis/ledcube/internal/render"
)

type Renderer struct {
	name   string
	preset string
}

func New(name string) *Renderer {
	return &Renderer{name: name, preset: "PanelChanSweep"}
}

func (r *Renderer) Name() string      { return r.name }
func (r *Renderer) Presets() []string { return []string{"PanelChanSweep"} }

// Satisfy your interface
func (r *Renderer) ApplyPreset(p string, u *render.Uniforms) {
	r.preset = p
}

// Optional: advertise tweakable knobs
func (r *Renderer) Params() map[string]float64 {
	// In Params():
	return map[string]float64{
		"PanelAxis": 2, // 0=X, 1=Y, 2=Z
		"FlipX":     0, "FlipY": 0, "FlipZ": 0,
		"Gamma":         1.8, // preview gamma lift
		"LRGamma":       1.2, // left→right darkening curve ( >1 steeper right edge )
		"TopWhitePow":   0.6, // bottom→top blend curve ( <1 quicker toward white )
		"TopWhiteMix":   1.0, // 0..1 how hard to pull toward white at top
		"BaseIntensity": 1.0, // overall intensity (pre-post)
		"RightFloor":    0.0, // minimum brightness at far-right (0..1)
		"Saturation":    1.0, // 0 = grayscale, 1 = full RGB
	}
}

// --- tiny helpers ---

func pget(u *render.Uniforms, key string, def float64) float64 {
	if u == nil || u.Params == nil {
		return def
	}
	if v, ok := u.Params[key]; ok {
		return v
	}
	return def
}

func bget(u *render.Uniforms, key string, def bool) bool {
	if u == nil || u.Bools == nil {
		return def
	}
	if v, ok := u.Bools[key]; ok {
		return v
	}
	return def
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

// Render fills 'dst' (one color per voxel). 'pos' is provided by the engine but we don't need it here.
// We iterate XYZ with x-fastest so the linear index matches your preview + VoxelCube.
func (r *Renderer) Render(dst []render.Color, _ []render.Vec3, dim render.Dimensions, _t float64, u *render.Uniforms, _ *render.Resources) {
	X, Y, Z := int(dim.X), int(dim.Y), int(dim.Z)
	if len(dst) < X*Y*Z {
		return
	}

	panelAxis := int(pget(u, "PanelAxis", 2)) // default Z panels
	flipX := bget(u, "FlipX", false) || pget(u, "FlipX", 0) > 0.5
	flipY := bget(u, "FlipY", false) || pget(u, "FlipY", 0) > 0.5
	flipZ := bget(u, "FlipZ", false) || pget(u, "FlipZ", 0) > 0.5
	gamma := pget(u, "Gamma", 1.8)
	if gamma <= 0 {
		gamma = 1
	}

	norm := func(i, n int) float64 {
		if n <= 1 {
			return 0
		}
		return float64(i) / float64(n-1)
	}

	i := 0
	for z := 0; z < Z; z++ {
		for y := 0; y < Y; y++ {
			for x := 0; x < X; x++ {

				// Visual flips (don’t change linear index ordering)
				vx, vy, vz := x, y, z
				if flipX {
					vx = (X - 1) - vx
				}
				if flipY {
					vy = (Y - 1) - vy
				}
				if flipZ {
					vz = (Z - 1) - vz
				}

				// Which panel are we on?
				panel := vz
				switch panelAxis {
				case 0:
					panel = vx
				case 1:
					panel = vy
				}

				// knobs
				lrPow := pget(u, "LRGamma", 1.2)
				topPow := pget(u, "TopWhitePow", 0.6)
				topMix := clamp01(pget(u, "TopWhiteMix", 1.0))
				rightFloor := clamp01(pget(u, "RightFloor", 0.0))
				baseInt := clamp01(pget(u, "BaseIntensity", 1.0))
				sat := clamp01(pget(u, "Saturation", 1.0))

				// base channel per panel
				ch := panel % 3
				r0, g0, b0 := 0.0, 0.0, 0.0
				switch ch {
				case 0:
					r0 = 1
				case 1:
					g0 = 1
				case 2:
					b0 = 1
				}

				// Left→Right: darken with curve + floor
				nx := norm(vx, X)
				lr := math.Pow(1.0-nx, lrPow)
				lr = rightFloor + (1.0-rightFloor)*lr

				// Bottom→Top: blend toward white with curve and strength
				ny := norm(vy, Y)
				bt := math.Pow(ny, topPow) // faster toward 1 near top
				bt *= topMix               // how hard to pull to white

				// start with a single channel and apply LR
				R := r0 * lr
				G := g0 * lr
				B := b0 * lr

				// blend toward white by bt
				R = R + (1.0-R)*bt
				G = G + (1.0-G)*bt
				B = B + (1.0-B)*bt

				// optional desaturation (visually cleaner)
				if sat < 1.0 {
					// simple luma
					Yl := 0.2126*R + 0.7152*G + 0.0722*B
					R = Yl + (R-Yl)*sat
					G = Yl + (G-Yl)*sat
					B = Yl + (B-Yl)*sat
				}

				// intensity before post
				R *= baseInt
				G *= baseInt
				B *= baseInt

				// preview gamma (keep as before)
				ig := 1.0 / math.Max(1e-6, pget(u, "Gamma", 1.8))
				R = math.Pow(clamp01(R), ig)
				G = math.Pow(clamp01(G), ig)
				B = math.Pow(clamp01(B), ig)

				// apply universal knobs (if your renderer didn’t already)
				baseI := pget(u, "BaseIntensity", 1.0)
				prevGamma := pget(u, "PreviewGamma", 1.6)

				Yl := 0.2126*R + 0.7152*G + 0.0722*B
				R = Yl + (R-Yl)*sat
				G = Yl + (G-Yl)*sat
				B = Yl + (B-Yl)*sat

				R = math.Pow(clamp01(R*baseI), 1.0/math.Max(1e-6, prevGamma))
				G = math.Pow(clamp01(G*baseI), 1.0/math.Max(1e-6, prevGamma))
				B = math.Pow(clamp01(B*baseI), 1.0/math.Max(1e-6, prevGamma))

				dst[i].R = float32(clamp01(R))
				dst[i].G = float32(clamp01(G))
				dst[i].B = float32(clamp01(B))

				i++
			}
		}
	}
}
