package imgui

import (
	"image"
	"image/color"
	"strconv"

	"github.com/AllenDang/giu"
	"github.com/coreman2200/funtimes-arcaluminis/model"
)

const (
	DFLT_WIDTH  = 750
	DFLT_HEIGHT = 550
)

type IMRenderWidget struct {
	id        string
	structure *model.LedStructure
	center    image.Point
	offset    image.Point
	width     int
	//pos       image.Point
	height int
}

func RenderWidget(s *model.LedStructure) IMRenderWidget {
	v := IMRenderWidget{
		id:        "Structure" + strconv.FormatUint(uint64(s.Index()), 10),
		structure: s,
	}

	v.width = DFLT_WIDTH
	v.height = DFLT_HEIGHT
	v.offset = image.Pt(0, 0)
	v.center = image.Pt(v.offset.X+(v.width/2), v.offset.Y+(v.height/2))

	return v
}

func (r *IMRenderWidget) Build() {
	giu.Layout{
		giu.Custom(r.Render),
	}.Build()

}

func (r *IMRenderWidget) Render() {

	canvas := giu.GetCanvas()
	bndMin := image.Pt(r.offset.X, r.offset.Y)
	bndMax := image.Pt(r.offset.X+r.width, r.offset.X+r.height)

	blk := color.RGBA{0, 0, 0, 0xFF}
	canvas.AddRectFilled(bndMin, bndMax, blk, 3, giu.DrawFlags_RoundCornersAll)

	for i := int(r.structure.PanelCount) - 1; i >= 0; i-- {
		obj := PanelObject(r.structure.Panel(i), bndMin)
		//obj.Offset = r.movToCenter(obj.Offset)
		obj.Draw()
	}

	// TODO Draw Render Window & Panels...
}

func (r *IMRenderWidget) relToCenter(p image.Point) image.Point {
	return image.Pt(p.X-r.center.X, p.Y-r.center.Y)
}

func (r *IMRenderWidget) movToCenter(p image.Point) image.Point {
	return image.Pt(p.X+r.center.X, p.Y+r.center.Y)
}

func (r *IMRenderWidget) Clear() {

}
