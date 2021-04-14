package model

import (
	"bytes"
	"image"
	"math"
	"sort"
)

const (
	MaxLedStripLength    uint8  = 26
	MaxPaneLedStripCount uint8  = 5
	MaxPaneCount         uint8  = 4  // should be 5..
	PaneSize             uint8  = 18 // Corresponds to Physical size (18^2") per panel
	PanePadding          uint8  = 1
	RefreshRate          uint16 = 800
)

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
}

func NewLed(p *LedStrip, i uint8, r uint8, c uint32) Led {
	v := Led{
		index:     i,
		row:       r,
		parent:    p,
		baseColor: NewColor(c),
		Color:     NewColor(c),
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
	Strip     []Led
	parent    *Pane
	baseColor ColorVal
	size      uint8
}

func NewStrip(p *Pane, i uint8, s uint8, d bool, c ColorVal) LedStrip {
	v := LedStrip{
		parent:    p,
		index:     i,
		size:      s,
		baseColor: c,
		Direction: d,
		Strip:     make([]Led, 0),
	}

	for i := 0; i < int(s); i++ {
		ss := NewLed(&v, uint8(i), uint8(v.index), DFLT_COLOR_INIT)
		v.Strip = append(v.Strip, ss)
	}

	return v
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
	ss := make([]*Led, s.size)

	for i, v := range s.Strip {
		ss[i] = &v
	}

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
	LedStrips  []LedStrip
	stripCount uint8
	baseColor  ColorVal
	Properties PhysicalProperties
}

func NewPane(p *LedStructure, i uint8, c uint8, r bool, b ColorVal) Pane {
	v := Pane{
		index:      i,
		stripCount: c,
		parent:     p,
		baseColor:  b,
		Reverse:    r,
		LedStrips:  make([]LedStrip, 0),
	}

	for i := 0; i < int(v.stripCount); i++ {
		d := math.Mod(float64(i), 2) > 0 && !v.Reverse
		ss := NewStrip(&v, uint8(i), uint8(MaxLedStripLength), d, b)
		v.LedStrips = append(v.LedStrips, ss)
	}

	return v
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
	s := make([]*LedStrip, p.stripCount)
	for i, v := range p.LedStrips {
		s[i] = &v
	}

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
	XOffset int
	YOffset int
	ZOffset int
	Padding int
	Width   uint
	Height  uint
	Depth   uint
}

type LedStructure struct {
	index      uint8
	baseColor  ColorVal
	panels     []Pane
	PanelCount uint8
	Properties PhysicalProperties
}

func NewLedStructure() LedStructure {
	v := LedStructure{
		index:      0,
		baseColor:  NewColor(DFLT_COLOR_INIT),
		panels:     make([]Pane, 0),
		PanelCount: MaxPaneCount,
		Properties: PhysicalProperties{
			Width:  uint(PaneSize),
			Height: uint(PaneSize),
		},
	}

	for i := 0; i < int(v.PanelCount); i++ {
		r := math.Mod(float64(i), 2) > 0
		ss := NewPane(&v, uint8(i), uint8(MaxLedStripLength), r, v.baseColor)
		v.panels = append(v.panels, ss)
	}

	return v
}

func (p *LedStructure) Image() *image.NRGBA {
	ls := p.Leds()
	im := image.NewNRGBA(image.Rect(0, 0, len(ls), 1))
	for x := 0; x < im.Rect.Max.X; x++ {
		im.SetNRGBA(x, 0, ls[x].Color.ToRGB())
	}
	return im
}

func (s *LedStructure) Panel(i int) Pane {
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
