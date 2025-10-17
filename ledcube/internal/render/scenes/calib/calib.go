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
	if u == nil {
		return
	}
	ensure(u, map[string]float64{
		"PanelAxis":     2,
		"FlipX":         0,
		"FlipY":         0,
		"FlipZ":         0,
		"Gamma":         1.7,
		"LRGamma":       1.4,
		"TopWhitePow":   2.0,
		"TopWhiteMix":   0.6,
		"BaseIntensity": 0.17,
		"RightFloor":    0.0,
		"Saturation":    1.0,
		"PreviewScale":  0.65,
	})
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
		"PreviewScale":  0.65,
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

func ensure(u *render.Uniforms, kv map[string]float64) {
	if u.Params == nil {
		u.Params = map[string]float64{}
	}
	for k, v := range kv {
		if _, ok := u.Params[k]; !ok {
			u.Params[k] = v
		}
	}
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
	lrPow := pget(u, "LRGamma", 1.2)
	topPow := pget(u, "TopWhitePow", 0.6)
	topMix := clamp01(pget(u, "TopWhiteMix", 1.0))
	rightFloor := clamp01(pget(u, "RightFloor", 0.0))
	baseInt := clamp01(pget(u, "BaseIntensity", 1.0))
	sat := clamp01(pget(u, "Saturation", 1.0))
	preview := false
	if u != nil && u.Params != nil {
		if u.Params["PreviewMode"] > 0.5 || u.Params["PreviewBypass"] > 0.5 {
			preview = true
		}
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
				lr := 1.0 - math.Pow(nx, lrPow)
				lr = rightFloor + (1.0-rightFloor)*lr

				// Bottom→Top: blend toward white with curve and strength
				ny := norm(vy, Y)
				bt := math.Pow(ny, topPow)
				if vy == Y-1 {
					bt = 1.0
				} else {
					bt *= topMix // how hard to pull to white for mid rows
				}

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

				// intensity before post (linear)
				R *= baseInt
				G *= baseInt
				B *= baseInt

				// clamp to linear 0..1 prior to downstream handling
				R = clamp01(R)
				G = clamp01(G)
				B = clamp01(B)

				var outR, outG, outB float64
				if preview {
					scale := clamp01(pget(u, "PreviewScale", 0.65))
					scaledR := clamp01(R * scale)
					scaledG := clamp01(G * scale)
					scaledB := clamp01(B * scale)
					if y == Y-1 {
						scaledR, scaledG, scaledB = 1, 1, 1
					}
					outR = scaledR
					outG = scaledG
					outB = scaledB
				} else {
					ig := 1.0 / math.Max(1e-6, gamma)
					outR = math.Pow(R, ig)
					outG = math.Pow(G, ig)
					outB = math.Pow(B, ig)
				}

				dst[i].R = float32(outR)
				dst[i].G = float32(outG)
				dst[i].B = float32(outB)

				i++
			}
		}
	}
}
