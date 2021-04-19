package model

import (
	"image/color"
	"math"
)

const MAX_BRIGHTNESS uint8 = 200

const (
	ALPHA_OFFSET uint8 = 0x18
	GREEN_OFFSET uint8 = 0x10
	RED_OFFSET   uint8 = 0x08
	BLUE_OFFSET  uint8 = 0x0
)
const DFLT_COLOR_INIT uint32 = 0xFF9911CC

type ColorVal struct {
	index uint8
	val   uint32
}

func NewColor(c uint32) ColorVal {
	v := ColorVal{
		val: c,
	}
	return v
}

func (c *ColorVal) Color() uint32 {
	return c.val
}

func (c *ColorVal) ToRGBA() color.RGBA {
	return color.RGBA{c.GetR(), c.GetG(), c.GetB(), c.GetA()}
}

func (c *ColorVal) ToRGB() color.NRGBA {
	aa := float64(c.GetA())
	if aa > float64(MAX_BRIGHTNESS) {
		aa /= aa / float64(MAX_BRIGHTNESS)
	}
	aa /= 255.0
	rr := float64(c.GetR()) * aa
	gg := float64(c.GetG()) * aa
	bb := float64(c.GetB()) * aa

	col := color.NRGBA{
		R: uint8(rr),
		G: uint8(gg),
		B: uint8(bb),
		A: 255,
	}

	return col
}

func setcolor(c uint32, n uint8, off uint8) uint32 {
	var val uint32 = uint32(n) << off
	var mask uint32 = 0xFF << off
	return (c & (^mask)) | val
}

func getcolor(c uint32, off uint8) uint8 {
	var mask uint32 = 0xFF << off
	return uint8((c & (mask)) >> off)
}

func (c *ColorVal) SetR(r uint8) {
	c.val = setcolor(c.val, r, RED_OFFSET)
}
func (c *ColorVal) SetG(g uint8) {
	c.val = setcolor(c.val, g, GREEN_OFFSET)

}
func (c *ColorVal) SetB(b uint8) {
	c.val = setcolor(c.val, b, BLUE_OFFSET)
}
func (c *ColorVal) SetA(a uint8) {
	c.val = setcolor(c.val, a, ALPHA_OFFSET)
}

func (c *ColorVal) GetR() uint8 {
	return getcolor(c.val, RED_OFFSET)
}
func (c *ColorVal) GetG() uint8 {
	return getcolor(c.val, GREEN_OFFSET)

}
func (c *ColorVal) GetB() uint8 {
	return getcolor(c.val, BLUE_OFFSET)
}
func (c *ColorVal) GetA() uint8 {
	return getcolor(c.val, ALPHA_OFFSET)

}

func (c *ColorVal) Index() uint8 {
	return c.index
}

func (c *ColorVal) Serialize() []byte {
	buf := make([]byte, 3)
	col := c.ToRGB()

	buf = append(buf, col.R)
	buf = append(buf, col.G)
	buf = append(buf, col.B)

	// TODO Rem this format if anything's awry..
	//grb := uint32(gg)<<(GREEN_OFFSET) + uint32(rr)<<(RED_OFFSET) + uint32(bb)<<(BLUE_OFFSET)

	return buf
}

type ColorChanger interface {
	Index()
	ColorF(func(v ColorVal, cs ...*Led))
	SetColor(ColorVal)
	ScaleColor(float32)
}

func FadeTo(a uint8, cs ...*Led) {
	l := len(cs)

	//vv := int32(v.val)
	fa := 255 / int32(l)

	for i, v := range cs {
		cv := v.Color
		fd := 255 - int(fa)*i
		aa := math.Min(float64(cv.GetA()), float64(fd))
		v.Color.SetA(uint8(aa))

	}
}

func WaveBy(v ColorVal, cs ...*Led) {
	l := len(cs)
	diff := int(v.val) / l

	for i, v := range cs {
		ww := int32(math.Sin(2.0*math.Pi*float64(i)) * float64(diff))

		cv := v.baseColor.val
		v.Color.val = uint32(int32(cv) + (int32(i) * ww))
	}
}
