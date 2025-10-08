# ADR-0001: Crossfade Strategy

**Decision:** Use frame-buffer alpha crossfade between current and next renderers during clip transitions.

**Context:** Renderers can have arbitrary internal state and parameter spaces; morphing parameters across different renderers is undefined and brittle. A mix at the framebuffer is renderer-agnostic, predictable, and cheap at our voxel counts.

**Consequences:**
- Engine must support dual rendering during fade windows and a per-frame mix.
- Sequencer provides a single `alpha` (0..1); engine handles the rest.
- Renderers don't need to know about transitions and can stay pure.
