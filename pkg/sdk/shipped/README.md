# shipped

A generated copy of the marketplace's core tier, so the binary carries it.

**Do not edit a rig here.**
[marketplace/core/](../../../marketplace/core/artists/) is the source. Edit it
there, run `mise exec -- just generate`, and commit both.
`TestTheEmbeddedRigsAreTheMarketplacesCore` fails when the two disagree, in
either direction, so a rig edited here and not there goes back on the next run.

## Why there are two copies

`go:embed` cannot name a path above its own package directory. The rigs people
read, send and submit belong at the top of the repository where somebody
browsing it will find them, and the embed has to sit beside the package that
serves them, so one of the two has to be a copy.

The copy is committed rather than built on demand, which is the arrangement the
catalog and the corpus statistics already have: a checkout builds without
running a generator first, and a test rather than a convention keeps the
artifact honest.

## What is in it

Curated knowledge about how a sound is built: which gear a player or style uses,
and what somebody wanted of it. Each subject is one ToneSpec, `rig:` for the
gear and `ask:` for what was wanted of it. It is the same format a preset reads
back as and the same one that compiles to a device.

This is the only data in the repository that is ours. The device catalog and the
gear map come from Line 6's files, so we cannot ship those.

[write-a-spec](../../../.claude/skills/write-a-spec/SKILL.md) says how to write
one. [../tone/data/tonespec.openapi.yaml](../tone/data/tonespec.openapi.yaml) is
the contract, and `pkg/sdk/tone` refuses to load a document that violates it.
