# Writing the rig down, and shipping it

The research is the hard half and it is not the whole job. A rig nobody committed
helps the person in front of you once; a rig in the marketplace is there for
everybody.

Run this yourself. The person asked for a Justin Chancellor rig, not to be told
which command scaffolds one.

## Which tier it goes in

[marketplace/](../../../../marketplace/README.md) has two, and the difference is
what the research reached.

`marketplace/community/artists/` unless every claim has a source somebody opened.
That is the normal answer for a rig written today.

`marketplace/core/artists/` when it does, because core ships in the binary and a
wrong claim there reaches everybody who installs this. A rig that starts in
community and gets sourced later moves across.

**Never write a rig into `pkg/sdk/shipped/artists/`.** It is a generated copy of
`marketplace/core/`, so a rig put there is deleted by the next `just generate` and
the work is gone with no error. After changing anything in `core/`, run
`mise exec -- just generate` and commit both, which is what packs it into the
binary.

## Scaffold rather than write from scratch

```bash
mise exec -- go run main.go rigs new --help
mise exec -- go run main.go rigs new --dir marketplace/community \
  --id justin-chancellor --amp "Mesa Boogie Dual Rectifier" \
  --band Tool --genre prog-metal
```

`--dir` is the tier, and it is the flag to get right: without it the scaffold
lands wherever a rig of yours would go rather than in the marketplace. It writes
one document, the ask and the rig that answers it, with the fields and the
comments already there, and appends `artists/` itself. Writing one by hand means
rediscovering which fields exist, and the contract is the thing that knows.

The two halves are separate on purpose: the rig is the gear, the ask is what
somebody wanted of it. [write-a-spec](../../write-a-spec/SKILL.md) owns every
field and which half a fact belongs in. Read it rather than guessing from a
neighbouring file.

## Before you commit it

Build it. A rig that does not resolve is not a rig yet:

```bash
mise exec -- go run main.go presets make --id <slug> \
  --rigs marketplace/community --out /tmp/check.hlx
```

Read what it printed. Every block it added that the rig did not name, every word
that moved nothing, every substitution it made. Those are decisions taken on the
player's behalf and this is the moment to disagree with them.

`rigs records` says whether the records behind the rig are from the era it
claims. A rig for *Lateralus* measured against records from two albums later
describes a different sound, which [era.md](../../measure-music/references/era.md)
owns.

## When it cannot be established

Stop and say so, which [vague.md](vague.md) states plainly: do not ship a
half-known rig as though it were known. Specifically, do not commit one.

A rig of `kind: llm` assertions is worse than no rig, because it looks like
knowledge and the confidence field is the only thing saying otherwise. What to
hand back instead is what was searched, what was found, and which piece of gear
the research could not reach. That is a useful answer. A plausible rig is not.

The honest middle is a rig with a real citation for the amplifier and nothing for
the cabinet, committed with `confidence: low` and a `caveat` saying which half is
guessed. Say that out loud rather than letting the fields imply it.

## The pull request

```bash
mise exec -- just ready
```

Then open it. The description says who the rig is for, which album or era, what
the sources are, and which claims are weaker than the others. A reviewer should
not have to open the YAML to learn how well sourced it is.

[CONTRIBUTING's Sourcing a rig](../../../../CONTRIBUTING.md#sourcing-a-rig) is the
bar, and it says what a pull request has to have finished before it is opened.
Read it before starting rather than after.
