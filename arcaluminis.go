package main

import (
	"fmt"
	"image"
	"image/color"
	"log"

	. "github.com/coreman2200/funtimes-arcaluminis/model"

	"periph.io/x/conn/v3/display"
	"periph.io/x/conn/v3/physic"
	"periph.io/x/conn/v3/spi/spireg"
	"periph.io/x/devices/v3/nrzled"
	"periph.io/x/extra/devices/screen"
	"periph.io/x/host/v3"
)

var Options nrzled.Opts = nrzled.Opts{
	NumPixels: int(MaxPaneCount) * int(MaxPaneLedStripCount) * int(MaxLedStripLength),
	Channels:  3,
	Freq:      2500 * physic.KiloHertz,
}

func main() {
	if _, err := host.Init(); err != nil {
		log.Fatal(err)
	}
	d := getLEDs()
	ss := NewLedStructure()

	if err := d.Draw(d.Bounds(), ss.Image(), image.Point{}); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("\n")
}

func getLEDs() display.Drawer {
	s, err := spireg.Open("")
	if err != nil {
		fmt.Printf("Failed to find a SPI port, printing at the console:\n")
		return screen.New(100)
	}

	d, err := nrzled.NewSPI(s, &Options)
	if err != nil {
		log.Fatal(err)
	}
	return d
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
