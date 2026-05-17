# Eldritch Horror Module

This package provides a runnable Eldritch Horror module MVP that currently reuses
the shared serverengine gameplay runtime while Eldritch-specific rules are expanded.

## Current Status

- Module registration exists.
- Runtime is functional and returns a module-owned engine wrapper.
- Selectable via `BOSTONFEAR_GAME=eldritchhorror`.

## Planned Structure

- model/: world map, investigators, encounters, mysteries
- rules/: action resolution, travel, condition effects
- scenarios/: ancient one setups and expansion variants
- adapters/: bindings to serverengine/common runtime contracts

## Dependency Direction

- Allowed: eldritchhorror -> serverengine/common/*
- Prohibited: serverengine/common/* -> eldritchhorror
