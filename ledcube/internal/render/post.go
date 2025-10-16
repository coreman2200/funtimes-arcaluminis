package render

import "math"

// Filmic/ACES tone map with exposure in EV and optional gamma (default 2.2).
// Reads from uniforms.Params:
//   - "ExposureEV" (default 0)
//   - "OutputGamma" (default 2.2)
func FilmicToneMap(buf []Color, u *Uniforms) {
	// exposure
	exposureEV := 0.0
	gamma := 2.2
	if u != nil && u.Params != nil {
		if v, ok := u.Params["ExposureEV"]; ok {
			exposureEV = v
		}
		if g, ok := u.Params["OutputGamma"]; ok && g > 0 {
			gamma = g
		}
	}
	exposure := float32(math.Pow(2.0, exposureEV))

	for i := range buf {
		r := buf[i].R * exposure
		g := buf[i].G * exposure
		b := buf[i].B * exposure

		r = acesApprox(r)
		g = acesApprox(g)
		b = acesApprox(b)

		if gamma != 1.0 {
			ig := float64(1.0 / gamma)
			r = powf(r, ig)
			g = powf(g, ig)
			b = powf(b, ig)
		}

		buf[i].R = clamp01(r)
		buf[i].G = clamp01(g)
		buf[i].B = clamp01(b)
	}
}

// DefaultToneMap now points to FilmicToneMap for convenience.
func DefaultToneMap(buf []Color) {
	FilmicToneMap(buf, &Uniforms{Params: map[string]float64{"ExposureEV": 0, "OutputGamma": 2.2}})
}

// DefaultLimiter applies a two-stage limiter:
// 1) Per-LED "white cap": scales (R,G,B) so R+G+B <= WhiteCap (default 3.0 = no cap)
// 2) Global current budget: estimates current and scales the whole frame to stay under Budget_mA
//
// Parameters (read from uniforms.Params):
//   - "WhiteCap" (sum of channels cap in linear space, default 3.0)
//   - "LEDChan_mA" (mA per color channel at full scale; WS2812 ≈ 20, default 20)
//   - "Budget_mA" (global budget in mA; if 0 or missing, limiter returns immediately)
//   - "LimiterKnee" (fraction of budget where soft limiting begins; default 0.9)
func DefaultLimiter(buf []Color, u *Uniforms) {
	if u == nil {
        return
    }
    // Preview disables limiter (new) + honor legacy flag
    if u.Params["PreviewMode"] > 0.5 || u.Params["PreviewBypass"] > 0.5 {
        return
    }

	// Params
	whiteCap := 3.0
	chanmA := 20.0
	budget := 0.0
	knee := 0.9
	if u.Params != nil {
		if v, ok := u.Params["WhiteCap"]; ok && v > 0 {
			whiteCap = v
		}
		if v, ok := u.Params["LEDChan_mA"]; ok && v > 0 {
			chanmA = v
		}
		if v, ok := u.Params["Budget_mA"]; ok && v > 0 {
			budget = v
		}
		if v, ok := u.Params["LimiterKnee"]; ok && v > 0 && v < 1 {
			knee = v
		}
	}

	// 1) Per-LED white cap
	wc := float32(whiteCap)
	for i := range buf {
		s := buf[i].R + buf[i].G + buf[i].B
		if s > wc && s > 0 {
			scale := wc / s
			buf[i].R *= scale
			buf[i].G *= scale
			buf[i].B *= scale
		}
	}

	// 2) Global budget
	if budget <= 0 {
		return
	}
	// Estimate total current
	var total float64
	cm := float32(chanmA)
	for i := range buf {
		total += float64((buf[i].R + buf[i].G + buf[i].B) * cm)
	}

	if total <= 0 {
		return
	}
	// Soft knee: start scaling gently after knee*budget, fully meet budget above budget
	ratio := total / budget
	if ratio <= 1.0 {
		if ratio <= knee {
			return // under knee, do nothing
		}
		// between knee and 1.0 -> apply gentle scale
		// map ratio in [knee,1] to scale s in [1, budget/total]
		minS := budget / total
		t := (ratio - knee) / (1.0 - knee) // 0..1
		s := float32(1.0 - t*(1.0-minS))
		applyGlobalScale(buf, s)
		return
	}
	// Above budget: hard scale to meet budget
	s := float32(budget / total)
	applyGlobalScale(buf, s)
}

func applyGlobalScale(buf []Color, s float32) {
	if s >= 1.0 {
		return
	}
	for i := range buf {
		buf[i].R *= s
		buf[i].G *= s
		buf[i].B *= s
	}
}

func clamp01(x float32) float32 {
	if x < 0 {
		return 0
	}
	if x > 1 {
		return 1
	}
	return x
}

func powf(x float32, p float64) float32 {
	return float32(math.Pow(float64(x), p))
}

// Approximate ACES filmic curve (Narkowicz 2015).
func acesApprox(x float32) float32 {
	a := float32(2.51)
	b := float32(0.03)
	c := float32(2.43)
	d := float32(0.59)
	e := float32(0.14)
	return clamp01((x * (a*x + b)) / (x*(c*x+d) + e))
}
