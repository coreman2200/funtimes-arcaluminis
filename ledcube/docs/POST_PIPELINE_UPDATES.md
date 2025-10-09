# Post pipeline updates (filmic + limiter)

**Filmic tone mapping** (ACES approx) with `ExposureEV` and `OutputGamma` params.
**Default power limiter** with per-LED WhiteCap and global mA budget (soft knee).

### Params (Uniforms.Params)
- `ExposureEV` (float, default 0): scene exposure in EV.
- `OutputGamma` (float, default 2.2): final gamma encode.
- `WhiteCap` (float, default 3.0): cap on `R+G+B` per voxel.
- `LEDChan_mA` (float, default 20): mA per color channel at full scale.
- `Budget_mA` (float, default 0): if > 0, enable budget limiter.
- `LimiterKnee` (float 0..1, default 0.9): fraction of budget where soft limiting begins.

### Hooking
Use as your engine's post:
```go
engpost := render.PostPipeline{
    ToneMap: func(buf []render.Color){ render.FilmicToneMap(buf, eng.UActive) },
    Limiter: render.DefaultLimiter,
}
```
