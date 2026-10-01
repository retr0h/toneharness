# Marketplace

Rigs people can use and send. A rig names real gear in signal order, carries a
source for every claim, and compiles to a preset for whichever Helix you own, so
it travels in a way a `.hlx` does not.

Two tiers, and the difference is what somebody had to prove.

|                                  | what is in it         | the bar                                                  | ships in the binary |
| -------------------------------- | --------------------- | -------------------------------------------------------- | ------------------- |
| [core/](core/artists/)           | 30 files, 15 subjects | every claim has a source somebody opened                 | yes                 |
| [community/](community/artists/) | submissions           | it parses, it builds, and it says how well sourced it is | no                  |

## Using them

Core needs nothing beyond the checkout: it is compiled into the binary, so
`rigs list` and `presets make --id` find it.

Community is a directory, so point at it:

```bash
mise exec -- go run main.go rigs list --dir marketplace/community
mise exec -- go run main.go presets make --id <slug> \
  --rigs marketplace/community --out ~/out.hlx
```

Or ask your agent, which is the shorter version of the same thing:

> - _"What rigs are there for bass?"_
> - _"Build the community rig for Justin Chancellor and put it on my pedal."_

## What is in a pair

Each subject is two files. `<slug>.yaml` is the RigSpec, the gear.
`<slug>.tone.yaml` is the ToneSpec beside it, what somebody wanted of that gear.
One describes an answer and the other describes the question, which is why they
are not one file.

The instrument is a field on the rig rather than a directory, so a bass rig and
a guitar rig sit side by side and `rigs list` has an instrument column. Do not
add `bass/` and `guitar/` levels: the loader reads `artists/` and nothing else.

[write-a-spec](../.claude/skills/write-a-spec/SKILL.md) owns every field on both
and says which document a fact belongs in. The contracts are
[rigspec.openapi.yaml](../pkg/sdk/rig/data/rigspec.openapi.yaml) and
[tonespec.openapi.yaml](../pkg/sdk/tone/data/tonespec.openapi.yaml), and a
document that violates either is refused at load rather than half-read.

## Submitting one

Scaffold rather than copy a neighbour, because the scaffold knows which fields
exist:

```bash
mise exec -- go run main.go rigs new --help
```

Then build it, because a rig that does not resolve is not a rig:

```bash
mise exec -- go run main.go presets make --id <slug> \
  --rigs marketplace/community --out /tmp/check.hlx
```

Read what it printed. Every block it added that you did not name, every word
that moved nothing, every substitution it made for gear the device has no model
of. Those are decisions taken on the player's behalf and this is the moment to
disagree with them.

[build-a-rig's shipping.md](../.claude/skills/build-a-rig/references/shipping.md)
is the whole of it, and
[CONTRIBUTING's Sourcing a rig](../CONTRIBUTING.md#sourcing-a-rig) is the bar.

**A rig of `kind: llm` assertions is worse than no rig**, because it looks like
knowledge. If the research did not reach a source, say which piece of gear it
could not reach rather than filling the field in. `confidence` and `caveat`
exist for exactly the honest middle: a cited amplifier, a guessed cabinet, and a
sentence saying which is which.

## Getting into core

Citations. That is the only difference, and it is not a formality: core ships in
the binary, so a wrong claim there reaches everybody who installs this and looks
like something somebody checked.

A community rig whose gear gets sourced moves across, and `rigs show <slug>`
prints the caveats so a reviewer can see what each claim rests on without
opening the YAML.

## The copy under pkg/sdk

`core/` is the source of truth and `pkg/sdk/shipped/artists/` is a generated
copy of it, because `go:embed` cannot name a path above its own package
directory and the binary has to carry these. Edit the rig here, run
`mise exec -- just generate`, and commit both. A test fails when the two
disagree, in either direction.
