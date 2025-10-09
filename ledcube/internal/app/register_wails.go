//go:build wails

package app

import (
	"github.com/coreman2200/funtimes-arcaluminis/ledcube/internal/render"
	grad "github.com/coreman2200/funtimes-arcaluminis/ledcube/internal/render/fake/grad"
	solid "github.com/coreman2200/funtimes-arcaluminis/ledcube/internal/render/fake/solid"
)

func registerDefaultRenderers(reg *render.Registry) {
	reg.Register(solid.New("solid", render.Color{R: 1}))
	reg.Register(grad.New("grad"))
}
