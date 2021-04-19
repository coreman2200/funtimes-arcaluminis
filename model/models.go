package model

import (
	"bytes"
	"image"
	"math"
	"sort"
	"time"

	"periph.io/x/conn/v3/physic"
)

const (
	MaxLedStripLength    uint8            = 26
	MaxPaneLedStripCount uint8            = 5
	MaxPaneCount         uint8            = 4   // should be 5..
	PaneSize             int              = 300 // Corresponds to Physical size (18^2") per panel
	PanePadding          uint8            = 1
	RefreshRate          physic.Frequency = 800
)

type LedRenderer interface {
	Render()
	Clear()
}

type serializable interface {
	Index() uint8
	Serialize() []byte
}

type Led struct {
	index     uint8
	Color     ColorVal
	row       uint8
	baseColor ColorVal
	parent    *LedStrip
	Props     PhysicalProperties
}

func NewLed(p *LedStrip, i uint8, r uint8, c uint32) Led {
	size := p.Props.Width / 2
	yoff := (p.Props.Height / float32(p.size))
	xoff := size / 2 // p.Props.XOffset / 2
	v := Led{
		index:     i,
		row:       r,
		parent:    p,
		baseColor: NewColor(c),
		Color:     NewColor(c),
		Props: PhysicalProperties{
			Width:   size,
			Height:  size,
			XOffset: xoff,
			YOffset: float32(i)*yoff + p.Props.YOffset,
			ZOffset: float32(p.parent.index),
		},
	}

	return v
}

func (l *Led) Index() uint8 {
	return l.index
}

func (l *Led) ColorF(f func(v ColorVal, cs ...*Led)) {
	f(l.baseColor, l)
}
func (l *Led) SetColor(cv ColorVal) {
	l.Color = cv
}
func (l *Led) ScaleColor(s float64) {
	if s > 1.0 || s < 0.0 {
		return
	}

	l.Color.val = uint32(float64(l.Color.val) * s)
}

func (l *Led) Serialize() []byte {
	buf := new(bytes.Buffer)
	buf.Write(l.Color.Serialize())

	return buf.Bytes()
}

type Quadrant struct {
}

type LedBuffer *bytes.Buffer
type LedStrip struct {
	index     uint8
	Direction bool // Up / down..
	Strip     []*Led
	parent    *Pane
	baseColor ColorVal
	size      uint8
	Props     PhysicalProperties
}

func NewStrip(p *Pane, i uint8, s uint8, d bool, c ColorVal) LedStrip {
	xoff := float32((p.Props.Width / float32(p.stripCount)))
	yoff := p.Props.YOffset
	v := LedStrip{
		parent:    p,
		index:     i,
		size:      s,
		baseColor: c,
		Direction: d,
		Strip:     make([]*Led, 0),
		Props: PhysicalProperties{
			Width:   xoff / 2,
			Height:  float32(PaneSize),
			XOffset: (xoff * float32(i)),
			YOffset: yoff,
			ZOffset: float32(p.index),
		},
	}

	for i := 0; i < int(s); i++ {
		ss := NewLed(&v, uint8(i), uint8(v.index), c.val)
		v.Strip = append(v.Strip, &ss)
	}

	return v
}

func (s *LedStrip) BaseColor() ColorVal {
	return s.baseColor
}

func (s *LedStrip) Leds() []*Led {
	return s.sorted()
}

func (s *LedStrip) Index() uint8 {
	return s.index
}

func (s *LedStrip) ColorF(f func(v ColorVal, cs ...*Led)) {
	f(s.baseColor, s.Leds()...)
}
func (s *LedStrip) SetColor(cv ColorVal) {
	for _, v := range s.Leds() {
		v.SetColor(cv)
	}
}
func (s *LedStrip) ScaleColor(ss float64) {
	if ss > 1.0 || ss < 0.0 {
		return
	}

	for _, v := range s.Leds() {
		v.ScaleColor(ss)
	}
}

func (s *LedStrip) Serialize() []byte {
	buf := new(bytes.Buffer)

	ss := s.sorted()

	for _, v := range ss {
		buf.Write(v.Serialize())
	}

	return buf.Bytes()
}

func (s *LedStrip) sorted() []*Led {
	ss := make([]*Led, 0)
	ss = append(ss, s.Strip...)

	sort.Slice(ss, func(i, j int) bool {
		if s.Direction {
			return ss[j].Index() < ss[i].Index()
		} else {
			return ss[i].Index() < ss[j].Index()
		}
	})

	return ss
}

type Pane struct {
	index      uint8
	Reverse    bool
	parent     *LedStructure
	LedStrips  []*LedStrip
	stripCount uint8
	baseColor  ColorVal
	Props      PhysicalProperties
}

func NewPane(p *LedStructure, i uint8, c uint8, r bool, b ColorVal) Pane {
	v := Pane{
		index:      i,
		stripCount: c,
		parent:     p,
		baseColor:  b,
		Reverse:    r,
		LedStrips:  make([]*LedStrip, 0),
		Props: PhysicalProperties{
			Width:   float32(PaneSize),
			Height:  float32(PaneSize),
			ZOffset: (float32(PaneSize) * float32(i)) / float32(p.PanelCount),
		},
	}

	for i := 0; i < int(v.stripCount); i++ {
		d := math.Mod(float64(i), 2) > 0 && !v.Reverse
		ss := NewStrip(&v, uint8(i), uint8(MaxLedStripLength), d, b)
		v.LedStrips = append(v.LedStrips, &ss)
	}

	return v
}

func (p *Pane) BaseColor() ColorVal {
	return p.baseColor
}

func (p *Pane) Image() *image.NRGBA {
	ls := p.Leds()
	im := image.NewNRGBA(image.Rect(0, 0, len(ls), 1))
	for x := 0; x < im.Rect.Max.X; x++ {
		im.SetNRGBA(x, 0, ls[x].Color.ToRGB())
	}
	return im
}

func (p *Pane) Leds() []*Led {
	r := make([]*Led, 0)

	ss := p.sorted()
	for _, v := range ss {
		r = append(r, v.Leds()...)
	}

	return r
}

func (p *Pane) Index() uint8 {
	return p.index
}

func (p *Pane) ColorF(f func(v ColorVal, cs ...*Led)) {
	f(p.baseColor, p.Leds()...)
}
func (p *Pane) SetColor(cv ColorVal) {
	p.baseColor = cv
	for _, v := range p.Leds() {
		v.SetColor(cv)
	}
}
func (p *Pane) ScaleColor(s float64) {
	if s > 1.0 || s < 0.0 {
		return
	}

	for _, v := range p.Leds() {
		v.ScaleColor(s)
	}
}

func (p *Pane) Serialize() []byte {
	buf := new(bytes.Buffer)

	ss := p.sorted()

	for _, v := range ss {
		buf.Write(v.Serialize())
	}

	return buf.Bytes()
}

func (p *Pane) sorted() []*LedStrip {
	s := make([]*LedStrip, 0)
	s = append(s, p.LedStrips...)

	sort.Slice(s, func(i, j int) bool {
		if p.Reverse {
			return s[j].Index() < s[i].Index()
		} else {
			return s[i].Index() < s[j].Index()
		}
	})

	return s
}

type PhysicalProperties struct {
	XOffset  float32
	YOffset  float32
	ZOffset  float32
	PaddingX float32
	PaddingY float32
	Width    float32
	Height   float32
	Depth    float32
}

type LedStructure struct {
	index      uint8
	baseColor  ColorVal
	panels     []*Pane
	PanelCount uint8
	//drawer     display.Drawer
	//handle     spi.Port
	Props PhysicalProperties
}

func NewLedStructure() *LedStructure {
	v := LedStructure{
		index:      0,
		baseColor:  NewColor(DFLT_COLOR_INIT),
		panels:     make([]*Pane, 0),
		PanelCount: MaxPaneCount,
		Props: PhysicalProperties{
			Width:  float32(PaneSize),
			Height: float32(PaneSize),
			Depth:  float32(PaneSize),
		},
	}

	for i := 0; i < int(v.PanelCount); i++ {
		r := math.Mod(float64(i), 2) > 0
		ss := NewPane(&v, uint8(i), uint8(MaxLedStripLength), r, v.baseColor)
		v.panels = append(v.panels, &ss)
	}

	return &v
}

func (s *LedStructure) Image() *image.NRGBA {
	ls := s.Leds()
	im := image.NewNRGBA(image.Rect(0, 0, len(ls), 1))
	for x := 0; x < im.Rect.Max.X; x++ {
		im.SetNRGBA(x, 0, ls[x].Color.ToRGB())
	}
	return im
}

func (s *LedStructure) Panel(i int) *Pane {
	return s.panels[i]
}

func (s *LedStructure) Leds() []*Led {
	r := make([]*Led, 0)

	for _, v := range s.panels {
		r = append(r, v.Leds()...)
	}

	return r
}

func (s *LedStructure) Index() uint8 {
	return s.index
}

func (s *LedStructure) ColorF(f func(v ColorVal, cs ...*Led)) {
	f(s.baseColor, s.Leds()...)
}
func (s *LedStructure) SetColor(cv ColorVal) {
	s.baseColor = cv

	for _, v := range s.Leds() {
		v.SetColor(cv)
	}
}
func (s *LedStructure) ScaleColor(ss float64) {
	if ss > 1.0 || ss < 0.0 {
		return
	}

	for _, v := range s.Leds() {
		v.ScaleColor(ss)
	}
}

func (s *LedStructure) Serialize() []byte {
	buf := new(bytes.Buffer)
	for _, v := range s.panels {
		buf.Write(v.Serialize())
	}

	return buf.Bytes()
}

func (s *LedStructure) Update(t time.Duration) {
	nms := float64(t.Milliseconds()) / 1000
	rad := float64(360*nms/180.0) * math.Pi

	// Orient, Transform, Scale, Projection..
	s.TestManip(rad)
}

func (s *LedStructure) TestManip(rad float64) {
	for i, v := range s.panels {
		mm := math.Mod(float64(i), 3)
		rr := mm == 0
		gg := mm == 1
		bb := mm == 2

		col := NewColor(0)
		col.SetA(255)
		if rr {
			col.SetR(255)
		} else if gg {
			col.SetG(255)

		} else if bb {
			col.SetB(255)

		}

		v.SetColor(col)

		for ii, vv := range v.LedStrips {
			aa := NewColor(0xAA000000)
			vv.ColorF(func(c ColorVal, cs ...*Led) {
				d := float64(1) / float64(v.stripCount)

				ll := 1 / float64(len(cs))
				for in, cc := range cs {
					rate := 1 / float64(i+1) // per panel..
					poff := float64(ii) * (d * float64(2*math.Pi))
					ioff := float64(in) * ll * 2 * math.Pi
					delta := (rate * rad) + poff + ioff
					diffA := (math.Cos(delta) * float64(aa.GetA()))
					oldA := cc.Color.GetA()
					val := math.Max(0, math.Min(255, float64(oldA)-diffA))
					cc.Color.SetA(uint8(val))

				}
			})

			FadeTo(0, vv.Strip...)
		}
	}
}

func (s *LedStructure) Clear() {
	s.SetColor(NewColor(0))
}

// 3D Scene Structures
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
