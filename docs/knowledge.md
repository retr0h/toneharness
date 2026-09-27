# What is built, and what is not

Orientation for somebody arriving at this repository. How to *do* any of it is
in [the skills](../.claude/skills/), which are the authority; this page is the
state of the work and nothing else.

## Constructing is not copying

The corpus holds a preset called "Basket Case". Copying it would inherit one
person's opinion, including their mistakes, and would answer nothing for a
player nobody has made a preset for.

The goal is a system that knows *how a chain is built*. That decomposes into six
problems with six different sources, and conflating them is why generated tones
come out generic.

| Problem                   | Source                                       | State                                       |
| ------------------------- | -------------------------------------------- | ------------------------------------------- |
| Who plays what            | `pkg/sdk/shipped/`, a hand-written pair each | thin, grows by correction                   |
| Gear to model ID          | `resources/schemas/gear-map.json`            | 547 models                                  |
| What order blocks go in   | statistics over `resources/schemas/corpus/`  | added blocks placed; a rig's own order kept |
| Which way a knob moves    | swept on the device, in `resources/sweeps/`  | eleven blocks measured and shipped          |
| What values to set        | catalog defaults, corpus medians, intent     | six axes of ten                             |
| What a genre sounds like  | displacement over `resources/music/`         | 3 tagged, 2 earning a word                  |
| What a player sounds like | measured over `resources/music/bass/`        | 15 players, 49 records, 4 earning a word    |

**Keep this table honest.** A pull request that finishes something marked not
built or partly built updates the line in the same pull request and says so in
its description. This table fell three features behind when nobody did, and it
is how the next session learns what exists.

## Three layers

|              | holds                                        | written by                   |
| ------------ | -------------------------------------------- | ---------------------------- |
| **ToneSpec** | what somebody means, and why it is believed  | a person                     |
| **RigSpec**  | gear in signal order, named as a person does | a person, or `tone build`    |
| **Plan**     | that rig realised on one device              | the compiler, never a person |

All three exist. A rig is portable because there is no longer anywhere in it to
put a Helix answer: the resolved model per block, the position, the snapshots
and footswitches, and the device state a lifted preset arrived with all live on
the Plan.

Only the first two have contracts, and the rule is worth stating: **becoming a
file is not what earns a contract. Being typed by somebody is.**

[A rig is a plan, for one device](superpowers/specs/2026-09-19-a-rig-is-a-plan-for-one-device-design.md)
is the record for that split, and
[ToneSpec is the ask](superpowers/specs/2026-09-19-tonespec-is-the-ask-design.md)
says how the first two divide, both superseding
[the RigSpec design record](superpowers/specs/2026-09-06-rigspec-as-the-one-model-design.md).

## The pipeline

```text
request      "a Mike Dirnt sound"
   │
   ▼
the ask      pkg/sdk/shipped/artists/mike-dirnt.tone.yaml     who it is for
   │         words: scooped, clean — or genre: grunge
   ▼
the rig      pkg/sdk/shipped/artists/mike-dirnt.yaml          who plays what
   │         amp: Ampeg SVT
   ▼
gear map     resources/schemas/gear-map.json                  gear to model
   │         HD2_AmpSVBeastNrm
   ▼
catalog      pkg/sdk/catalog/data/hx-stomp.json.gz            what the device accepts
   │         Drive 0.0–1.0, default 0.53, DSP 26.67
   ▼
grammar      pkg/sdk/corpus/data/hx-stomp.stats.json.gz       what a chain almost always holds
   │
   ▼
genres       pkg/sdk/audio/data/genres.json                   what a genre is displaced on
   │         grunge: scooped, clean
   ▼
values       corpus medians + the ask's words + the genre's   what to set
   │
   ▼
RigSpec      validated against the catalog
   │
   ▼
.hlx         written, and put on a device over USB
   │
   ▼
a person     listens, and corrects the pair                   nothing above can hear
```

## Nothing here can hear

No part of this system can judge whether a preset sounds right, and no quantity
of corpus data changes that. **The evaluator is a person**, and the architecture
assumes it in three places.

**The correction loop must be cheap.** Generate, push to the device, listen, fix
one line, never hear that mistake again. That is why the device library matters
more than more corpus: it removes a manual HX Edit import from every iteration.

**Decisions must be inspectable.** A generated rig records why each block was
chosen and how confident that choice was, so a wrong amplifier is visible before
anybody plugs in rather than after.

**Corrections must be permanent.** A fix belongs in the files, where it outranks
generated knowledge for good.

## A word is earned against a population, so it is not permanent

The corpus went from nine players to fifteen and six words stopped being earned.
`scooped` and `clean` came off Mike Dirnt's ask, `scooped` off Pino Palladino's,
and three more that survived on a citation had their measurement corrected to
say the records no longer support them. Pino's `scooped` margin had been the
narrowest in the corpus at 0.1%, with a note saying one more player would take
it. Six arrived and it did.

That is the population doing its job rather than a wobble in it. **A word whose
only evidence was a comparison against nine people is a word about those nine**,
so the rule is that such a word goes when the comparison stops supporting it,
while a word with a citation behind it stays and the measurement beside it is
corrected to say what it now says.

## What is still missing

The step from a word to a value is half built: a term says which way to move a
control, and where it carries the figures that earned it the distance follows
the gap. A word nobody measured still moves a fixed step.

Nothing above the person listens, so whether half a step of drive is the right
amount of drive is a question no measurement here answers. The method for
closing that is designed and not built:
[solving for knob positions](superpowers/specs/2026-09-27-solving-for-knob-positions-design.md).
