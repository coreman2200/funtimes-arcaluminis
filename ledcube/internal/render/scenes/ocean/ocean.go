package ocean

import (
	"math"

	"github.com/coreman2200/funtimes-arcaluminis/ledcube/internal/render"
)

type Renderer struct {
	name, preset string

	// persistent sim state (XZ footprint)
	H    []float64 // height
	V    []float64 // velocity
	X, Z int

	// time book-keeping (optional)
	prevT float64
	initd bool
}

func New(name string) *Renderer {
	return &Renderer{name: name, preset: "CalmDawn"}
}

func (r *Renderer) Name() string { return r.name }
func (r *Renderer) Presets() []string {
	return []string{"CalmDawn", "SunnyDay", "Sunset", "NightStorm"}
}

func assign(u *render.Uniforms, kv map[string]float64) {
	if u == nil {
		return
	}
	if u.Params == nil {
		u.Params = map[string]float64{}
	}
	for k, v := range kv {
		u.Params[k] = v
	}
}

func (r *Renderer) ApplyPreset(p string, u *render.Uniforms) {
	r.preset = p
	if u == nil {
		return
	}
	// sensible defaults per preset
	switch p {
	case "CalmDawn":
		assign(u, map[string]float64{
			"TideAmp": 0.2, "TidePeriodS": 120.0, "WaveSpeed": 0.9, "Damping": 0.015, "Wind": 0.05,
			"Foaminess": 0.15, "Choppiness": 0.35, "SkySat": 0.9, "DayPeriodS": 240.0, "Storminess": 0.0,
			"WaterHue": 0.58, "WaterAbsorb": 0.20, "BaseIntensity": 1.0, "PreviewGamma": 1.6,
			"SkyCycleScale": 0.0,
		})
	case "SunnyDay":
		assign(u, map[string]float64{
			"TideAmp": 0.25, "TidePeriodS": 180.0, "WaveSpeed": 1.2, "Damping": 0.01, "Wind": 0.1,
			"Foaminess": 0.18, "Choppiness": 0.5, "SkySat": 1.0, "DayPeriodS": 240.0, "Storminess": 0.0,
			"WaterHue": 0.55, "WaterAbsorb": 0.15, "BaseIntensity": 1.1, "PreviewGamma": 1.6,
			"SkyCycleScale": 0.0,
		})
	case "Sunset":
		assign(u, map[string]float64{
			"TideAmp": 0.22, "TidePeriodS": 180.0, "WaveSpeed": 1.0, "Damping": 0.012, "Wind": 0.08,
			"Foaminess": 0.14, "Choppiness": 0.45, "SkySat": 1.1, "DayPeriodS": 240.0, "Storminess": 0.0,
			"WaterHue": 0.53, "WaterAbsorb": 0.18, "BaseIntensity": 1.0, "PreviewGamma": 1.7,
			"SkyCycleScale": 0.0,
		})
	case "NightStorm":
		assign(u, map[string]float64{
			"TideAmp": 0.3, "TidePeriodS": 150.0, "WaveSpeed": 1.3, "Damping": 0.02, "Wind": 0.35,
			"Foaminess": 0.30, "Choppiness": 0.8, "SkySat": 0.7, "DayPeriodS": 240.0, "Storminess": 0.8,
			"LightningRate": 0.15, "WaterHue": 0.60, "WaterAbsorb": 0.25, "BaseIntensity": 1.0, "PreviewGamma": 1.6,
			"SkyCycleScale": 0.0,
		})
	}
}

func (r *Renderer) Params() map[string]float64 {
	return map[string]float64{
		"TideAmp": 0.22, "TidePeriodS": 180, "WaveSpeed": 1.1, "Damping": 0.012, "Wind": 0.08,
		"Foaminess": 0.18, "Choppiness": 0.45, "SkySat": 1.0, "DayPeriodS": 240.0, "Storminess": 0.0,
		"LightningRate": 0.0, "WaterHue": 0.56, "WaterAbsorb": 0.18, "BaseIntensity": 1.0, "PreviewGamma": 1.6,
		"SeaLevel": 0.65, // 0..1 baseline waterline (fraction of height)
		"WaveAmp":  0.10, // scale of H contribution
		"HMax":     0.5,  // hard cap for |H| to avoid runaway
		// orientation helpers if needed
		"FlipX": 0, "FlipZ": 1,
		"SkyCycleScale": 0.0,
	}
}

// read a param with default
func pget(u *render.Uniforms, key string, def float64) float64 {
	if u == nil || u.Params == nil {
		return def
	}
	if v, ok := u.Params[key]; ok {
		return v
	}
	return def
}

// clamp to [0,1]
func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

// ---- render entrypoint ----
func (r *Renderer) Render(dst []render.Color, _ []render.Vec3, dim render.Dimensions, t float64, u *render.Uniforms, _ *render.Resources) {
	X, Y, Z := int(dim.X), int(dim.Y), int(dim.Z)
	if len(dst) < X*Y*Z {
		return
	}

	timeScale := pget(u, "TimeScale", 1.0)
	if timeScale < 0 {
		timeScale = 0
	}
	phaseT := t * timeScale
	simScale := timeScale
	if simScale < 0.01 {
		simScale = 0.01
	}

	// ensure state
	if !r.initd || r.X != X || r.Z != Z {
		r.X, r.Z = X, Z
		r.H = make([]float64, X*Z)
		r.V = make([]float64, X*Z)
		seedHeights(r.H, X, Z)
		r.prevT = phaseT
		r.initd = true
	}

	// params
	tideAmp := pget(u, "TideAmp", 0.22)
	tidePeriod := pget(u, "TidePeriodS", 180.0)
	waveSpeed := pget(u, "WaveSpeed", 1.1)
	damping := pget(u, "Damping", 0.012)
	wind := pget(u, "Wind", 0.08)
	foaminess := pget(u, "Foaminess", 0.18)
	choppy := pget(u, "Choppiness", 0.45)
	skySat := pget(u, "SkySat", 1.0)
	dayPeriod := pget(u, "DayPeriodS", 240.0)
	storminess := pget(u, "Storminess", 0.0)
	lightRate := pget(u, "LightningRate", 0.0)
	waterHue := pget(u, "WaterHue", 0.56)
	absorb := pget(u, "WaterAbsorb", 0.18)
	baseI := pget(u, "BaseIntensity", 1.0)
	prevGamma := pget(u, "PreviewGamma", 1.6)
	sea := clamp01(pget(u, "SeaLevel", 0.45))
	waveAmp := pget(u, "WaveAmp", 0.10)
	hMax := clamp01(pget(u, "HMax", 0.35))

	flipX := pget(u, "FlipX", 0) > 0.5
	flipZ := pget(u, "FlipZ", 1) > 0.5

	// integrate sim (small fixed dt; use t for phase drift)
	dt := clamp(r.prevT, 0, 1e9)
	_ = dt
	// fixed timestep ~16ms for stability, decoupled from host t
	r.stepSim(0.016, waveSpeed*simScale, damping, wind*simScale, choppy)

	// --- zero-mean & limit H to prevent drift ---
	n := r.X * r.Z
	if n > 0 {
		var sum float64
		for i := 0; i < n; i++ {
			sum += r.H[i]
		}
		mean := sum / float64(n)
		// remove average (kills DC drift)
		for i := 0; i < n; i++ {
			r.H[i] -= mean
		}
		// soft clip to +/- hMax (prevents rare runaway)
		for i := 0; i < n; i++ {
			if r.H[i] > hMax {
				r.H[i] = hMax
			}
			if r.H[i] < -hMax {
				r.H[i] = -hMax
			}
		}
	}

	tide := tideAmp * math.Sin(2*math.Pi*phaseT/math.Max(1e-6, tidePeriod))
	// clamp the final mean sea level to [0,1]
	baseLevel := clamp01(sea + tide)

	// sun angle for sky gradient
	skyCycle := pget(u, "SkyCycleScale", 0.0)
	dayPhase := 0.0
	if skyCycle > 0 {
		dayPhase = math.Mod((t*skyCycle)/math.Max(1e-6, dayPeriod), 1.0)
	}
	sunElev := math.Sin(2 * math.Pi * dayPhase) // -1..1

	// lightning flash
	flash := 0.0
	if storminess > 0 && lightRate > 0 {
		// cheap pseudo-random flash envelope
		p := fract(math.Sin(phaseT*13.37) * 43758.5453)
		if p < lightRate*0.02 {
			flash = 1.0
		}
	}

	// draw columns
	i := 0
	for z := 0; z < Z; z++ {
		for y := 0; y < Y; y++ {
			for x := 0; x < X; x++ {
				// visual axis flips (don’t change linear order)
				vx := x
				vz := z
				if flipX {
					vx = (X - 1) - vx
				}
				if flipZ {
					vz = (Z - 1) - vz
				}

				// surface height (0..Y-1)
				h := r.surfaceHeight(vx, vz, baseLevel, float64(Y), waveAmp)
				hSurf := h + 0.25

				if float64(y) <= hSurf {
					// WATER voxel
					depthN := clamp((h-float64(y))/4.0, 0, 1) // attenuate with depth near surface
					// blue from hue + subtle green
					R, G, B := hsv(waterHue, 0.85, 0.9)
					// absorption in depth
					G *= (1.0 - absorb*0.5*depthN)
					R *= (1.0 - absorb*0.8*depthN)

					// apply universal knobs (if your renderer didn’t already)
					sat := clamp01(pget(u, "Saturation", 1.0))
					baseI := pget(u, "BaseIntensity", 1.0)

					Yl := 0.2126*R + 0.7152*G + 0.0722*B
					R = Yl + (R-Yl)*sat
					G = Yl + (G-Yl)*sat
					B = Yl + (B-Yl)*sat

					// surface highlight near h
					nearSurf := clamp(1.0-math.Abs(float64(y)-h), 0, 1) // use true h
					spec := 0.3 * nearSurf * clamp(0.2+0.8*sunElev, 0, 1)
					R = clamp(R+spec, 0, 1)
					G = clamp(G+spec, 0, 1)
					B = clamp(B+spec, 0, 1)
					// foam if steep/fast
					if r.foamy(vx, vz, foaminess) && nearSurf > 0.5 {
						R = clamp(R+0.8*nearSurf, 0, 1)
						G = clamp(G+0.8*nearSurf, 0, 1)
						B = clamp(B+0.8*nearSurf, 0, 1)
					}
					// If the engine Preview pipeline will do tonemap+gamma, avoid double gamma here:
					preview := u != nil && u.Params != nil && (u.Params["PreviewMode"] > 0.5 || u.Params["PreviewBypass"] > 0.5)
					var outR, outG, outB float64
					if preview {
						// write scene-linear 0..1 here; let post handle exposure+tonemap+gamma
						outR = clamp01(R * baseI)
						outG = clamp01(G * baseI)
						outB = clamp01(B * baseI)
					} else {
						// (LED or non-preview path): keep your original renderer gamma if you need it
						ig := 1.0 / math.Max(1e-6, prevGamma)
						outR = math.Pow(clamp01(R*baseI), ig)
						outG = math.Pow(clamp01(G*baseI), ig)
						outB = math.Pow(clamp01(B*baseI), ig)
					}
					dst[i].R = float32(outR)
					dst[i].G = float32(outG)
					dst[i].B = float32(outB)
				} else {
					// SKY voxel
					yn := float64(y) / float64(Y-1) // 0..1 bottom→top
					R, G, B := skyGradient(yn, sunElev, skySat)
					// storm flash lifts sky
					if flash > 0 {
						R = clamp(R+flash, 0, 1)
						G = clamp(G+flash, 0, 1)
						B = clamp(B+flash, 0, 1)
					}
					// If the engine Preview pipeline will do tonemap+gamma, avoid double gamma here:
					preview := u != nil && u.Params != nil && (u.Params["PreviewMode"] > 0.5 || u.Params["PreviewBypass"] > 0.5)
					var outR, outG, outB float64
					if preview {
						// write scene-linear 0..1 here; let post handle exposure+tonemap+gamma
						outR = clamp01(R * baseI)
						outG = clamp01(G * baseI)
						outB = clamp01(B * baseI)
					} else {
						// (LED or non-preview path): keep your original renderer gamma if you need it
						ig := 1.0 / math.Max(1e-6, prevGamma)
						outR = math.Pow(clamp01(R*baseI), ig)
						outG = math.Pow(clamp01(G*baseI), ig)
						outB = math.Pow(clamp01(B*baseI), ig)
					}
					dst[i].R = float32(outR)
					dst[i].G = float32(outG)
					dst[i].B = float32(outB)
				}
				i++
			}
		}
	}

	r.prevT = phaseT
}

// ---- sim + color helpers ----

func (r *Renderer) idx(x, z int) int { return z*r.X + x }

func (r *Renderer) stepSim(dt, c, damping, wind, choppy float64) {
	if r.X*r.Z == 0 {
		return
	}
	X, Z := r.X, r.Z

	// discrete laplacian on H → accelerates V
	for z := 0; z < Z; z++ {
		for x := 0; x < X; x++ {
			i := r.idx(x, z)
			// neighbors with clamped boundaries (reflect)
			hc := r.H[i]
			hl := r.H[r.idx(clampi(x-1, 0, X-1), z)]
			hr := r.H[r.idx(clampi(x+1, 0, X-1), z)]
			hd := r.H[r.idx(x, clampi(z-1, 0, Z-1))]
			hu := r.H[r.idx(x, clampi(z+1, 0, Z-1))]
			lap := (hl + hr + hd + hu - 4.0*hc)

			acc := c * c * lap
			r.V[i] += acc * dt
			r.V[i] *= (1.0 - damping)
		}
	}
	// add a little wind/chop (moving phase)
	phase := func(x, z int) float64 { return math.Sin(0.11*float64(x) + 0.13*float64(z) + 1.7*float64(choppy)) }
	for z := 0; z < Z; z++ {
		for x := 0; x < X; x++ {
			i := r.idx(x, z)
			r.V[i] += wind * 0.02 * phase(x, z)
		}
	}
	// integrate height
	for i := 0; i < X*Z; i++ {
		r.H[i] += r.V[i] * dt
	}

	// light viscosity on H (Gaussian-ish 1-4-1 kernel separable)
	if r.X*r.Z > 0 {
		tmp := make([]float64, r.X*r.Z)
		// horizontal pass
		for z := 0; z < r.Z; z++ {
			for x := 0; x < r.X; x++ {
				i := r.idx(x, z)
				l := r.H[r.idx(clampi(x-1, 0, r.X-1), z)]
				c := r.H[i]
				rgt := r.H[r.idx(clampi(x+1, 0, r.X-1), z)]
				tmp[i] = (l + 4*c + rgt) / 6.0
			}
		}
		// vertical pass
		for z := 0; z < r.Z; z++ {
			for x := 0; x < r.X; x++ {
				i := r.idx(x, z)
				d := tmp[r.idx(x, clampi(z-1, 0, r.Z-1))]
				c := tmp[i]
				u := tmp[r.idx(x, clampi(z+1, 0, r.Z-1))]
				r.H[i] = (d + 4*c + u) / 6.0
			}
		}
	}

}

func (r *Renderer) surfaceHeight(x, z int, baseLevel float64, yMax float64, waveAmp float64) float64 {
	h := baseLevel + waveAmp*r.H[r.idx(x, z)]
	return clamp(h*(yMax-1), 0, yMax-1)
}

func (r *Renderer) slope(x, z int) float64 {
	X, Z := r.X, r.Z
	hl := r.H[r.idx(clampi(x-1, 0, X-1), z)]
	hr := r.H[r.idx(clampi(x+1, 0, X-1), z)]
	hd := r.H[r.idx(x, clampi(z-1, 0, Z-1))]
	hu := r.H[r.idx(x, clampi(z+1, 0, Z-1))]
	return math.Abs(hr-hl) + math.Abs(hu-hd)
}

func (r *Renderer) foamy(x, z int, foaminess float64) bool {
	s := r.slope(x, z)
	v := math.Abs(r.V[r.idx(x, z)])
	return s+v > (0.15 + 0.8*(1.0-foaminess)) // more foam when foaminess↑
}

// color utilities
func hsv(h, s, v float64) (float64, float64, float64) {
	h = h - math.Floor(h) // 0..1
	i := int(h * 6)
	f := h*6 - float64(i)
	p := v * (1 - s)
	q := v * (1 - f*s)
	t := v * (1 - (1-f)*s)
	switch i % 6 {
	case 0:
		return v, t, p
	case 1:
		return q, v, p
	case 2:
		return p, v, t
	case 3:
		return p, q, v
	case 4:
		return t, p, v
	default:
		return v, p, q
	}
}

func skyGradient(y, sunElev, sat float64) (float64, float64, float64) {
	// sunElev: -1..1  (night..noon)
	day := clamp01((sunElev + 0.2) * 0.7)            // 0 near night/twilight, 1 near day
	twilight := clamp01(1.0 - math.Abs(sunElev)*1.8) // glow near sunrise/sunset

	// bases
	nightTop := [3]float64{0.02, 0.04, 0.10}
	nightBot := [3]float64{0.05, 0.07, 0.12}

	dayTop := [3]float64{0.30, 0.55, 1.00} // bright blue
	dayBot := [3]float64{0.65, 0.80, 1.00} // near horizon brighten

	duskTop := [3]float64{0.35, 0.20, 0.45} // magenta-violet
	duskBot := [3]float64{1.00, 0.50, 0.20} // orange

	// choose night/day anchors then blend a pinch of twilight
	top := mix3n(nightTop, dayTop, day)
	bot := mix3n(nightBot, dayBot, day)
	top = mix3n(top, duskTop, twilight*0.35)
	bot = mix3n(bot, duskBot, twilight*0.35)

	// vertical blend bottom→top
	R := bot[0]*(1-y) + top[0]*y
	G := bot[1]*(1-y) + top[1]*y
	B := bot[2]*(1-y) + top[2]*y

	// apply saturation
	l := 0.2126*R + 0.7152*G + 0.0722*B
	R = l + (R-l)*sat
	G = l + (G-l)*sat
	B = l + (B-l)*sat

	return clamp(R, 0, 1), clamp(G, 0, 1), clamp(B, 0, 1)
}

func mix3n(a, b [3]float64, t float64) [3]float64 {
	return [3]float64{
		a[0] + (b[0]-a[0])*t,
		a[1] + (b[1]-a[1])*t,
		a[2] + (b[2]-a[2])*t,
	}
}

func clamp(x, a, b float64) float64 {
	if x < a {
		return a
	}
	if x > b {
		return b
	}
	return x
}
func clampi(x, a, b int) int {
	if x < a {
		return a
	}
	if x > b {
		return b
	}
	return x
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
func fract(x float64) float64 { return x - math.Floor(x) }

// seed small bumps
func seedHeights(H []float64, X, Z int) {
	for z := 0; z < Z; z++ {
		for x := 0; x < X; x++ {
			i := z*X + x
			// low amplitude random-ish seed based on coords
			n := math.Sin(float64(37*x+57*z))*0.03 + math.Sin(float64(11*x+23*z))*0.02
			H[i] = n
		}
	}
}
