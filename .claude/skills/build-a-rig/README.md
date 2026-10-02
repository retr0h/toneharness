# build-a-rig

Answers "make my bass sound like Dookie" without guessing at the gear.

## Install

```
/plugin marketplace add retr0h/toneharness
/plugin install build-a-rig@toneharness
```

Or copy `.claude/skills/build-a-rig/` into any checkout.

## Usage

Say what you want to sound like. The skill does the research, writes the
request down, resolves it and tells you which claim it has.

| Ask                                                | You get                                                                                    |
| -------------------------------------------------- | ------------------------------------------------------------------------------------------ |
| _"Make my bass sound like Dookie"_                 | the gear Mike Dirnt actually used in 1994, cited, as a ToneSpec and the rig it resolves to |
| _"Like Dirnt but chunkier"_                        | the same, with the nudge carried as a word rather than as a number somebody invented       |
| _"What did Jaco play through?"_                    | the research and the sources, and nothing written until you want it                        |
| _"Punk, but on my Jazz bass"_                      | a genre and your own instrument, which are two different documents                          |
| _"Measure this recording and pick me an amp"_      | the nearest of the instrument's own amplifiers to what the file measures, with the figures |
| _"Put it on the pedal"_                            | a preset in a scratch slot, and what it would have replaced                                |
| _"The mids are honky"_                             | an edit to the rig, and what moved                                                         |

## How it works

Two documents a person writes, and one of them holds two halves.

A **ToneSpec**'s `ask:` is what you want: a player, a genre, some words, gear you
insist on. A **Setup** is what you own, which changes when you buy something
rather than every request. The ToneSpec's `rig:` is what those two resolve to:
exact models, in order, deterministic, which is the half worth sharing because
two people compiling one get the same preset.

Nothing here guesses. An amplifier chosen by measuring is chosen by pushing a
recording through every amplifier the instrument has and comparing the same nine
figures, and
every claim about real gear carries a source somebody opened. Where the tool
cannot answer, it says so in notes that travel with the answer rather than as
prose above it.

It follows the [Agent Skills] format: a slim `SKILL.md` that routes, with the
detail in reference files read only when the question calls for them.

[agent skills]: https://agentskills.io
