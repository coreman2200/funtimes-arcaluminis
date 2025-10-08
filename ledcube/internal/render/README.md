# `internal/render` — Engine & Renderer API

**Intent:** Provide a tiny, fast render engine with a stable plugin API for renderers, dual-buffer crossfades, and a post-processing hook (tone map + limiter).

## What’s here
- `types.go` — shared types (`Vec3`, `Color`, `Dimensions`, `Uniforms`, `Resources`, `Renderer`, `Registry`)
- `engine.go` — `Engine` with `RenderOnce`, crossfade hooks (`SetRenderer`, `ArmNext`, `SetCrossfade`), param updates.
- `mix.go` — framebuffer mix utility.
- `post.go` — default tone map (gamma 2.2) + limiter hook (no-op by default).
- `engine_test.go` — fake renderer/driver tests for mix & crossfade.

## Wiring to the Sequencer
Map Sequencer Hooks → Engine methods:
- `SetRenderer(name,preset)` → `Engine.SetRenderer(name,preset, registry)`
- `ArmNext(name,preset)` → `Engine.ArmNext(name,preset, registry)`
- `SetCrossfade(alpha)` → `Engine.SetCrossfade(alpha)`
- `SetParam/SetBool` → `Engine.SetParam`, `Engine.SetBool`

Then, in your main loop (or a goroutine), call:
```go
for {
    _ = engine.RenderOnce(-1) // uses Engine.Now()
}
```
(Or feed an explicit `t` if you’re running in a fixed-step simulation.)

## Notes
- Colors are **linear** [0,1]. Tone-mapping applies gamma by default; replace `DefaultToneMap` if you want a filmic curve.
- Add your power limiter by setting `Engine.post.Limiter`.
- Crossfade promotes `RNext` → `RActive` when alpha reaches 1.0.
