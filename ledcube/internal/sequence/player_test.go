package sequence

import "testing"

func TestEnvelopeEval(t *testing.T) {
	env := Envelope{Keys: []Keyframe{
		{T: 0, V: 0, Ease: "linear"},
		{T: 10, V: 10, Ease: "linear"},
	}}
	if v := env.Eval(-1); v != 0 {
		t.Fatalf("expected 0 before start, got %v", v)
	}
	if v := env.Eval(0); v != 0 {
		t.Fatalf("expected 0 at t=0, got %v", v)
	}
	if v := env.Eval(5); v != 5 {
		t.Fatalf("expected 5 at t=5, got %v", v)
	}
	if v := env.Eval(10); v != 10 {
		t.Fatalf("expected 10 at t=10, got %v", v)
	}
	if v := env.Eval(11); v != 10 {
		t.Fatalf("expected 10 after end, got %v", v)
	}
}

func TestSequencerCrossfade(t *testing.T) {
	log := []string{}
	h := Hooks{
		SetRenderer: func(name, preset string) { log = append(log, "Set:"+name+"/"+preset) },
		ArmNext:     func(name, preset string) { log = append(log, "Arm:"+name+"/"+preset) },
		SetCrossfade: func(a float64) {
			// log a few key alphas
			if a == 0 || a == 0.5 || a == 1.0 {
				log = append(log, "Alpha")
			}
		},
		SetParam: func(name string, v float64) {},
		SetBool:  func(name string, b bool) {},
	}
	p := NewPlayer(h)
	prog := Program{
		Version: "seq.v1",
		Loop:    false,
		Clips: []Clip{
			{Name: "A", Renderer: "ocean", Preset: "Calm", DurationS: 4, XFadeS: 2},
			{Name: "B", Renderer: "warp", Preset: "Blue", DurationS: 4, XFadeS: 0},
		},
	}
	if err := p.Load(prog); err != nil {
		t.Fatalf("load: %v", err)
	}
	p.Start()
	// advance to just before fade
	p.Tick(1.9) // t=1.9
	// in middle of fade window (A's fade starts at t=2.0)
	p.Tick(0.2) // t=2.1 -> should arm B
	p.Tick(0.9) // t=3.0
	p.Tick(1.0) // t=4.0 -> should switch to B

	// Simple assertions: Arm happens after initial Set to A (ignore alpha logs)
	var cleaned []string
	for _, entry := range log {
		if entry == "Alpha" {
			continue
		}
		cleaned = append(cleaned, entry)
	}
	wantOrder := []string{"Set:ocean/Calm", "Arm:warp/Blue"}
	if len(cleaned) < len(wantOrder) {
		t.Fatalf("unexpected log order: %#v", log)
	}
	for i := range wantOrder {
		if cleaned[i] != wantOrder[i] {
			t.Fatalf("unexpected log order: %#v", log)
		}
	}
}
