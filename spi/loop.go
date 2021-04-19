package spi

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/coreman2200/funtimes-arcaluminis/model"
)

const DFLT_FPS = 30

type Looper struct {
	quit     chan bool
	ctx      context.Context
	cancel   context.CancelFunc
	wg       *sync.WaitGroup
	c        chan os.Signal
	start    time.Time
	leds     *model.LedStructure
	renderer SPILedRenderer
}

func (l *Looper) refresh() {
	delta := 1000 * time.Millisecond / time.Duration(DFLT_FPS)
	ticker := time.NewTicker(delta)

	fd := float32(delta)

	for {
		select {
		case <-ticker.C:
			t := time.Now()
			duration := t.Sub(l.start)
			l.leds.Update(duration)
			l.renderer.Render()

			//fps := (float32(delta) / fd) * DFLT_FPS
			delta = time.Duration(fd) - time.Since(t)
			if delta.Milliseconds() > 0 {
				ticker.Stop()
				ticker = time.NewTicker(delta)
			}

		case <-l.quit:
			ticker.Stop()
			l.cancel()
			l.wg.Done()
			return

		case sig := <-l.c:
			fmt.Printf("Got %s signal. Aborting...\n", sig)
			ticker.Stop()
			l.cancel()
			l.wg.Done()
			return

		case <-l.ctx.Done():
			ticker.Stop()
			l.cancel()
			l.wg.Done()
			return
		}

	}
}

func InitSPILooper(s *model.LedStructure) (Looper, bool) {
	r := InitLedRenderer(s)
	v := Looper{
		leds:     s,
		renderer: r,
	}

	return v, r.Spi
}

func (l *Looper) Start() {
	l.quit = make(chan bool)

	l.ctx = context.Background()
	l.ctx, l.cancel = context.WithCancel(l.ctx)

	l.wg = &sync.WaitGroup{}
	l.wg.Add(1)

	l.c = make(chan os.Signal, 1)
	signal.Notify(l.c, os.Interrupt)
	defer func() {
		signal.Stop(l.c)
		l.cancel()
	}()

	l.start = time.Now()
	go l.refresh()

	l.wg.Wait()
}
