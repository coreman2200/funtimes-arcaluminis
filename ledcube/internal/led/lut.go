package led

import "github.com/coreman2200/funtimes-arcaluminis/ledcube/internal/render"

// Order holds panel/row flip behaviors. Extend as needed.
type Order struct {
	XFlipEveryRow   bool
	YFlipEveryPanel bool
}

// BuildLUT constructs normalized world positions [0,1]^3 for each index in raster order.
// This stub produces a regular lattice ignoring pitch/gap; extend to bake your physical layout.
func BuildLUT(dim render.Dimensions, order Order, pitchMM, gapMM float64) []render.Vec3 {
	n := dim.X * dim.Y * dim.Z
	out := make([]render.Vec3, n)
	idx := 0
	for z := 0; z < dim.Z; z++ {
		for y := 0; y < dim.Y; y++ {
			// Optionally flip X per row
			if order.XFlipEveryRow && (y%2 == 1) {
				for x := dim.X - 1; x >= 0; x-- {
					out[idx] = render.Vec3{
						X: float64(x) / float64(max(1, dim.X-1)),
						Y: float64(y) / float64(max(1, dim.Y-1)),
						Z: float64(z) / float64(max(1, dim.Z-1)),
					}
					idx++
				}
			} else {
				for x := 0; x < dim.X; x++ {
					out[idx] = render.Vec3{
						X: float64(x) / float64(max(1, dim.X-1)),
						Y: float64(y) / float64(max(1, dim.Y-1)),
						Z: float64(z) / float64(max(1, dim.Z-1)),
					}
					idx++
				}
			}
		}
	}
	return out
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
