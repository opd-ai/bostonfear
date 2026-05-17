# Final Hour Module

This package provides a runnable Final Hour module MVP that currently reuses the
shared serverengine gameplay runtime while Final Hour-specific rules are expanded.

## Current Status

- Module registration exists.
- Runtime is functional and returns a module-owned engine wrapper.
- Selectable via `BOSTONFEAR_GAME=finalhour`.

## Planned Structure

- model/: investigators, objective tracks, map state
- rules/: simultaneous action programming and resolution
- scenarios/: setup variants and difficulty presets
- adapters/: bindings to serverengine/common runtime contracts

## Dependency Direction

- Allowed: finalhour -> serverengine/common/*
- Prohibited: serverengine/common/* -> finalhour
