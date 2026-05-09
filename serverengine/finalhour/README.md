# Final Hour Module (Boilerplate)

This package is a scaffolding root for a future Final Hour implementation.

## Current Status

- Module registration exists.
- Runtime is a placeholder returning not implemented.

## Planned Structure

- model/: investigators, objective tracks, map state
- rules/: simultaneous action programming and resolution
- scenarios/: setup variants and difficulty presets
- adapters/: bindings to serverengine/common runtime contracts

## Dependency Direction

- Allowed: finalhour -> serverengine/common/*
- Prohibited: serverengine/common/* -> finalhour
