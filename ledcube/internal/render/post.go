package render

import "math"

// DefaultToneMap applies a simple gamma curve and clamps to [0,1].
func DefaultToneMap(buf []Color) {
	const invGamma = 1.0 / 2.2
	for i := range buf {
		buf[i].R = clamp01(powf(buf[i].R, invGamma))
		buf[i].G = clamp01(powf(buf[i].G, invGamma))
		buf[i].B = clamp01(powf(buf[i].B, invGamma))
	}
}

// DefaultLimiter is a no-op placeholder; wire your power limiter here.
func DefaultLimiter(buf []Color, u *Uniforms) {
	// Intentionally empty — hook your power-aware logic here.
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
