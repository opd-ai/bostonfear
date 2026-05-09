# Asset Pipeline Rollout Playbook

## Purpose

This document defines the staged rollout procedure and rollback process for the
YAML-driven visual asset pipeline introduced in Phase 3–6 of the graphics
quality upgrade plan.

---

## Feature Flag

The asset pipeline is controlled by a single environment variable:

| Variable | Value | Effect |
|---|---|---|
| `BOSTONFEAR_USE_LEGACY_ASSETS` | unset (default) | YAML manifest pipeline active |
| `BOSTONFEAR_USE_LEGACY_ASSETS` | any non-empty string | Legacy hardcoded pipeline active |

The flag is read once at first resolver construction and cached for the life of
the process. Changing the variable after startup has no effect. Restart the
client to pick up a new value.

---

## Staged Rollout Gates

### Stage 1 — Internal (developer machines)

Acceptance criteria:
- `go test -race ./...` exits 0.
- `go vet ./...` exits 0.
- Benchmarks run without error: `go test -bench=. ./client/ebiten/render/`
- Asset telemetry snapshot shows zero `ManifestParseErrors` and zero `AtlasBuildErrors`.
- Desktop client starts and renders all four neighbourhood locations.

Gate: one developer confirms Stage 1 criteria before proceeding.

### Stage 2 — Beta (limited external testers)

Acceptance criteria:
- All Stage 1 criteria hold.
- Beta testers confirm location labels, action panels, and resource HUD render
  correctly on at least two screen resolutions.
- No crash or hang reported within a 30-minute play session.
- Asset telemetry `FallbacksUsed` > 0 is investigated and explained before
  proceeding; unexplained fallbacks block the stage.

Gate: beta report with no unresolved P0/P1 issues.

### Stage 3 — Full release

Acceptance criteria:
- All Stage 2 criteria hold.
- `BOSTONFEAR_USE_LEGACY_ASSETS` is removed from all deployment configs.
- Legacy resolver code is retained but documented as deprecated.
- Release notes updated to describe YAML asset pipeline.

Gate: project maintainer sign-off.

---

## Rollback Procedure

If a regression is detected after enabling the YAML pipeline:

1. **Immediate mitigation**: set `BOSTONFEAR_USE_LEGACY_ASSETS=1` and restart
   the affected client. This activates `LegacyAtlasResolver` and bypasses the
   YAML manifest entirely.

2. **Verify rollback**: confirm via `AssetMetrics().Snapshot()` log output that
   `ManifestParseErrors` and `ComponentLoadFailures` are no longer incrementing.

3. **Diagnose**: check `render preflight:` log lines to identify which manifest
   keys or asset files triggered the failure.

4. **Fix forward or revert**: either correct the manifest and asset files, or
   revert the manifest/asset commit and run the full test suite before
   re-enabling the YAML pipeline.

5. **Clear the flag**: once the fix is validated, unset `BOSTONFEAR_USE_LEGACY_ASSETS`
   and restart the client to return to the YAML pipeline.

---

## Incident Checklist

- [ ] Flag set to legacy mode and client restarted.
- [ ] Asset telemetry confirms counters no longer increment.
- [ ] Root cause identified (manifest key, bad file path, parse error).
- [ ] Fix committed, tests pass, vet clean.
- [ ] Stage 1 acceptance criteria re-verified.
- [ ] Flag cleared, YAML pipeline re-enabled.
- [ ] Post-incident note added to CHANGELOG.md.

---

## Asset Telemetry Reference

The `render.AssetMetrics()` function returns a pointer to the package-level
`AssetTelemetry` struct. Call `AssetMetrics().LogSummary()` to emit a log line
with all current counter values, or `AssetMetrics().Snapshot()` to capture
values for health-check comparisons.

| Counter | Meaning | Healthy value |
|---|---|---|
| `ManifestParseErrors` | YAML decode or validation failures | 0 |
| `ComponentLoadFailures` | Individual asset files that could not be loaded | 0 |
| `FallbacksUsed` | Placeholder substitutions applied at startup | 0 |
| `AtlasBuildErrors` | Failures building the final sprite atlas PNG | 0 |

Any non-zero value after a clean startup indicates missing or malformed assets
and should be investigated before promoting to the next rollout stage.
