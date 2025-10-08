package sequence

// Keyframe represents a value at time T (seconds) with an easing function
// that applies to the segment starting at this keyframe.
type Keyframe struct {
	T    float64 `json:"t"`
	V    float64 `json:"v"`
	Ease string  `json:"ease,omitempty"` // "linear","smooth","cubic"
}

// Envelope is a sorted list of keyframes; Eval(t) interpolates a value.
type Envelope struct {
	Keys []Keyframe `json:"-"` // populated after unmarshal or by code
}

// Clip is one segment of a show: selects a renderer + preset, sets duration,
// optional crossfade into the NEXT clip, and controls parameter automation.
type Clip struct {
	Name      string               `json:"name"`
	Renderer  string               `json:"renderer"`
	Preset    string               `json:"preset,omitempty"`
	DurationS float64              `json:"durationS"`
	XFadeS    float64              `json:"xFadeS,omitempty"`
	Params    map[string]Envelope  `json:"-"` // numeric params over time
	Bools     map[string]Envelope  `json:"-"` // 0..1 thresholded to bool
}

// Program is a full sequence of clips.
type Program struct {
	Version string `json:"version"` // e.g., "seq.v1"
	Loop    bool   `json:"loop,omitempty"`
	Seed    int64  `json:"seed,omitempty"`
	Clips   []Clip `json:"clips"`
}

// PlayerState enumerates sequencer states.
type PlayerState string

const (
	Idle    PlayerState = "idle"
	Running PlayerState = "running"
	Paused  PlayerState = "paused"
)

// Hooks are dependency-injected callbacks into the render engine.
type Hooks struct {
	// Set active renderer/preset immediately.
	SetRenderer func(name, preset string)
	// Parameter and boolean setters for the ACTIVE renderer.
	SetParam func(name string, v float64)
	SetBool  func(name string, b bool)
	// Prepare the next renderer/preset for crossfade.
	ArmNext      func(name, preset string)
	SetCrossfade func(alpha float64) // 0..1 mix between active and armed
}

// Player owns the current Program timeline and uses Hooks to drive the engine.
type Player struct {
	State PlayerState

	prog Program
	nowS float64 // position within program
	idx  int     // current clip index

	// crossfade bookkeeping
	armedIndex   int  // which clip is armed next (-1 means none)
	armed        bool // whether next is armed
	lastAlpha    float64

	// injection
	hooks Hooks

	// control
	fixedDT bool
}
