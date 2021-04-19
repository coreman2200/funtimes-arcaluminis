package main

import (
	"github.com/coreman2200/funtimes-arcaluminis/imgui"
	"github.com/coreman2200/funtimes-arcaluminis/model"
	"github.com/coreman2200/funtimes-arcaluminis/spi"
)

const DFLT_FPS = 30

type DisplayInterface interface {
	Start()
}

func main() {

	ss := model.NewLedStructure()

	ll, spi := spi.InitSPILooper(ss)
	if spi {
		ll.Start()
	} else {
		gui := imgui.NewIMWindow(ss)
		gui.Start()
	}

	//defer ss.Clear()

}
