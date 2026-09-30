# measure-a-device

Answers "what does that knob actually do" with a number somebody else can take
again.

## Install

```
/plugin marketplace add retr0h/toneharness
/plugin install measure-a-device@toneharness
```

Or copy `.claude/skills/measure-a-device/` into any checkout.

## Usage

Attach the pedal, quit HX Edit, and ask. The skill establishes the signal path
before it believes a reading, and says which claim it has when it answers.

| Ask                                              | You get                                                                       |
| ------------------------------------------------ | ----------------------------------------------------------------------------- |
| _"What does this amp's Treble do?"_              | a curve, every position, and the chain it was taken in                        |
| _"Which parameter is index 3?"_                  | the catalog's order held to the hardware rather than trusted                   |
| _"Measure every block once"_                     | one reading per block against an empty-loop baseline, resumable               |
| _"That reading looks wrong"_                     | the noise floor, the silent positions, and which of five causes refused       |
| _"Put this in front of the pedal"_               | a live replace that writes no flash and leaves every slot as it was           |

## What it covers

A recording says what the answer should sound like and cannot say what any one
control did, because every control is already in it and none can be moved. So a
fixed reference signal goes in, one thing moves, and what comes out gets
measured into the same figures a record is described in.

The care is in the parts that produce plausible wrong numbers: silence reads as
a huge bright move, a slope measured in a full chain is not a property of the
control, a live edit leaves the control where it finished, and a parameter has
no name on the wire. Each of those was got wrong first, and each reference says
which reading it invalidated.

It follows the [Agent Skills] format: a slim `SKILL.md` that routes, with the
detail in reference files read only when the question calls for them.

[agent skills]: https://agentskills.io
