package spi

import (
	"fmt"
	"image"
	"log"

	"github.com/coreman2200/funtimes-arcaluminis/model"
	"periph.io/x/conn/v3/display"
	"periph.io/x/conn/v3/physic"
	"periph.io/x/conn/v3/spi/spireg"
	"periph.io/x/devices/v3/nrzled"
	"periph.io/x/extra/devices/screen"
	"periph.io/x/host/v3"
)

type SPILedRenderer struct {
	Structure *model.LedStructure
	drawer    display.Drawer
	Spi       bool
}

func (r *SPILedRenderer) Render() {
	if err := r.drawer.Draw(r.drawer.Bounds(), r.Structure.Image(), image.Point{}); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("\n")
}

func (r *SPILedRenderer) Clear() {
	if err := r.drawer.Halt(); err != nil {
		log.Fatal(err)
	}
}

func InitLedRenderer(s *model.LedStructure) SPILedRenderer {
	rr := SPILedRenderer{
		Structure: s,
	}
	if _, err := host.Init(); err != nil {
		log.Fatal(err)
	}
	ss, err := spireg.Open("")
	if err != nil {
		fmt.Printf("Failed to find a SPI port, printing at the console:\n")
		rr.drawer = screen.New(100)
		rr.Spi = false
		return rr
	}

	rr.Spi = true

	var Options nrzled.Opts = nrzled.Opts{
		NumPixels: len(s.Leds()),
		Channels:  3,
		Freq:      ((model.RefreshRate * 3) + 100) * physic.KiloHertz,
	}

	d, err := nrzled.NewSPI(ss, &Options)
	if err != nil {
		log.Fatal(err)
	}
	d.Halt()
	rr.drawer = d
	return rr
}
