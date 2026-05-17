# Elder Sign Module

This package provides a runnable Elder Sign module MVP that currently reuses the
shared serverengine gameplay runtime while Elder Sign-specific rules are expanded.

## Current Status

- Module registration exists.
- Runtime is functional and returns a module-owned engine wrapper.
- Selectable via `BOSTONFEAR_GAME=eldersign`.

## Planned Structure

- model/: Elder Sign domain types and constants
- rules/: action and resolution rules
- scenarios/: setup presets and expansions
- adapters/: bindings to serverengine/common runtime contracts

## Dependency Direction

- Allowed: eldersign -> serverengine/common/*
- Prohibited: serverengine/common/* -> eldersign
