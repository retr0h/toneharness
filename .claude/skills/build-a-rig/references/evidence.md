# Claims and citations

The only knowledge in this repository that is ours is what a rig says about
real gear. Everything else is generated. So the standard for a claim is the
standard for code.

## Which claim do you have

Three, and only the first is currently possible here:

1. the rig validates against the catalog
2. HX Edit imported the file
3. the hardware loaded it and it sounded right

Never report one as another. Never describe work as verified on evidence you
did not gather. **If you did not run it, say you did not run it.**

## Evidence travels with the claim

Per claim, not per document: the amplifier may come from an interview and the
drive figure from measuring a corpus. A chain entry keeps its own evidence for a
reason: a published rig that cannot say why this amplifier is in it is a rig
nobody can check.

The kinds a claim may carry, strongest first, are in the contract's
`description:` fields. Read them there rather than from a list here, because the
list has changed and a copy gives no sign when it goes stale.

`kind: llm` is honest and weak. Use it rather than dressing an assertion as a
citation, and say in `caveat` that it is not sourced to anything anybody
opened.

## A wall in the research is not a reason to drop the subject

Some players have no citable gear. The search bottoms out at Wikipedia, or the
threads are enthusiasts discussing the playing, or every sighting is from the
audience and none is dated to the record. That is an ordinary outcome, not a
failure, and the answer is **not** to leave the player out.

Leaving them out loses the work. Nobody can see what was searched, nobody knows
the gap exists, and the next person starts from nothing. Write the claim as
`kind: llm` with `confidence: low`, name in `caveat` what was searched and where
it ran out, and invite the correction:

```yaml
- role: amp
  gear: Ampeg SVT
  evidence:
    - kind: llm
      note: >-
        asserted by a model and confirmed by nobody. The period rig is not
        established here.
      caveat: >-
        searched the archived bass magazines, TalkBass and the published
        interviews, and found only sightings from the audience, none dated to
        this record. The "according to the man himself" quote repeated
        everywhere is a forum post quoting Wikipedia, which cites nothing.
        If you know what was actually used, please open a pull request.
  confidence: low
```

`substitute` is the other half where the device models nothing close: it says
what to put there instead, and keeps the real name as the claim.

An unsourced entry labelled `llm` is checkable and correctable. An absent player
is neither. The thing this project refuses is a guess **wearing a citation**, not
a guess that says what it is.

**When a page is dead, cite the archived copy and pin it.** Most of the richest
sources here are `web.archive.org` snapshots of magazines that no longer exist.

## Where the gear cannot be identified, offer a sound-alike

The step above keeps the player and labels the guess. There is a better answer
where the research has genuinely been done and bottomed out, and it is to stop
attributing the rig to the person at all.

**Measure their records and let the measurement pick the amplifier**, then
attribute the document to a sound rather than to them. The subject takes
`kind: sound`, the evidence takes `kind: audio`, and the caveat says in its first
line that this is not a claim about what anybody played. It files under
`sounds/` in the tier, beside `artists/` and `genres/`.

The mechanism is already there and needs no hardware. A build aimed at a
recording ranks the device's amplifiers against it:

```bash
mise exec -- go run main.go tone build --ask <ask naming like.recording>
#   amp  Del Sol 300 is the closest of 27 measured to …/bass.wav, 98 Hz against 120
```

Point it at the player's separated stems, **one record at a time**, and read
whether their own records agree. That agreement is the finding and it belongs in
the note. Three sound-alikes were written this way and not one was unanimous:
two of three records each for ZZ Top's Tres Hombres, the mid-seventies Marshall
Tucker albums and Sum 41's All Killer No Filler, while Berry Oakley's three
records picked three different amplifiers and Dan Andriano's did the same. Where
the records disagree that badly, write nothing.

**Say what is weak about it, because two things are.** The records disagree, and
the note has to say how. And the winners cluster at the dark end of what the
device has: the Acoustic 360 reads 91Hz and the Sunn Coliseum 300 reads 98Hz,
the two darkest bass amplifiers it models, and a record sitting below most
candidates picks whichever is darkest rather than whichever is nearest. That is
the pathology the measured library's own notes record. A sound-alike is
buildable and honest, not settled.

Which to reach for:

| the state of the research                                  | what to write                          |
| ---------------------------------------------------------- | -------------------------------------- |
| gear cited, and this device models it                       | the rig, in `core/`                    |
| gear cited, and this device models nothing like it          | the rig, in `community/`, substitution named |
| gear not established, and the records agree                 | a sound-alike under `sounds/`          |
| gear not established, and the records disagree              | the `llm` entry above, saying so       |

**A sound-alike never carries the player's name as its subject.** Name it for the
record or the era. Putting their name on an amplifier the measurement chose is
the same false claim as guessing, with arithmetic in front of it.

## A rig is not a notebook

It carries claims and their sources. An absent field already says nobody
established it, so it needs no paragraph explaining the absence.

The exception is the one above: where a claim is written on a model's say-so,
what was searched and where it ran out belongs in that claim's `caveat`, because
it is what tells the next reader the gap is known rather than overlooked.

Full standard: [Sourcing a rig](../../../../CONTRIBUTING.md#sourcing-a-rig).
Field meanings: the `description:` fields in
`pkg/sdk/tone/data/tonespec.openapi.yaml`.
