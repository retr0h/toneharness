# Marketplace

Rigs people can use and send. A rig names real gear in signal order, carries a
source for every claim, and compiles to a preset for whichever Helix you own, so
it travels in a way a `.hlx` does not.

Two tiers, and the difference is what somebody had to prove.

|                                  | what is in it         | the bar                                                  | ships in the binary |
| -------------------------------- | --------------------- | -------------------------------------------------------- | ------------------- |
| [core/](core/artists/)           | 30 files, 15 subjects | every claim has a source somebody opened                 | yes                 |
| [community/](community/artists/) | submissions           | it parses, it builds, and it says how well sourced it is | no                  |

[core/examples/](core/examples/README.md) sits beside the rigs rather than in
them: one of each document, commented, for somebody learning the formats. The
loader reads `artists/` and nothing else, so nothing there is listed as a rig or
packed into the binary.

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

## What is in a document

Each subject is one file, `<slug>.yaml`, holding both halves. `rig:` is the gear
and is required; `ask:` is what somebody wanted of it. One describes an answer
and the other the question, and they travel together because neither is much use
without the other.

They were two files until version 2 of the contract, `<slug>.rig.yaml` and
`<slug>.tone.yaml`. The pair was one-to-one in every case that existed and an
ask with no rig was already refused, so the second file was an annotation kept
in step by hand.

The identifier is `id:` inside the file and the filename stem is expected to
match it. Two files claiming the same identifier are reported rather than one of
them quietly winning.

The instrument is a field on the rig rather than a directory, so a bass rig and
a guitar rig sit side by side and `rigs list` has an instrument column. Do not
add `bass/` and `guitar/` levels: the loader reads `artists/` and nothing else.

[write-a-spec](../.claude/skills/write-a-spec/SKILL.md) owns every field and
says which half a fact belongs in. The contract is
[tonespec.openapi.yaml](../pkg/sdk/tone/data/tonespec.openapi.yaml), and a
document that violates it is refused at load rather than half-read.

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
