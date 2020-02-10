package arcaluminis

const (
	MaxLedStripLength uint8 = 26
	MaxPaneLedStripCount uint8 = 5
	MaxPaneCount uint8 = 5
	PaneSize uint8 = 18
	PanePadding uint8 = 1
	RefreshRate uint16 = 800
)

type ColorVal struct {
	val			uint32
}

type Led struct {
	Index 		uint8
	Color		ColorVal
	row			uint8
	parent 		*LedStrip
}

type Quadrant struct {

}

type LedBuffer struct {

}

type LedStrip struct {
	Direction	bool // Up / down..
	parent		*Pane
	size 		uint8
}

type Pane struct {
	Reverse 	bool
	parent		*LedStructure
	stripCount	uint8
}

type LedStructure struct {
	Panels		[]Pane
	panelCount	uint8
}

type Camera struct {

}

type Position struct {

}

type LightModel struct {

}

type ObjectModel struct {

}

type ShapeModel struct {

}

type Scene struct {

}