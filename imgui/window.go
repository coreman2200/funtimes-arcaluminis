package imgui

import (
	"fmt"
	"math"
	"time"

	"github.com/AllenDang/giu"
	im "github.com/AllenDang/giu/imgui"
	"github.com/coreman2200/funtimes-arcaluminis/model"
)

const (
	Title     = "Led Sim"
	WinWidth  = 800
	WinHeight = 600
	Ratio     = WinWidth / WinHeight
	Dpi       = 10
	DFLT_FPS  = 30
)

type IMWindow struct {
	Win      *giu.MasterWindow
	leds     *model.LedStructure
	start    time.Time
	duration time.Duration
	fps      float32
}

func (w *IMWindow) refresh() {
	delta := 1000 * time.Millisecond / time.Duration(DFLT_FPS)
	ticker := time.NewTicker(delta)

	fd := float32(delta)

	for {
		select {
		case <-ticker.C:
			t := time.Now()
			w.duration = t.Sub(w.start)

			w.leds.Update(w.duration)
			giu.Update()

			w.fps = (float32(delta) / fd) * DFLT_FPS
			delta = time.Duration(fd) - time.Since(t)
			if delta.Milliseconds() > 0 {
				ticker.Stop()
				ticker = time.NewTicker(delta)
			}

		}
	}
}

func (w *IMWindow) loop() {
	dur := float64(w.duration.Milliseconds())
	nms := float64(dur) / 1000
	rad := float64(360*nms/180.0) * math.Pi
	//rad := math.Mod(ang, 2*math.Pi)
	//fmt.Println("Radians: " + strconv.FormatFloat(rad, 'f', -1, 32))

	rr := RenderWidget(w.leds)
	giu.SingleWindow("Update").Layout(
		&rr,
		giu.Label(fmt.Sprintf("FPS: %f", w.fps)),
		giu.Label(fmt.Sprintf("Duration: %f", dur/1000)),
		giu.Label(fmt.Sprintf("Radians: %f", rad)),
	)
}

func loadFont() {
	fonts := giu.Context.IO().Fonts()
	fontPath := "/Users/new/Library/Fonts/InconsolataGo Nerd Font Complete.ttf"
	fonts.AddFontFromFileTTFV(fontPath, 16, im.DefaultFontConfig, fonts.GlyphRangesDefault())
}

func (s *IMWindow) Start() {
	go s.refresh()

	s.Win.Run(s.loop)
}

func NewIMWindow(s *model.LedStructure) *IMWindow {
	v := IMWindow{
		Win:   giu.NewMasterWindow(Title, WinWidth, WinHeight, giu.MasterWindowFlagsNotResizable, loadFont),
		leds:  s,
		start: time.Now(),
	}

	return &v
}
