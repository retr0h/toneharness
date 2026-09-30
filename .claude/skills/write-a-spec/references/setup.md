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
