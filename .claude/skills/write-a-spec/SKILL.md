---
name: write-a-spec
description: Write or correct the two documents this project's formats are made of, a ToneSpec and a RigSpec, by hand. Covers which of the three layers a fact belongs to, naming gear the way a musician does, signal order, the musical settings vocabulary and what decides a value, the words an ask may carry, a song written as sections, what an expression pedal moves, recording what somebody thought of the result, and what belongs to a Setup instead. Use when asked to write, read, correct or explain a ToneSpec, a RigSpec, a Plan or a Setup, when a field is refused, or when deciding which document a fact goes in.
compatibility: Requires a toneharness checkout with mise available. Every command runs through `mise exec -- go run main.go`, never a bare `toneharness`.
license: MIT
metadata:
  author: retr0h
  source: https://github.com/retr0h/toneharness
---

# Write a spec

## 1. The contract is the grammar, not a page about it

Two hand-authored contracts, each embedded in the package that reads it:

- `pkg/sdk/tone/data/tonespec.openapi.yaml` is what somebody may ask for
- `pkg/sdk/rig/data/rigspec.openapi.yaml` is the gear that answers it

**Read them.** Their `description:` fields are the grammar, they list every
field and its allowed values, and the tool refuses a document that breaks them.
Nothing renders them into prose, because there is nothing to add: the Go types,
the validator and every client are generated from the same file, so a second
copy of the rules would only drift.

Never work from a list of fields, kinds or words written into this skill. A list
here is right the day it is written and wrong after the next change, with
nothing marking the moment. Ask the contract.

Check a document before believing it:

```bash
mise exec -- go run main.go rigs show --id mike-dirnt --json
mise exec -- go run main.go tone build --ask request.yaml --json
```

## 2. Three layers, and only two are typed

|              | holds                                        | written by                |
| ------------ | -------------------------------------------- | ------------------------- |
| **ToneSpec** | what somebody means, and why it is believed  | a person                  |
| **RigSpec**  | gear in signal order, named as a person does | a person, or the tool     |
| **Plan**     | that rig realised on one device              | the compiler, never a person |

A Plan has no contract, and that is the rule behind the split: **becoming a file
is not what earns a contract. Being typed by somebody is.**

## Two surfaces, one SDK

Every command here has a tool beside it over MCP, named after the command:
`device select` is `device_select`, `corpus music genres` is
`corpus_music_genres`. Use whichever the session offers. **The tools are the
same operations, not a reimplementation**: both surfaces call `pkg/sdk` and
answer with the same types, and a test walks the command tree against the
registered tools both ways, so neither can quietly gain a capability the other
lacks.

`.mcp.json` starts the server with `go run`, so it compiles the working tree
every launch and cannot serve a stale binary.

Three commands have no tool, and the reason is in that test's exempt list:
`measure blocks`, `measure controls` and `measure names` are sweeps. One reading
is about eight seconds, so a twelve-control amplifier is most of an hour, and a
tool that blocks that long is not one anybody can use. Run those from a terminal
where the progress shows and Ctrl-C reaches the session holding the pedal.

## 3. Route

| The question                                                     | Read                                                      |
| ---------------------------------------------------------------- | --------------------------------------------------------- |
| which document a fact belongs in, and how a pair is stored        | [references/pair.md](references/pair.md)                  |
| naming gear, signal order, gear the device does not model         | [references/gear.md](references/gear.md)                   |
| what a control is set to, and what decides it                     | [references/settings.md](references/settings.md)           |
| the words an ask carries, and what each one does                  | [references/terms.md](references/terms.md)                 |
| who the rig is for, when it applied, what it was played on        | [references/subject.md](references/subject.md)             |
| a song as sections, and what moves while you play                 | [references/song.md](references/song.md)                   |
| what a Plan holds that no rig does, and why                       | [references/plan.md](references/plan.md)                   |
| recording what a person thought of the result                     | [references/corrections.md](references/corrections.md)     |
| impulse responses, bought models, what one person owns            | [references/setup.md](references/setup.md)                 |
| all of it, in order                                               | All nine, in that order                                   |

## 4. Every field earns its place

A field is worth writing when it makes a claim checkable later. A field nothing
reads is worse than an absent one, because it looks like a feature: `drive: 0.47`
sat in a shipped rig doing nothing until something finally read it, and a
`requires` list sat in the contract for months. A test now walks both contracts
and fails on any field no code names.

So an absent field already says nobody established it. It needs no paragraph
explaining the absence, and what was searched for and not found belongs in a
task rather than in the document.
