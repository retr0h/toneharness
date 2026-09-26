---
name: build-a-rig
description: Turn what somebody says they want to sound like into a Line 6 Helix preset, with evidence at every step. Covers researching a player's real gear and citing it, writing a ToneSpec, resolving it to a rig, measuring a recording to choose an amplifier, putting a preset on the pedal, tuning live and exporting what worked. Use when asked to make something sound like an artist, a band, a record or a genre, to build or correct a rig or preset, to find out what gear somebody used, to measure a recording, or to get a tone onto a Helix.
compatibility: Requires a tonestack checkout with mise available. Every command runs through `mise exec -- go run main.go`, never a bare `tonestack`.
license: MIT
metadata:
  author: retr0h
  source: https://github.com/retr0h/tonestack
---

# Build a rig

## 1. Establish what the device can actually do

Every run, before naming any gear. The catalog is the only answer to what
exists, and a model it does not carry does not exist:

```bash
mise exec -- go run main.go catalog list --category amp --json
mise exec -- go run main.go recipes list --json
```

Never work from a list written into this skill. The device carries 665 blocks
across four models and Line 6 rename things between releases, so a list here
is right the day it is written and wrong after the next one, with nothing
marking the moment. The same goes for the shipped rigs, the block categories
and the figures a reading is reported in: ask the tool.

`--json` works on every command. Use it. The painted tables are for somebody
reading a terminal and they leave things out on purpose.

## 2. Route

| The user asks                                                    | Read                                                        |
| ---------------------------------------------------------------- | ----------------------------------------------------------- |
| "make it sound like X", "build me a rig", what gear somebody used | [references/research.md](references/research.md)            |
| to write or change a request, a ToneSpec, a Setup                 | [references/asking.md](references/asking.md)                |
| to measure a recording, or to choose gear by measuring            | [references/measuring.md](references/measuring.md)          |
| to get it onto the pedal, or to read what the pedal holds         | [references/device.md](references/device.md)                |
| to change a rig after hearing it                                  | [references/correcting.md](references/correcting.md)        |
| anything touching claims, citations or evidence                   | [references/evidence.md](references/evidence.md)            |
| all of it, in order                                               | All six, in that order                                      |

## 3. Say which claim you have

Three different claims, and only the first is currently possible here:

1. the rig validates against the catalog
2. HX Edit imported the file
3. the hardware loaded it and it sounded right

Never report one as another, and never describe work as verified on evidence
you did not gather. If you did not run it, say you did not run it. This is the
failure this project cares about most, and machine-readable output makes it
easier to commit rather than harder.

Report in this order, because it is the order that avoids rework:

1. what the request was, in the user's words
2. what was established and from which source, with a link
3. what the tool answered, including every note it returned about what it
   could not honour
4. which of the three claims above you have
