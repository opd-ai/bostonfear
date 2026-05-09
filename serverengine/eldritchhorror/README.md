# Eldritch Horror Module (Boilerplate)

This package is a scaffolding root for a future Eldritch Horror implementation.

## Current Status

- Module registration exists.
- Runtime is a placeholder returning not implemented.

## Planned Structure

- model/: world map, investigators, encounters, mysteries
- rules/: action resolution, travel, condition effects
- scenarios/: ancient one setups and expansion variants
- adapters/: bindings to serverengine/common runtime contracts

## Dependency Direction

- Allowed: eldritchhorror -> serverengine/common/*
- Prohibited: serverengine/common/* -> eldritchhorror
