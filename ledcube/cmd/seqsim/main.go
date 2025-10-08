package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"time"

	"internal/sequence"
)

func main() {
	var programPath string
	var fps int
	flag.StringVar(&programPath, "program", "", "Path to Program JSON (seq.v1)")
	flag.IntVar(&fps, "fps", 60, "Simulation frames per second")
	flag.Parse()

	if programPath == "" {
		log.Fatal("Provide -program path to a Program JSON")
	}

	data, err := ioutil.ReadFile(programPath)
	if err != nil {
		log.Fatalf("read program: %v", err)
	}

	// Parse minimally: We only need fields used by the player;
	// Envelopes/Params are typically loaded by your higher-level config layer.
	type ProgIn struct {
		Version string `json:"version"`
		Loop    bool   `json:"loop"`
		Seed    int64  `json:"seed"`
		Clips   []struct {
			Name      string  `json:"name"`
			Renderer  string  `json:"renderer"`
			Preset    string  `json:"preset"`
			DurationS float64 `json:"durationS"`
			XFadeS    float64 `json:"xFadeS"`
		} `json:"clips"`
	}
	var in ProgIn
	if err := json.Unmarshal(data, &in); err != nil {
		log.Fatalf("json: %v", err)
	}

	var prog sequence.Program
	prog.Version = in.Version
	prog.Loop = in.Loop
	prog.Seed = in.Seed
	for _, c := range in.Clips {
		prog.Clips = append(prog.Clips, sequence.Clip{
			Name:      c.Name,
			Renderer:  c.Renderer,
			Preset:    c.Preset,
			DurationS: c.DurationS,
			XFadeS:    c.XFadeS,
			Params:    map[string]sequence.Envelope{},
			Bools:     map[string]sequence.Envelope{},
		})
	}

	// simple logger hooks
	h := sequence.Hooks{
		SetRenderer: func(name, preset string) {
			fmt.Printf("[SetRenderer] %s / %s\n", name, preset)
		},
		ArmNext: func(name, preset string) {
			fmt.Printf("[ArmNext] %s / %s\n", name, preset)
		},
		SetCrossfade: func(alpha float64) {
			fmt.Printf("[Crossfade] alpha=%.3f\n", alpha)
		},
		SetParam: func(name string, v float64) {},
		SetBool: func(name string, b bool) {},
	}
	player := sequence.NewPlayer(h)
	if err := player.Load(prog); err != nil {
		log.Fatalf("load: %v", err)
	}
	player.Start()

	dt := time.Second / time.Duration(fps)
	ticker := time.NewTicker(dt)
	defer ticker.Stop()

	start := time.Now()
	for {
		select {
		case <-ticker.C:
			player.Tick(dt.Seconds())
			elapsed := time.Since(start).Seconds()
			// End when player returns to Idle (no loop)
			if player.State == sequence.Idle {
				fmt.Println("Done at t=", elapsed)
				os.Exit(0)
			}
		}
	}
}
