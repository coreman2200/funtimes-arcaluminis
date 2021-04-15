package main

import (
	"image/color"
	"log"
	"math"
	"time"

	"github.com/coreman2200/funtimes-arcaluminis/model"

	"periph.io/x/host/v3"
)

const DFLT_FPS = 30

func main() {
	if _, err := host.Init(); err != nil {
		log.Fatal(err)
	}
	ss := model.NewLedStructure()
	defer ss.Clear()

	quit := make(chan bool)

	go func() {
		start := time.Now()

		delta := time.Millisecond / time.Duration(DFLT_FPS)
		ticker := time.NewTicker(delta)

		for {

			select {
			case <-ticker.C:
				t := time.Now()
				duration := t.Sub(start)
				rad := math.Mod((float64(duration.Milliseconds())*math.Pi)/180.0, 2*math.Pi)
				//log.Println("Radians: " + strconv.FormatFloat(rad, 'f', -1, 32))

				// Orient, Transform, Scale, Projection..
				ss.TestManip(rad)
				ss.Draw()

				//fps := (DFLT_FPS * (float32(delta.Milliseconds())/1000.0))
				delta = time.Millisecond/time.Duration(DFLT_FPS) - time.Since(t)
				if delta.Milliseconds() > 0 {
					ticker.Stop()
					ticker = time.NewTicker(delta)
				}

			case <-quit:
				ticker.Stop()
				return
			}

		}
	}()

	time.Sleep(10 * time.Second)

	log.Println("stopping ticker...")
	quit <- true

	time.Sleep(500 * time.Millisecond)

}

func colorWheel(h float64) color.NRGBA {
	h *= 6
	switch {
	case h < 1.:
		return color.NRGBA{R: 255, G: byte(255 * h), A: 255}
	case h < 2.:
		return color.NRGBA{R: byte(255 * (2 - h)), G: 255, A: 255}
	case h < 3.:
		return color.NRGBA{G: 255, B: byte(255 * (h - 2)), A: 255}
	case h < 4.:
		return color.NRGBA{G: byte(255 * (4 - h)), B: 255, A: 255}
	case h < 5.:
		return color.NRGBA{R: byte(255 * (h - 4)), B: 255, A: 255}
	default:
		return color.NRGBA{R: 255, B: byte(255 * (6 - h)), A: 255}
	}
}
