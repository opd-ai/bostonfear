# Nightglass Core Set Content Specification (Default Demo Scenario)

## 1. Executive Summary
- This specification defines an original, copyright-safe base set called **Nightglass Core Set** for BostonFear demo usage.
- The canonical default demo scenario is `scn.nightglass.harbor-signal` (Harbor Signal).
- Content is structured for runtime loading via Go using versioned YAML/JSON-friendly documents and ID-based references.
- All required component classes are covered: investigators, items, abilities, threats, encounters, locations, conditions, and resources.
- Validation requirements are explicit for required fields, bounds, enum membership, adjacency integrity, and cross-file references.
- Default scenario resolution and fallback behavior are defined for missing/invalid configuration values.
- Dependency mapping distinguishes required and optional assets for deterministic startup validation.
- Copyright guardrails and originality checks are included as release-gating criteria.

## 2. Assumptions and Non-goals

### Assumptions
- Runtime already supports scenario loading at startup from external content files.
- Engine action loop supports two actions per player turn and global escalation hooks.
- Loader can parse YAML or JSON and validate referential integrity before serving sessions.
- Clients can render labels and flavor text from content IDs.

### Non-goals
- No Go code changes in this specification deliverable.
- No use of any Fantasy Flight names, text patterns, or proprietary symbols.
- No expansion scenarios or campaign chains beyond one complete demo scenario.
- No final balancing pass for competitive tuning beyond demo-ready knobs.

## 3. Content Architecture

### Directory and File Layout Proposal

```text
serverengine/arkhamhorror/content/nightglass/
  manifest.yaml
  enums.yaml
  base-set/
    investigators.yaml
    items.yaml
    abilities.yaml
    threats.yaml
    encounters.yaml
    locations.yaml
    conditions.yaml
    resources.yaml
  scenarios/
    index.yaml
    nightglass-harbor-signal.yaml
  text/
    en-US.yaml
```

### Schema Field Definitions Per Component Type

#### Common Envelope (all files)
- `schemaVersion`: string, semver, required
- `contentPackId`: string, required
- `language`: string, required (BCP-47)
- `generatedAt`: string timestamp, required
- `records`: array, required

#### Common Record Fields (all records)
- `id`: string, required, globally unique, format `type.namespace.slug`
- `name`: string, required
- `summary`: string, required
- `enabled`: bool, required
- `version`: int, required, `>= 1`
- `tags`: string array, optional

#### Investigators
- Required: `archetype`, `startingLocationId`, `stats`, `startingResources`, `passiveAbilityId`
- Optional: `activeAbilityIds`, `personalConditionDeckIds`
- Stats required keys: `vitalityMax`, `composureMax`, `wit`, `force`, `finesse`

#### Items
- Required: `slot`, `cost`, `modifiers`, `actionGrantIds`
- Optional: `charges`, `keywords`

#### Abilities
- Required: `type`, `trigger`, `cost`, `effect`
- Optional: `cooldownTurns`, `oncePerRound`

#### Threats
- Required: `class`, `spawnRules`, `stats`, `behaviorScriptId`, `onDefeatEffects`, `reward`

#### Encounters
- Required: `deck`, `locationTagFilter`, `test`, `outcomes`, `weight`
- Outcomes required keys: `pass`, `fail`, `criticalPass`, `criticalFail`

#### Locations
- Required: `districtType`, `adjacency`, `traits`, `encounterDeckRefs`, `threatCapacity`, `coordinates`

#### Conditions
- Required: `polarity`, `stackable`, `maxStacks`, `duration`, `expireTrigger`, `effects`

#### Resources
- Required: `scope`, `min`, `max`, `defaultValue`, `uiPriority`, `visibility`
- Optional: `lossConditionFlag`, `winConditionFlag`

### Validation Rules

#### Required Fields
- All required fields must be present and non-empty.
- Unknown enum values are validation errors.

#### Bounds and Value Rules
- Resource values: `min <= defaultValue <= max`.
- `res.nightfall` hard bounds: `0..12`.
- Investigator maxima must not exceed resource maxima.
- Difficulty scalar bounds:
  - `escalationRate`: `1..3`
  - `threatHealthScalar`: `0.5..2.0`
  - `victoryWardTarget`: `1..12`

#### Referential Integrity
- Every referenced ID must resolve to an enabled record.
- Scenario required dependency IDs must all resolve.
- Encounter decks and spawn tables may only reference existing threat/encounter IDs.

#### Topology Integrity
- Location adjacency must be symmetric unless explicitly flagged one-way.
- Scenario starting locations must exist and be connected to the graph.

#### Startup Integrity
- Exactly one default scenario ID must resolve after fallback sequence.
- If no enabled scenario remains, startup must fail with explicit validation error.

### Content Dependency Map
- `serverengine/arkhamhorror/content/nightglass/manifest.yaml`:
  - Declares pack-level required files and optional localization files.
  - Declares pack-level `defaultScenarioId`.
- `serverengine/arkhamhorror/content/nightglass/scenarios/index.yaml`:
  - Declares enabled scenario records and scenario-level default pointer.
- `serverengine/arkhamhorror/content/nightglass/scenarios/nightglass-harbor-signal.yaml`:
  - Requires resources, locations, threats, and key encounter IDs.
  - References action IDs and objective IDs used by runtime flow.
- `serverengine/arkhamhorror/content/nightglass/base-set/investigators.yaml`:
  - Depends on `locations`, `abilities`, and optional `conditions`.
- `serverengine/arkhamhorror/content/nightglass/base-set/items.yaml`:
  - Depends on `resources` for costs and runtime action IDs for grants.
- `serverengine/arkhamhorror/content/nightglass/base-set/abilities.yaml`:
  - Depends on `resources`, `conditions`, and action identifiers.
- `serverengine/arkhamhorror/content/nightglass/base-set/threats.yaml`:
  - Depends on behavior script IDs and optional condition/resource effects.
- `serverengine/arkhamhorror/content/nightglass/base-set/encounters.yaml`:
  - Depends on `threats`, `conditions`, and `resources`.
- `serverengine/arkhamhorror/content/nightglass/base-set/locations.yaml`:
  - Depends on valid location self-references for adjacency and encounter deck IDs.
- `serverengine/arkhamhorror/content/nightglass/base-set/conditions.yaml`:
  - Depends on valid effect payload schema and referenced resource/stat/action IDs.
- `serverengine/arkhamhorror/content/nightglass/text/en-US.yaml`:
  - Optional; depends on stable UI key IDs only.

## 4. Base Set Inventory Table

| Component Type | Target Count | Design Intent | Required Fields | Example IDs |
|---|---:|---|---|---|
| Investigators | 6 | Team role diversity and replayability | archetype, stats, startingLocationId, passiveAbilityId | `inv.mara-quill`, `inv.ion-voss` |
| Items/Equipment | 12 | Build expression and resource sinks | slot, cost, modifiers/actionGrantIds | `item.tide-lantern`, `item.breakwater-key` |
| Abilities | 12 | Distinct role actions and reactions | type, trigger, cost, effect | `abl.chalk-aegis`, `abl.fade-step` |
| Threats | 10 | Escalation and tactical pressure | class, spawnRules, stats, behaviorScriptId | `thr.gutter-maw`, `thr.rift-sleet` |
| Encounters/Events | 12 | State variance and narrative beats | deck, test, outcomes, weight | `enc.omen-low-tide`, `enc.crisis-anchor-slip` |
| Locations | 8 | Movement puzzle and district identity | adjacency, districtType, encounterDeckRefs | `loc.mirror-square`, `loc.north-breakwater` |
| Conditions | 12 | Ongoing effects and consequences | polarity, duration, effects | `cond.shaken`, `cond.lucid-focus` |
| Resources/Tokens | 7 | Core economy and global pressure | scope, min/max, defaultValue | `res.vitality`, `res.nightfall` |

## 5. Demo Scenario Spec (Default)

### Scenario ID and Default Flag/Config Location
- Canonical scenario ID: `scn.nightglass.harbor-signal`
- Content metadata default declaration:
  - `serverengine/arkhamhorror/content/nightglass/manifest.yaml` -> `defaultScenarioId`
  - `serverengine/arkhamhorror/content/nightglass/scenarios/index.yaml` -> `defaultScenarioId`
  - `serverengine/arkhamhorror/content/nightglass/scenarios/nightglass-harbor-signal.yaml` -> record-level `default: true`
- Runtime config declaration:
  - `config.toml` -> `[scenario] default_id = "scn.nightglass.harbor-signal"`

### Narrative Premise
A deepwater beacon sends impossible tones that drag living shadows toward Brindlehaven. Investigators must stabilize ward anchors and sever the source signal before city-wide collapse.

### Setup
- Players: 1-6
- Map nodes: 8 fixed locations in one connected graph
- Starting resources per investigator: vitality 8, composure 8, leads 1, focus 1, salvage 0
- Starting global resources: nightfall 1, ward-charge 0
- Initial threat spawns:
  - Always: 1 threat at Blackwater Docks
  - Additional for 4+ players: 1 threat at Old Reservoir

### Turn and Phase Flow
1. Coordination phase: reveal one Omen encounter; apply start-of-round effects.
2. Investigator phase: each investigator performs exactly two actions.
3. Escalation phase: increase nightfall, spawn threats by tier, resolve one Crisis encounter.

### Escalation Mechanics
- Fracture result on any test increases `res.nightfall` by 1.
- End-of-round escalation increases `res.nightfall` by difficulty rate.
- Threat spawns scale by nightfall tier (`0-4`, `5-8`, `9-12`).
- Surge rule: if 3+ fractures occur in one round, spawn one extra hazard threat.

### Victory and Defeat Conditions
- Victory requires both:
  - Reach ward-charge target (difficulty dependent).
  - Complete Signal Sever at North Breakwater with team leads threshold by player count.
- Defeat occurs when any:
  - `res.nightfall >= 12`
  - all investigators incapacitated
  - resonance breach counter reaches 3

### Difficulty Knobs
- `easy`: escalation 1, survey threshold 1, seal lead cost 1, threat health scalar 0.8, ward target 4
- `standard`: escalation 2, survey threshold 2, seal lead cost 2, threat health scalar 1.0, ward target 5
- `hard`: escalation 3, survey threshold 3, seal lead cost 2 plus higher composure tax, threat health scalar 1.25, ward target 6

### Fallback Behavior
1. Use `[scenario].default_id` if valid and enabled.
2. Else use `serverengine/arkhamhorror/content/nightglass/scenarios/index.yaml` defaultScenarioId.
3. Else use first enabled scenario sorted by ID and log warning.
4. If none available, abort startup with content validation error.

## 6. Copyright Safety Section

### Prohibited Overlap Categories
- Any reused proprietary names, places, factions, entities, card labels, or iconic terms from Fantasy Flight products.
- Any copied rules text, flavor text cadence, or distinctive templating patterns.
- Any visual icon naming tied to known product identity systems.

### Originality Checks
- Lexicon blocklist scan against prohibited names/terms.
- Phrase similarity review on all long-form text snippets.
- Independent two-reviewer naming check before release.
- Final legal/editorial signoff checkpoint in release process.

### Red-Flag Examples to Avoid
- Direct usage of trademarked proper nouns.
- Near-copy wording for action names or encounter text structures.
- One-to-one renamed clones of known location maps or campaign beats.

Checklist artifact: `docs/content/COPYRIGHT_ORIGINALITY_CHECKLIST.md`.

## 7. Delivery Plan

### Phased Implementation Order
1. Schema and enum freeze.
2. Resource and location graph authoring.
3. Investigator/item/ability authoring.
4. Threat/encounter/condition authoring.
5. Scenario + default-index wiring.
6. Loader validation and startup fallback tests.
7. Balance pass and multiplayer smoke test.

### Acceptance Checks Before Runtime Integration
- All component classes exist with required schema fields.
- No unresolved ID references across content files.
- Scenario is playable from setup to both win and lose outcomes.
- Default scenario resolution works for valid, missing, and invalid configured IDs.
- Copyright checklist completed with zero blocked overlaps.
- Package is handoff-ready for Go runtime loading without further design decisions.
