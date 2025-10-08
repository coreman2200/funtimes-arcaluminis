package sequence

import "math"

// clamp01 clamps x in [0,1].
func clamp01(x float64) float64 {
	if x < 0 {
		return 0
	}
	if x > 1 {
		return 1
	}
	return x
}

// smootherstep (cubic-ish) for ease="cubic"
func smootherstep(x float64) float64 {
	// 6x^5 - 15x^4 + 10x^3
	return x*x*x*(x*(x*6-15)+10)
}

func easeApply(kind string, x float64) float64 {
	switch kind {
	case "linear", "":
		return x
	case "smooth":
		// classic smoothstep 3x^2 - 2x^3
		return x*x*(3-2*x)
	case "cubic":
		return smootherstep(x)
	default:
		return x
	}
}

// Eval returns the value of the envelope at time t (seconds).
// If there are no keys, returns 0; if one key, returns its value.
// Keys must be sorted by T ascending.
func (e Envelope) Eval(t float64) float64 {
	n := len(e.Keys)
	if n == 0 {
		return 0
	}
	if n == 1 {
		return e.Keys[0].V
	}
	// before first
	if t <= e.Keys[0].T {
		return e.Keys[0].V
	}
	// after last
	if t >= e.Keys[n-1].T {
		return e.Keys[n-1].V
	}
	// find segment
	for i := 0; i < n-1; i++ {
		a := e.Keys[i]
		b := e.Keys[i+1]
		if t >= a.T && t <= b.T {
			den := (b.T - a.T)
			if den <= 0 {
				return b.V
			}
			u := (t - a.T) / den
			u = clamp01(u)
			u = easeApply(a.Ease, u)
			return a.V + (b.V-a.V)*u
		}
	}
	return e.Keys[n-1].V
}

// BoolEval thresholds the envelope at 0.5 into a boolean.
func (e Envelope) BoolEval(t float64) bool {
	return e.Eval(t) >= 0.5
}
