# Fake renderers + conductor simulator

This pack gives you two minimal renderers and a CLI to prove out the **Sequencer ⇄ Engine** wiring without Wails or hardware.

## What’s included
- `internal/render/fake/solid/solid.go` — solid color with presets and optional `PulseHz` param.
- `internal/render/fake/grad/grad.go` — spatial gradient with simple "Rainbow" motion.
- `internal/driver/fake/driver.go` — prints average/first pixel per frame.
- `cmd/conductorsim/main.go` — runs a two-clip looping program and logs set/arm/alpha + frame summaries.

## Run it
```bash
go run ./cmd/conductorsim
```
You should see logs like:
```
SetRenderer: solid Red
[frame 0001] avg=(0.45,0.00,0.00) ...
Alpha: 0.00
...
ArmNext: grad Rainbow
Alpha: 0.33
Alpha: 0.67
...
SetRenderer: grad Rainbow
...
```

## Looping one render infinitely (options)

**Simplest (no code changes):**
- Use the sequencer with a single clip and `Loop: true`. Set `XFadeS: 0` to avoid any transition at the loop boundary, and choose a long `DurationS` (e.g., hours). Example:
```json
{ "version":"seq.v1", "loop": true,
  "clips":[ { "name":"SolidWhite", "renderer":"solid", "preset":"White", "durationS": 36000, "xFadeS": 0 } ]
}
```

**Programmatic infinite (tiny patch):**
- Treat `DurationS <= 0` as "hold forever". In `sequence.Player.Tick` just before the "Clip end?" check, insert:
```go
// If this clip is infinite (DurationS<=0), never advance.
if clip.DurationS <= 0 {
    return
}
```
(You can also gate it on a boolean env like `clip.Bools[\"Hold\"]` if you prefer a togglable hold.)

**Renderer-level infinite:**
- Renderers are already time-driven and run indefinitely. If you want to bypass the sequencer entirely for a single renderer, just set it once:
```go
_ = eng.SetRenderer("grad", "Rainbow", reg)
for { _ = eng.RenderOnce(-1) }
```
