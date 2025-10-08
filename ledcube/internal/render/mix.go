package render

// Mix blends two framebuffers (a,b) into dst using alpha (0..1).
// Channels are linear; no gamma assumed.
func Mix(dst, a, b []Color, alpha float64) {
	if alpha <= 0 {
		copy(dst, a)
		return
	}
	if alpha >= 1 {
		copy(dst, b)
		return
	}
	af := float32(1.0 - alpha)
	bf := float32(alpha)
	n := len(dst)
	for i := 0; i < n; i++ {
		dst[i].R = a[i].R*af + b[i].R*bf
		dst[i].G = a[i].G*af + b[i].G*bf
		dst[i].B = a[i].B*af + b[i].B*bf
	}
}
