package render

type Vec3 struct{ X, Y, Z float64 }
type Color struct{ R, G, B float32 }

type Dimensions struct{ X, Y, Z int }

type Uniforms struct {
	GlobalBrightness float64
	TimeScale        float64
	SunDir, MoonDir  Vec3
	Params           map[string]float64
	Bools            map[string]bool
}

type Resources struct {
	Voxels   [][]uint8
	Audio    interface{}
	Sensors  map[string]float64
	LUTs     interface{}
}

type Renderer interface {
	Name() string
	Presets() []string
	ApplyPreset(name string, u *Uniforms)
	Render(dst []Color, pLUT []Vec3, dim Dimensions, t float64, u *Uniforms, r *Resources)
}

type Registry struct{ m map[string]Renderer }

func NewRegistry() *Registry { return &Registry{m: map[string]Renderer{}} }

func (r *Registry) Register(rr Renderer) {
	if rr == nil {
		return
	}
	r.m[rr.Name()] = rr
}

func (r *Registry) Get(name string) (Renderer, bool) { rr, ok := r.m[name]; return rr, ok }
func (r *Registry) List() []string {
	out := make([]string, 0, len(r.m))
	for k := range r.m {
		out = append(out, k)
	}
	return out
}
