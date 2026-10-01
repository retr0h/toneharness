# What one person owns

A Setup says what is in somebody's room. It is the third document a person
writes, and it changes when they buy something rather than on every request.

```yaml
schema: Setup
owns:
  - { kind: ir, name: Ownhammer SVT 8x10, slot: 82 }
```

Folding this into the ask would mean restating their bass in every request, and
the twelfth one would contradict the first.

## What the pedal is plugged into

`plays_into`, optional, one of four: `pa`, `headphones`, `amp-front`,
`amp-return`.

```yaml
schema: Setup
plays_into: amp-return
```

It belongs here rather than in an ask for the same reason the bass does:
somebody who plays through a PA plays through a PA next week too.

The two amplifier entries are different paths rather than one answer spelled
twice. `amp-front` is the instrument input, so the amplifier's own preamp sits
in front of its speaker and the chain is stacked on a whole amplifier.
`amp-return` is the effects return, which bypasses that preamp and leaves the
power section and the speaker. A single `amp` would let somebody write one and
mean the other, which is why the contract refuses it.

**It does not say whether to use a cabinet block.** That is the question it
lets somebody ask, not the answer. A cabinet block is how a chain is made to
sound like a recorded rig, so somebody chasing a record may want one into a
real amplifier too, and somebody who wants their own amplifier to be the sound
may not. Both are reasonable and the difference is taste.

It is also not a tone correction, and there is no plan for it to become one.
Nothing here has measured an amplifier in anybody's room. Every figure this
project ships was measured through a cabinet block into a computer, which is
one of these four, and a reading is worth less to somebody on another. That is
what recording it is for.

## How this person plays

`technique`, optional, and the same schema a subject uses. `attack` is
required within it, one of `pick`, `fingers`, `slap`, `thumb`, `hybrid`.
`position` and `muting` are optional.

```yaml
schema: Setup
technique:
  attack: fingers
```

Here as well as on a subject because they are different claims. On a subject it
is how the player being emulated played. Here it is how the person holding the
instrument plays, and stating both is what lets the difference be compensated.

A pick puts high-frequency attack into every note that fingers do not, so a rig
tuned from a picked recording is duller played fingered. With both sides stated
the build adds a word on the attack axis, `audible-pick-attack` when this person
plays softer than the subject and `soft-attack` when harder, and says which and
why:

```text
playing the rig was played with pick and you play with fingers, so
        audible-pick-attack compensates for the front of the note, and
        mid-forward compensates for the mids
```

Three things about it are worth knowing before relying on it.

It compensates on two axes, and only one of them was a guess anybody could have
made. The front of the note is ranked: what is touching the string, flesh through
to a plectrum. The second is measured, from 468 notes played both ways on the
same instrument at the same pickup setting, and the surprise is which control it
reaches. A pick adds no treble at all. It moves 0.17 of the energy out of the low
band and into the mid, so the axis is `mids` and the word is `mid-forward` or
`scooped`. `pkg/sdk/audio/data/hands.json` is the figure and
`resources/dry/README.md` says where the notes came from.

A pair of hands nobody has measured compensates on the attack axis alone, which
is every pair but that one. The ranking is cheap and the measurement is not.

A word the ask already uses for either axis wins, and the report says which stood
down. Two terms on one axis cancel by design, so a second word would have taken
the ask's own out with it, and the two axes stand down independently: an ask
naming `audible-pick-attack` leaves the mids still compensated.

A thumb ranks with fingers rather than between fingers and a pick, because the
rank is about what is touching the string rather than about the tone.

Ask for it when writing a Setup. It is the one field here that changes what a
build does for the person rather than for the record, and somebody who does not
state it gets a preset tuned for whoever made the recording.

## Why a rig may not name what a device lacks

A rig used to carry a `requires` list naming impulse responses and bought models.
It is gone, and nothing in this repository ever read it: it sat in the contract
for months looking like a feature. A test now walks both contracts and fails on
any field no code names, so the next one cannot last as long.

What it was reaching for belongs to the person rather than to the rig. **Which
impulse responses are loaded is a fact about one pedal in one room**, and a rig is
the thing two people are supposed to be able to exchange.

## It does something, which is why it is worth writing

A preset stores the **slot number, never the audio.** So a chain depending on slot
82 sounds like whoever built it only if the same impulse response is loaded there.

That is why generated chains never reach for a user IR block: choosing one for
somebody who has loaded nothing picks a block that plays silence. A Setup naming
what it holds puts those blocks back in the ranking, and reading somebody else's
preset flags one.

Nothing else needs declaring. Whether a model exists on a device tier, or needs
newer firmware, the catalog already knows, because it carries the supported
device list and the release it came from.
