package fake

import (
	"fmt"

	"github.com/coreman2200/funtimes-arcaluminis/ledcube/internal/render"
)

// Driver prints a compact summary of the frame (first voxel & avg), useful for headless tests.
type Driver struct {
	Count int
}

func (d *Driver) Write(buf []render.Color) error {
	d.Count++
	// compute simple average for log
	var r, g, b float64
	for i := range buf {
		r += float64(buf[i].R)
		g += float64(buf[i].G)
		b += float64(buf[i].B)
	}
	n := float64(len(buf))
	if n == 0 {
		n = 1
	}
	fmt.Printf("[frame %04d] avg=(%.2f,%.2f,%.2f) first=(%.2f,%.2f,%.2f)\n",
		d.Count, r/n, g/n, b/n, buf[0].R, buf[0].G, buf[0].B)
	return nil
}
