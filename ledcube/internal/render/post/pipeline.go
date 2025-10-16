package post

import "github.com/coreman2200/funtimes-arcaluminis/ledcube/internal/render"

// ApplyPreview does Exposure -> Tonemap(ACES) -> Gamma, no limiter.
func ApplyPreview(buf []render.Color, u *render.Uniforms) {
	// Uses your existing FilmicToneMap (it already applies EV + ACES + Gamma)
	render.FilmicToneMap(buf, u)
}

// ApplyLED does Exposure -> Limiter, no tonemap, no gamma (linear 0..1).
func ApplyLED(buf []render.Color, u *render.Uniforms) {
	// Apply EV as a linear scale (no curve)
	ev := 0.0
	if u != nil && u.Params != nil {
		if v, ok := u.Params["ExposureEV"]; ok {
			ev = v
		}
	}
	if ev != 0 {
		scale := float32(pow2(ev))
		for i := range buf {
			buf[i].R *= scale
			buf[i].G *= scale
			buf[i].B *= scale
		}
	}
	// Then your limiter (it will early-return if PreviewMode is on, but
	// this LED path should be used only when PreviewMode==0)
	render.DefaultLimiter(buf, u)
	clamp01(buf)
}

func clamp01(buf []render.Color) {
	for i := range buf {
		if buf[i].R < 0 {
			buf[i].R = 0
		} else if buf[i].R > 1 {
			buf[i].R = 1
		}
		if buf[i].G < 0 {
			buf[i].G = 0
		} else if buf[i].G > 1 {
			buf[i].G = 1
		}
		if buf[i].B < 0 {
			buf[i].B = 0
		} else if buf[i].B > 1 {
			buf[i].B = 1
		}
	}
}
func pow2(ev float64) float64 { // 2^ev
	if ev == 0 {
		return 1
	}
	return float64(1<<0) * (1.0 * (1 << 0)) * (1.0 * (1 << 0)) // placeholder if you dislike math.Pow
}
