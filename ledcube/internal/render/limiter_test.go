package render

import "testing"

// helper to estimate current in mA using same model as limiter
func estCurrent(buf []Color, chanmA float32) float64 {
	total := 0.0
	for i := range buf {
		total += float64((buf[i].R + buf[i].G + buf[i].B) * chanmA)
	}
	return total
}

func TestDefaultLimiterBudgetClamp(t *testing.T) {
	// 10 voxels all white
	n := 10
	buf := make([]Color, n)
	for i := range buf {
		buf[i] = Color{1,1,1}
	}
	u := &Uniforms{Params: map[string]float64{
		"LEDChan_mA": 20,       // 60mA at white per LED
		"Budget_mA":  300,      // allow 300 mA total
		"WhiteCap":   3.0,
		"LimiterKnee": 0.9,
	}}

	// pre-limit current would be 10 * 60 = 600 mA
	DefaultLimiter(buf, u)
	cur := estCurrent(buf, 20)
	if cur > 300.1 {
		t.Fatalf("expected <= 300mA after limit, got %.2f mA", cur)
	}
}

func TestWhiteCap(t *testing.T) {
	buf := []Color{{1,1,1}} // sum=3
	u := &Uniforms{Params: map[string]float64{"WhiteCap": 1.5}} // cap to 1.5
	DefaultLimiter(buf, u)
	sum := buf[0].R + buf[0].G + buf[0].B
	if sum > 1.5001 {
		t.Fatalf("expected sum <= 1.5, got %f", sum)
	}
}
