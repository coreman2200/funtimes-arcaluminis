
package layout

type Dim struct{ X, Y, Z int }

type Serpentine struct {
	XFlipEveryRow   bool
	YFlipEveryPanel bool
}

type Layout struct {
	Dim        Dim
	Order      Serpentine
	PanelGapMM float64
	PitchMM    float64
}

// Index maps x,y,z -> linear LED index (0..N-1)
func (l Layout) Index(x, y, z int) int {
	yy := y
	xx := x
	if (y%2 == 1) && l.Order.XFlipEveryRow {
		xx = l.Dim.X - 1 - x
	}
	if l.Order.YFlipEveryPanel && (z%2 == 1) {
		yy = l.Dim.Y - 1 - y
	}
	perPanel := l.Dim.X * l.Dim.Y
	return z*perPanel + yy*l.Dim.X + xx
}

func (l Layout) Count() int {
	return l.Dim.X * l.Dim.Y * l.Dim.Z
}
