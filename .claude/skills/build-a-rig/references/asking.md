# Write the request

Two documents, and the ToneSpec holds two halves.

|                  | Holds                              | Changes                         |
| ---------------- | ---------------------------------- | ------------------------------- |
| **Setup**        | what somebody owns                 | when they buy something         |
| ToneSpec `ask:`  | what they want this time           | every request                   |
| ToneSpec `rig:`  | the gear those two resolve to      | every request, or hand-authored |

The Setup stays a separate file because folding the instrument into the ask would
mean restating their bass in every request, and the twelfth one would contradict
the first. The ask and the rig travel together because neither is much use
without the other, and the rig is required.

A researcher writes a rig by hand too, when the gear is the thing being
established. What every field may say, and which half it belongs in, is the
`write-a-spec` skill's; this page is only how a request gets resolved.

## What a ToneSpec may say

Every field is optional and any combination is legal. The contract is the only
authority: the `description:` fields in
`pkg/sdk/tone/data/tonespec.openapi.yaml`.

**It never carries a knob position.** Decided 2026-09-19. "More drive" is a
request; `drive: 0.7` is a plan, and a number is only meaningful against a
chain. Words go in `words` or `nudges` and are resolved against measured
corpus data later. Inventing a number here is the guessing this project exists
to remove.

`gear` entries name real-world gear the way a person says it. `insist: true`
means refuse rather than substitute.

**Order is not what you typed.** Gear is sorted into the ordinary signal path,
because listing gear is not stating one: a compressor belongs in front of the
amplifier whichever way round it was written. Where somebody means otherwise,
`after: amp` on an entry puts it behind that role. A drive behind the amplifier
is the case worth knowing, and it is a known way to use one rather than a
mistake.

Named by role rather than by position, because a request does not know how many
blocks the chain ends up with: "after the amp" survives the compiler adding a
cabinet and "position 4" does not.

## Which instrument a chain is for, and what it refuses on the strength of it

The Setup first, because what somebody owns is a fact. Then the ask's own
`instrument:`, which is the one the Setup cannot answer: a request carrying
`instrument: guitar` and no Setup used to be answered "the setup names none, so
this is for bass" and built a bass rig. The contract's own description of the
field is "what this is for, where the request asserts it rather than leaving it
to the Setup". A bass where neither says, and the report says which of the three
happened.

**And the two evidence tiers above measurement refuse gear from the other
instrument.** `translate.genreTarget` had that guard on a genre's *figures* and
nothing had it on its *gear*, so a guitar ask naming hard-rock was told "bass
sits about an octave from guitar and the solve would spend every control getting
there" in the same breath as being handed the Ampeg SVT three of hard-rock's
five documented players used. Gear that is cited for the wrong instrument is a
worse answer than none, because every claim on it is true.

So a guitar ask naming a bass genre now gets this and nothing else:

```text
could not  hard-rock  its players are measured on bass and this is for guitar,
                      so none of their gear was taken either
could not  hard-rock  that genre is measured on bass and this asks for guitar,
                      so its figures are not a target for this
```

**And a player is refused the same way, which took a second pass to notice.**
The guard was on a genre's figures and a genre's gear, and `like: artist:` went
straight past both: an ask refused hard-rock's figures for being measured on a
bass was handed a hard-rock bassist's figures in the same breath by naming
them. A centroid is about an instrument before it is about a player.

**And so is the reading of the pedal itself.** This is the one that was easiest
to miss, because it is not about the corpus at all. Every target ends as a
displacement from the library's own empty loop, and that loop is a bass through
the pedal: 147.9Hz with 93.9% of its energy low before a block touches it. A
guitar does not start there, so there is nothing for a guitar chain to be
displaced from, and a guitar ask aimed at a handed-over recording now refuses
rather than subtracting the wrong baseline from it. The same goes for a control
set from a sweep and a cabinet's microphone chosen from one: all 215 committed
readings were taken with a bass, and each is refused for a chain built for
anything else.

A reading that cannot say which instrument it was taken on is refused too,
which is `measure-a-device`'s `sweeping.md` to explain and not this page.

### What does work for a guitar, measured rather than assumed

A guitar rig whose gear is authored builds all the way to a preset. One was
written naming a Marshall JCM-800 2203 and a 4x12 1960 AT75, and the chain came
out as five blocks at 54% of the chip with every control set:

- **the catalog is not bass-only.** 173 of its 224 amplifiers are guitar models
  and `reachable` already draws from the right half
- **the preset corpus is not bass-only either**, which is the thing task 266's
  own survey missed. It filled the chain from guitar statistics: a Plateaux
  reverb because "almost every chain has one (89% of chains)", a Scream 808 and
  a Simple Delay bypassed because 77% of chains carry one and under half run it
- **the controls came from guitar corpus medians**, 13 of them on the amplifier,
  rather than from a target. Correct: there is no guitar target, so nothing
  aimed and nothing guessed

What is missing is the *measured* layer and only that: no guitar corpus of
records, no guitar-referenced sweep, no guitar genre. A guitar rig is as good as
its citations today and cannot be aimed at anything, and the refusals above are
what makes that a stated answer rather than a silent bass one.

### One trap left in place

`gearIsFor` reads any Setup gear name without "bass" in it as a guitar, so a
Setup naming a Rickenbacker 4001 or a Fender Jazz answers guitar for an
instrument that is a bass. It is recorded in task 266 rather than fixed here,
because the fix is a list of names or a question for the catalog and neither is
this change.

## Resolve it

```bash
mise exec -- go run main.go tone build \
  --ask request.yaml --setup mine.yaml --json
```

**Read the notes.** They are in the answer, not printed above it. They say
what it substituted, what it assumed when no setup was given, and what it
could not honour. A run reported without its notes is a run that hid half of
what happened.

Worked examples:
[docs/examples/](../../../../docs/examples/), every
`.yaml` there but the Setup and the Plan.
