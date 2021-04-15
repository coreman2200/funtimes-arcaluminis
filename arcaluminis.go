package main

import (
	"context"
	"fmt"
	"image/color"
	"log"
	"math"
	"os"
	"os/signal"
	"sync"
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

	ctx := context.Background()

	wg := &sync.WaitGroup{}
	wg.Add(1)

	// trap Ctrl+C and call cancel on the context
	ctx, cancel := context.WithCancel(ctx)
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	defer func() {
		signal.Stop(c)
		cancel()
	}()

	go func(cc context.Context) {
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
				cancel()
				wg.Done()
				return

			case sig := <-c:
				fmt.Printf("Got %s signal. Aborting...\n", sig)
				ticker.Stop()
				cancel()
				wg.Done()
				return

			case <-cc.Done():
				ticker.Stop()
				cancel()
				wg.Done()
				return
			}

		}
	}(ctx)

	wg.Wait()

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
