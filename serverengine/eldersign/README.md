# Elder Sign Module (Boilerplate)

This package is a scaffolding root for a future Elder Sign implementation.

## Current Status

- Module registration exists.
- Runtime is a placeholder returning not implemented.

## Planned Structure

- model/: Elder Sign domain types and constants
- rules/: action and resolution rules
- scenarios/: setup presets and expansions
- adapters/: bindings to serverengine/common runtime contracts

## Dependency Direction

- Allowed: eldersign -> serverengine/common/*
- Prohibited: serverengine/common/* -> eldersign
