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
