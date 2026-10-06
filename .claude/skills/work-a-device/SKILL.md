---
name: work-a-device
description: Read, write and rearrange what a Line 6 Helix holds, over USB or in an HX Edit backup. Covers listing slots, reading a preset back as a rig, auditioning a chain without writing flash, placing one into a slot, copying, swapping and moving presets, exporting and restoring, and the session rules that keep a pedal on the bus. Use when asked what is on the pedal, to get a preset onto it, to move or swap slots, to back one up or put it back, to work from a .hlb with nothing attached, or when a device stops answering.
compatibility: Requires a toneharness checkout with mise available. Every command runs through `mise exec -- go run main.go`, never a bare `toneharness`. Device access is macOS only; every `--file` path works everywhere.
license: MIT
metadata:
  author: retr0h
  source: https://github.com/retr0h/toneharness
---

# Work a device

Reading, writing and rearranging what a Helix holds. Building a preset from a
request is `build-a-rig`; pushing signal through the pedal and measuring what
comes back is `measure-a-device`.

## 1. Establish what is attached, and what it holds

Every run, before naming a slot. **Quit HX Edit first.** It claims the editor
interface exclusively and nothing here can reach the device while it runs.

```bash
mise exec -- go run main.go --help                    # which groups exist
mise exec -- go run main.go device --help             # what that group takes
mise exec -- go run main.go device hardware --json    # what is on the bus
mise exec -- go run main.go slots list --json         # what it holds
```

`--help` on a checkout compiles the checkout, so it is the source's own answer
rather than a description of it, and it shows the flags a parent registers that
reading one file would miss. Read `cmd/` only to change a command, never to find
out what one does.

Never work from a list written into this skill. Slot contents, setlist counts,
model numbers and which pedals are on the bus are things to ask, and a list here
is right the day it is written and wrong after the next firmware, with nothing
marking the moment. `--json` works on every command; use it, because the painted
tables leave things out on purpose.

Two groups, and the seam between them is not device-against-file. `slots` is the
positions a setlist holds and takes either backing: name `--file` and it edits
that backup, name none and it reaches the attached device. `device` is what only
ever means anything on live hardware.

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

## 2. Route

| The user asks                                                        | Read                                                       |
| -------------------------------------------------------------------- | ---------------------------------------------------------- |
| what is on the pedal, what a slot holds, whether a slot is empty      | [references/reading.md](references/reading.md)             |
| to get a preset onto the device, to try one, to put a backup back     | [references/writing.md](references/writing.md)             |
| to load, copy, swap or move a preset between slots                    | [references/rearranging.md](references/rearranging.md)     |
| why the front panel is dead, why the pedal stopped answering          | [references/session.md](references/session.md)             |
| what a `.hlx`, `.hls`, `.hlb` or `.bin` is, and what converting loses | [references/formats.md](references/formats.md)             |
| what a chain carries besides blocks: snapshots, routing, the grid     | [references/chain.md](references/chain.md)                 |
| which device this is for, and which catalog answers                   | [references/identity.md](references/identity.md)           |
| anything else                                                        | All seven, in that order                                   |

## The slot called SCRATCH

**A slot named `SCRATCH` is the working slot.** Anything may be written there,
played there and measured there, as many times as a run needs. It is how
somebody says which slot they are willing to lose.

**Every other slot holds somebody's work until they say otherwise.** A name is
not permission: a slot called "Mike Dirnt" is somebody's preset even when it is
exactly what you were about to build, and that is the reasoning this rule exists
to stop. `slots list` says what each one is called; if nothing is called
`SCRATCH`, ask for one rather than choosing.

`TONEHARNESS_SCRATCH_SLOT` names it for the device tests. Naming the preset in it
`SCRATCH` is what makes the same answer visible on the pedal, where a person
reading the screen can see that the slot they are on is the disposable one.

Rename it back to `SCRATCH` when a run is finished, whatever was being worked on.
A slot left called "SCRATCH dirnt" is one the next session has to think about.

## 3. Never write a slot nobody offered

Read `slots list` first and write only to something empty, or where the person
whose pedal it is has said to. Every slot holds somebody's work until shown
otherwise, and a high bank is the convention here only because low banks are
where most people keep what they play. `TONEHARNESS_SCRATCH_SLOT` is how somebody
names the slot they are willing to lose.

A device has no undo: whatever a slot held is read and kept before a write
replaces it, and the answer says where that backup went.

Then say which claim you have, because three get confused:

1. the tool wrote the slot and read the same bytes back
2. HX Edit or the pedal's own display shows the blocks
3. signal went through it and it sounded right

Reading a slot back does not establish the second, and no amount of reading ever
will: a preset can be stored perfectly, read back byte for byte, and rendered by
the pedal as an empty chain. Two faults have done that and both are fixed, but
the asymmetry that hid them is permanent. See
[references/formats.md](references/formats.md). Report what you ran and what
it answered, including every note it returned, and if you did not run it, say you
did not run it.

## Say which claim you have, and where it went

Four claims, and the gap between the second and the fourth is what this skill
exists to keep honest:

1. a file was written
2. the device accepted the write
3. the slot reads back byte for byte
4. the pedal is playing it

The third has passed while the fourth failed: for a fortnight every preset this
tool wrote read back identically and drew nothing on the screen. So a write is
not a load, and `device current` is what tells them apart.

**Name every path.** A slot write keeps whatever the slot held and prints where it
went; repeat that path in the answer, because a reply that drops it has turned a
recoverable change into a lost one. Say which slots were written, too — each is a
flash write, and the person deciding whether to do it again is holding the pedal.

[AGENTS.md's what to report](../../../AGENTS.md#what-to-report-every-time) is the
shape, and it is the same for every skill here.
