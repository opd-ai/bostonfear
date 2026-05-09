# Asset YAML Schema Specification

This document satisfies Phase 2 item "YAML schema drafted and approved" from PLAN.md.

## Version

- Schema version: `1`
- Format: YAML
- Target path convention: relative file paths under a configured base path

## Root Shape

```yaml
content:
  visuals:
    version: 1
    basePath: assets/png
    placeholders:
      missing: ui/missing.png
    components:
      board.background:
        file: board/board_main.png
        scaleMode: cover
```

## Fields

| Field | Type | Required | Description |
|---|---|---|---|
| `content.visuals.version` | integer | yes | Schema version for compatibility checks.
| `content.visuals.basePath` | string | yes | Root directory for component asset files.
| `content.visuals.placeholders.missing` | string | yes | Fallback asset when component file is missing/unreadable.
| `content.visuals.components` | map | yes | Map of component keys to asset definitions.

## Component Entry

```yaml
components:
  ui.action.button.default:
    file: ui/actions/default.png
    hover: ui/actions/hover.png
    pressed: ui/actions/pressed.png
    disabled: ui/actions/disabled.png
    scaleMode: fit
    anchor: center
```

| Component Field | Type | Required | Description |
|---|---|---|---|
| `file` | string | yes | Primary PNG path relative to `basePath`.
| `hover` | string | no | Optional hover-state PNG.
| `pressed` | string | no | Optional pressed-state PNG.
| `disabled` | string | no | Optional disabled-state PNG.
| `scaleMode` | string enum | no | `fit`, `cover`, or `stretch`.
| `anchor` | string enum | no | `topLeft`, `top`, `topRight`, `left`, `center`, `right`, `bottomLeft`, `bottom`, `bottomRight`.

## Validation Rules

1. `version` must be `1`.
2. `basePath` must not be empty.
3. `placeholders.missing` must not be empty.
4. `components` must contain at least one key.
5. Every component entry must include non-empty `file`.
6. Component keys must be unique.
7. File paths must be relative and end with `.png`.
8. Absolute paths and parent traversal (`..`) are invalid.
9. If present, `scaleMode` must be one of: `fit`, `cover`, `stretch`.
10. If present, `anchor` must be one of the defined anchor enums.

## Initial Required Component Keys

The following keys are required in the first integration pass:

- `board.background`
- `location.downtown`
- `location.university`
- `location.rivertown`
- `location.northside`
- `token.investigator.default`
- `hud.doom.marker`
- `ui.action.button.default`

## Approval Note

This schema is approved as the baseline contract for implementing the loader, resolver, and fallback behavior in subsequent Phase 2 tasks.