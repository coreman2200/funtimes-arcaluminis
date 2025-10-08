# Arcaluminis Pipeline (Sequencer → Renderer → Driver)

```
[Sequencer Program]
      │  (clips, envelopes)
      ▼
[Player] ──hooks──▶ [Render Engine]
                       │
                       ├── Active Renderer (A)
                       ├── Armed Renderer (B)   ← for crossfade
                       └── Post: ToneMap → Limiter → Driver(SPI)
```

- **Sequencer**: chooses *what* plays when, automates parameters, and emits a crossfade alpha near clip boundaries.
- **Render Engine**: calls the active `Renderer` to fill a framebuffer; during fades renders both A and B, mixes by alpha, then applies tone mapping and power limiting before writing to LEDs.
- **UI/Wails**: sends JSON control messages (`setRenderer`, `setParams`, `loadSequence`, `start`, `pause`, `seek`), subscribes to status/fps.

This drop ships the **Sequencer MVP** and a minimal **Renderer registry** stub. Next drops will fill in `engine.go`, post pipeline, and Wails handlers.
