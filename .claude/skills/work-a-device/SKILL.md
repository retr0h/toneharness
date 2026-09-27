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

## 3. Never write a slot nobody offered

Banks 01 to 10 hold the owner's own work and are not to be touched. Scratch
writes go to slot 40 and above. A device has no undo: whatever a slot held is
read and kept before a write replaces it, and the answer says where that backup
went.

Then say which claim you have, because three get confused:

1. the tool wrote the slot and read the same bytes back
2. HX Edit or the pedal's own display shows the blocks
3. signal went through it and it sounded right

Reading a slot back does not establish the second. A preset can be stored
perfectly, read back byte for byte, and rendered by the pedal as an empty chain.
See [references/formats.md](references/formats.md). Report what you ran and what
it answered, including every note it returned, and if you did not run it, say you
did not run it.
