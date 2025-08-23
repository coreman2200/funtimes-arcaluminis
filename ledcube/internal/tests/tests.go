
package tests

import "github.com/example/ledcube-wails/internal/layout"

type Kind string
const (
    None Kind = ""
    IndexSweep Kind = "index_sweep"
    RGBTest Kind = "rgb_channels"
    PlaneZ Kind = "plane_z"
)

type Plan struct { Kind Kind }

type Runner struct {
    plan Plan
    step int
}

func NewRunner(plan Plan) *Runner { return &Runner{plan: plan} }
func (r *Runner) Kind() Kind { return r.plan.Kind }

// Step fills rgb; returns false when complete.
func (r *Runner) Step(l layout.Layout, rgb []byte) bool {
    n := l.Count()
    for i := 0; i < n*3; i++ { rgb[i] = 0 }

    switch r.plan.Kind {
    case IndexSweep:
        idx := r.step
        if idx >= n { return false }
        rgb[idx*3+0], rgb[idx*3+1], rgb[idx*3+2] = 255, 255, 255
    case RGBTest:
        phase := r.step % 3
        for i := 0; i < n; i++ {
            switch phase {
            case 0: rgb[i*3+0] = 255
            case 1: rgb[i*3+1] = 255
            case 2: rgb[i*3+2] = 255
            }
        }
    case PlaneZ:
        perPanel := l.Dim.X * l.Dim.Y
        z := r.step
        if z >= l.Dim.Z { return false }
        for i := z*perPanel; i < (z+1)*perPanel; i++ {
            rgb[i*3+1], rgb[i*3+2] = 255, 255 // cyan
        }
    default:
        return false
    }
    r.step++
    return true
}
