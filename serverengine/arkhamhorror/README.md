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

## Content Expansions

### Base Content Pack: Nightglass Core
- **Content Pack ID**: `nightglass.core`
- **Location**: `serverengine/arkhamhorror/content/nightglass/`
- **Investigators**: 6 base investigators (Mara Quill, Ion Voss, Selene Raft, Bram Caul, Petra Kline, Orrin Hale)
- **Scenarios**: 4 base scenarios (Harbor Signal, Asylum Echoes, Market Ritual, Archive Corruption)
- **Locations**: 8 city locations across 4 neighborhoods
- **Status**: Embedded and auto-installed on server startup

### Expansion Pack: Dead of Night
- **Content Pack ID**: `deadofnight.expansion`
- **Location**: `serverengine/arkhamhorror/content/deadofnight/`
- **Dependencies**: Requires `nightglass.core`
- **Investigators**: 4 new investigators (Evelyn Cross, Marcus Graves, Vera Night, Silas Thorne)
- **Scenarios**: 2 new scenarios (Museum Awakening, Graveyard Rising)
- **Encounter Decks**: Museum, Graveyard, Nocturnal
- **New Items**: Silver Lantern, Burial Shroud, Obsidian Amulet, Grave Dirt, Bone Whistle, Museum Key
- **New Threats**: Museum Sentinel, Grave Revenant, Spectral Warden, Cursed Relic
- **Status**: Content structure defined; not yet embedded or auto-loaded

### Adding New Content Packs

To add a new expansion:
1. Create content directory: `serverengine/arkhamhorror/content/{pack_name}/`
2. Define manifest.yaml with pack ID, dependencies, and file list
3. Create base-set YAML files: investigators, items, abilities, threats, encounters, conditions
4. Create scenarios directory with index.yaml and scenario YAML files
5. Add tests in `content/{pack_name}_test.go` to verify structure
6. Update this README with expansion details
7. (Optional) Add embedding via `//go:embed` directive in `content/embedded.go`
