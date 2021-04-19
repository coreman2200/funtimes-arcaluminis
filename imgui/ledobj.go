package imgui

import (
	"image"
	"math"

	"github.com/AllenDang/giu"
	"github.com/coreman2200/funtimes-arcaluminis/model"
)

type RenderObject interface {
	/*parent() *RenderObject
	children() []*RenderObject

	RelX() float32
	RelY() float32
	RelZ() float32
	Width() float32
	Height() float32
	ScaleW(w float32)
	ScaleH(h float32) */
	Draw()
}

type PanelRenderObject struct {
	Offset image.Point
	Center image.Point
	Size   image.Point
	Scale  image.Point
	Pane   *model.Pane
}

func PanelObject(p *model.Pane, o image.Point) PanelRenderObject {
	pp := p.Props
	off := image.Pt(int((pp.XOffset + pp.ZOffset)), int((pp.YOffset + pp.ZOffset)))
	off = off.Add(o)
	siz := image.Pt(int(pp.Width), int(pp.Height))
	ctr := image.Pt((off.X + ((siz.X) / 2)), (off.Y + ((siz.Y) / 2)))
	v := PanelRenderObject{
		Pane:   p,
		Offset: off,
		Size:   siz,
		Center: ctr,
	}

	return v
}

func (p *PanelRenderObject) Draw() {
	pp := p.Pane
	canvas := giu.GetCanvas()
	cv := pp.BaseColor()
	col := cv.ToRGBA()
	col.A = 20 - 5*pp.Index()

	end := image.Pt(p.Offset.X+p.Size.X, p.Offset.Y+p.Size.Y)
	canvas.AddRectFilled(p.Offset, end, col, 1, giu.DrawFlags_RoundCornersNone)

	for _, v := range pp.LedStrips {
		obj := StripObject(v, p.Offset)
		obj.Draw()
	}
}

type LedStripRenderObject struct {
	Offset image.Point
	Center image.Point
	Size   image.Point
	Scale  image.Point
	Strip  *model.LedStrip
}

func StripObject(s *model.LedStrip, o image.Point) LedStripRenderObject {
	pp := s.Props
	off := image.Pt(int((pp.XOffset)), int((pp.YOffset)))
	off = off.Add(o)
	siz := image.Pt(int(pp.Width), int(pp.Height))
	ctr := off.Add(siz.Div(2))
	v := LedStripRenderObject{
		Strip:  s,
		Offset: off,
		Size:   siz,
		Center: ctr,
	}

	return v
}

func (s *LedStripRenderObject) Draw() {
	ss := s.Strip
	canvas := giu.GetCanvas()
	cv := ss.BaseColor()
	col := cv.ToRGBA()
	col.A = 255 - uint8(ss.Props.ZOffset)*10
	alpha := float32(col.A) * 0.10
	col.A = uint8(alpha)
	end := s.Offset.Add(s.Size)
	canvas.AddRectFilled(s.Offset, end, col, 0, giu.DrawFlags_RoundCornersNone)

	for _, v := range ss.Strip {
		obj := LedObject(v, s.Offset)
		obj.Draw()
	}

}

type LedRenderObject struct {
	Offset image.Point
	Center image.Point
	Size   image.Point
	Scale  image.Point
	Led    *model.Led
}

func LedObject(l *model.Led, o image.Point) LedRenderObject {
	pp := l.Props
	off := image.Pt(int((pp.XOffset)), int((pp.YOffset)))
	off = off.Add(o)
	siz := image.Pt(int(pp.Width), int(pp.Height))
	ctr := off.Add(siz.Div(2))
	v := LedRenderObject{
		Led:    l,
		Offset: off,
		Size:   siz,
		Center: ctr,
	}

	return v
}

func (l *LedRenderObject) Draw() {
	ll := l.Led

	canvas := giu.GetCanvas()
	col := ll.Color.ToRGBA()
	col.A = uint8(math.Max(0, float64(col.A)-float64(ll.Props.ZOffset)*10.0))
	canvas.AddCircleFilled(l.Offset, float32(l.Size.X)/2, col)
}
