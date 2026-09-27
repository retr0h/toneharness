# work-a-device

Answers "what is on my Helix, and get this onto it" without bricking the pedal.

## Install

```
/plugin marketplace add retr0h/toneharness
/plugin install work-a-device@toneharness
```

Or copy `.claude/skills/work-a-device/` into any checkout.

## Usage

Plug the Helix in, quit HX Edit, and say what you want done to it. Everything
also works from an HX Edit backup with nothing attached.

| Ask                                     | You get                                                                 |
| --------------------------------------- | ----------------------------------------------------------------------- |
| _"What's on the pedal?"_                | every named slot and what it actually holds, not just what it is called  |
| _"Put this on it"_                      | a preset in a scratch slot, and a backup of whatever it replaced         |
| _"Let me hear it first"_                | the chain in front of the device with no flash written at all            |
| _"Move 01A to 27B"_                     | a swap, which is how a move is done, in the order that cannot lose it    |
| _"Back it up before I break something"_ | a `.hlb` holding every setlist, and the restore that puts it back        |
| _"Read 31A back as a rig"_              | the gear, portable, and it compiles back into the preset it came from    |
| _"The screen has gone dead"_            | the reason, which is that something is still holding the editor session  |

## What it covers

Reading, writing and rearranging what a device holds. Building a preset from a
request is `build-a-rig`; pushing signal through the pedal and measuring what
comes back is `measure-a-device`.

The judgement it carries is the part that cost hardware: a session left open
kills the front panel, a select has to be waited for or the pedal wipes its edit
buffer, a named slot can still be empty, a burst of writes can take a setlist
past what a power cycle clears, and a preset can be stored perfectly, read back
byte for byte, and still render as nothing. Each of those looks like a bug in the
tool the first time, and none of them is.

Banks 01 to 10 are never written. Scratch goes to slot 40 and above.

It follows the [Agent Skills] format: a slim `SKILL.md` that routes, with the
detail in reference files read only when the question calls for them.

[agent skills]: https://agentskills.io
