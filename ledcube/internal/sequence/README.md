# `internal/sequence` — Show Sequencer

**Intent:** Orchestrate which renderer runs when, with param automation and crossfades, deterministically.

## Status
MVP implemented:
- Program/Clip/Envelope data model
- Easing (`linear`, `smooth`, `cubic`)
- Player with `Load/Start/Pause/Resume/Stop/Seek/Tick`
- Crossfade pre-arming + alpha emission
- Unit tests for envelopes & basic crossfade behavior
- CLI `cmd/seqsim` example

## Quick Start
```bash
# inside your module
go test ./internal/sequence -v

# run the simulator
go run ./cmd/seqsim -program ./docs/examples/seq-demo.json
```

## Hooks contract
The sequencer does not import your engine. Provide callbacks:
- `SetRenderer(name, preset)` — switch active renderer immediately.
- `SetParam(name, v)` / `SetBool(name, b)` — update active renderer controls.
- `ArmNext(name, preset)` — prepare next renderer for crossfade.
- `SetCrossfade(alpha)` — 0..1 mix between active and armed.
