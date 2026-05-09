# Resolution Support Matrix

This document satisfies Phase 1 item "Resolution matrix approved" from PLAN.md.

## Current Implementation Baseline

- Logical game resolution: `800x600` (`client/ebiten/app/game.go`).
- Desktop startup window: `800x600`, resizable enabled (`cmd/desktop.go`).
- WASM startup window: `800x600` (`cmd/web_wasm.go`).
- Rendering model: fixed logical canvas with Ebitengine scaling to outside size.

## Target Matrix

| Profile | Outside Resolution Example | Aspect Ratio | Current Support Status | Notes |
|---|---|---|---|---|
| Minimum Supported | `800x600` | 4:3 | Approved | Explicit project minimum and test baseline.
| Desktop Baseline | `1280x720` | 16:9 | Approved | Scales from fixed logical surface.
| Full HD | `1920x1080` | 16:9 | Approved | Scaled rendering; readability validation required each release.
| QHD | `2560x1440` | 16:9 | Approved | Scaled rendering expected; monitor text readability.
| Ultrawide | `2560x1080` | 21:9 | Conditionally approved | Works via scaling; letterboxing/panel adaptation tracked in Phase 4.
| Mobile Landscape | `1334x750` to `2532x1170` | ~16:9 to ~19.5:9 | Conditionally approved | Touch parity and safe-area behavior validated in mobile runbook.

## Approval Notes

1. `800x600` is the minimum acceptance gate for all gameplay-critical text and controls.
2. Any viewport above minimum relies on Ebitengine scaling from the fixed logical surface.
3. Phase 4 remains responsible for formal breakpoint assertions and extreme-aspect layout handling.