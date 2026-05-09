# Arkham Horror Module

This package tree contains Arkham Horror-specific engine logic layered on top of shared runtime primitives in serverengine/common.

## Responsibilities

- Own Arkham Horror rules, phases, and scenario setup.
- Bind Arkham actions and validation to the shared runtime pipeline.
- Keep all Arkham constants and map topology out of common packages.

## Runtime Binding

The module entrypoint is module.go and returns the existing Arkham engine implementation as a contracts.Engine.

## Dependency Direction

- Allowed: arkhamhorror -> serverengine/common/*
- Allowed: arkhamhorror -> protocol, monitoringdata
- Prohibited: serverengine/common/* -> arkhamhorror

## Migration Notes

The current binding is intentionally compatibility-first. Existing gameplay behavior is preserved while internals are incrementally moved into this package tree.
