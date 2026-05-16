# Architecture Decision Records (ADRs)

This directory contains Architecture Decision Records documenting significant design decisions made in the BostonFear project.

## What is an ADR?

An Architecture Decision Record (ADR) captures an important architectural decision along with its context and consequences. ADRs help:
- Communicate design rationale to new contributors
- Prevent revisiting settled debates
- Document trade-offs and alternatives considered
- Preserve institutional knowledge

## ADR Index

| ADR | Title | Status | Date | Topic |
|-----|-------|--------|------|-------|
| [001](001-ebitengine-client.md) | Go/Ebitengine Client Over JavaScript/Canvas | Accepted | 2026-05-16 | Client technology selection |
| [002](002-interface-based-networking.md) | Interface-Based Networking with net.Conn | Accepted | 2026-05-16 | Networking abstraction |
| [003](003-modular-game-architecture.md) | Modular Game-Family Architecture | Accepted | 2026-05-16 | Game engine modularity |

## ADR Template

When creating a new ADR, use this structure:

```markdown
# ADR NNN: [Title]

## Status
[Proposed | Accepted | Deprecated | Superseded by ADR-XXX]

## Date
YYYY-MM-DD

## Context
What is the issue we're facing? What constraints exist?

## Decision
What is the change we're proposing/making?

## Rationale
Why did we choose this option? What are the benefits?

## Alternatives Considered
What other options did we evaluate? Why were they rejected?

## Consequences
What becomes easier or harder as a result of this decision?

## Validation
How do we know this decision is working?

## Related Decisions
Links to other ADRs that are related or depend on this one.

## References
External links, documentation, or resources.
```

## Numbering

ADRs are numbered sequentially starting from 001. Use three-digit zero-padded numbers (001, 002, ..., 010, 011, ...).

## Status Values

- **Proposed**: Decision is under discussion
- **Accepted**: Decision is approved and implemented
- **Deprecated**: Decision is no longer relevant
- **Superseded by ADR-XXX**: Decision replaced by a newer ADR

## When to Write an ADR

Write an ADR when:
- Choosing between multiple valid approaches
- Making a decision that affects multiple modules
- Adopting a non-obvious pattern or convention
- Rejecting a commonly requested feature
- Changing a previously established pattern

Don't write an ADR for:
- Routine bug fixes
- Trivial code organization
- Implementation details within a single function

## Contributing

See [CONTRIBUTING.md](../../CONTRIBUTING.md) for guidelines on proposing new ADRs.
