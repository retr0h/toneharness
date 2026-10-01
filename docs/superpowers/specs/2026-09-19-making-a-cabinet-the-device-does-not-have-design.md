# Making a cabinet the device does not have

2026-09-19

**Status: implemented.** `pkg/sdk/cab` captures a cabinet from a recording,
matches one to a target, and writes an impulse response a device loads. "Selling
one" below is reasoning about what may be sold rather than scope, and the one
buildable line in it is built: `write.go` puts the provenance in the file's
LIST/INFO chunk, naming what it was made from, by what method, through what, and
on what date.

## The ask

Build a cabinet or an amplifier Line 6 do not ship, and be able to sell it. And
short of that, when a rig calls for a cabinet nobody has, make one we do have
sound like it.

## What the device will actually load

The two halves of that ask are not the same, and only one of them is possible.

**A cabinet, yes.** An HX Stomp loads user impulse responses, at 1024 or 2048
samples, and this repository already knows those blocks:
`HD2_ImpulseResponse1024`, `1024Dual` and `2048`. An impulse response is a short
WAV file. It is a real, loadable, shareable, sellable artefact, and people do
sell them.

**An amplifier, no.** Line 6 publish no format for a user-made amp model. There
is nothing to write and nothing to load. Platforms built around capture exist
and this is not one of them.

So the honest shape is: **cabinets can be built, amplifiers can only be
approximated** — with an amp the device does have, plus the tone controls and an
equaliser in front of or behind it, which is exactly what
[the algorithm](../../algorithm.md) already does.

That distinction should not be blurred, because one half is a file somebody can
sell and the other is a preset.

### So, can this SDK build one

**A cabinet, yes.** Everything it needs is either here or small: the rig
measures, the arithmetic that turns a magnitude response into a loadable impulse
response is a few dozen lines, and the file is a WAV. The result loads on the
device and is a component rather than a preset.

**An amplifier, no, and not with more effort either.** The obstacle is not
difficulty, it is that there is no file to produce. What the SDK can do is make
an amplifier the device has, with the controls around it, measurably approach
one it does not have — and say by how much it missed, which is the part that
separates it from a claim.

Worth being exact about the difference, because the first is something to sell
and the second is something to load.

## Two different jobs

**Capture** is when the thing exists and you have it. A real cabinet, a real
microphone, a room. You play a known signal through it, record what comes back,
and deconvolve the recording against the signal to recover the impulse response.
That is the standard method, it is well understood, and it needs the physical
cabinet.

**Match** is when the thing exists and you do not have it. You have a recording,
or a measurement, or somebody else's cabinet you want yours to resemble. There
is nothing to deconvolve against, so instead you measure the target, measure
what you have, and close the difference.

The second is the interesting one here, because it is the one the measuring rig
already built makes possible.

## Matching, as a loop

The rig can already push a known signal through a chain and measure what comes
back. Matching is that loop with a different thing being adjusted:

```
   reference signal ──▶ our cabinet ──▶ measure ──▶ compare to the target
                             ▲                              │
                             └────── adjust the IR ─────────┘
```

And the adjustment is not trial and error in the sense of guessing. An impulse
response is a filter, and the difference between two magnitude responses is
itself a filter. So:

1. Measure the target's magnitude response.
2. Measure what our cabinet does to the same signal.
3. The ratio between them is the correction.
4. Build an impulse response carrying that correction, minimum phase.
5. Load it, measure again, and fold the residual back in.

Step four is arithmetic rather than search: given a magnitude response, a
minimum-phase impulse response with that magnitude is computed directly. Two or
three passes close most of what is closable, and the loop reports the residual
rather than declaring victory.

## What a match cannot fix

Being clear about this matters more than the method, because a matched cabinet
that is sold as the real thing is a lie somebody paid for.

**Phase and time.** A minimum-phase reconstruction has the target's magnitude
and not its phase. For a speaker that is usually close, because a speaker is
roughly minimum phase. For anything with a reflection, a room or a second
microphone at a distance, it is not, and the difference is audible as the sense
of space rather than as tone.

**Non-linearity.** A speaker compresses when driven hard and an impulse response
cannot, because a filter is linear by definition. Cabinet IRs everywhere share
this limit; it is why an IR of a cranked cabinet sounds polite.

**Whatever else is in the recording.** Matching against a record matches the
bass, the player, the room, the desk and the mastering along with the cabinet.
The result is a filter that makes your signal resemble that whole recording,
which is a legitimate and useful thing and is not a cabinet. Say which one is
being made.

A match derived from a Helix cabinet block is the clean case: the rig can
measure that block in isolation, so the only thing in the answer is the cabinet.

## Selling one

Worth saying plainly because it was part of the ask.

An impulse response **you captured from a cabinet you own** is yours. That is
what the people selling IRs are selling.

An impulse response **derived by measuring somebody else's commercial IR** is a
copy with extra steps, whatever the method. Measuring a competitor's product and
reproducing its response is not made acceptable by the reproduction being
approximate.

An impulse response **derived from a record** carries somebody's recording in
it, and the question of whether a filter fitted to a master is a derivative work
is not one to find out by selling it.

The tooling cannot tell these apart, so the provenance travels with the file:
what it was made from, by what method, and on what date. That is the same
discipline every claim in this repository already carries, applied to something
that leaves the building.

## Where this fits

The same measuring loop, pointed at a different question. Nothing new is needed
to measure; what is needed is:

1. **Deconvolution**, to recover an impulse response from a sweep. Needed for
   capture and useful for matching.
2. **Minimum-phase construction**, to turn a magnitude response into a loadable
   impulse response.
3. **Writing the file**, which is a WAV at the device's rate, 1024 or 2048
   samples.
4. **Loading it**, which is an IR slot on the device and a preset that names it.
   `NeedsUserIR` already exists because a generated chain must not reach for one
   blindly, and the same reasoning says a chain may reach for one deliberately.

None of it is large. All of it waits on the loop being able to reach a chain
block, which [measuring.md](../../measuring.md) says is still open.

## The name

This is the part that makes "toneharness" a better name than "tonestack". A tone
stack is a circuit; a harness is what holds a thing still while you measure it
and change it. The project stopped being a preset generator the night it could
hear its own output, and making a cabinet is the same loop pointed at making a
component rather than a setting.

## Related

- [algorithm.md](../../algorithm.md), which solves for settings the same way
  this solves for a filter
- [measuring.md](../../measuring.md), the loop both depend on
- [a genre is a corpus with a name](2026-09-19-a-genre-is-a-corpus-with-a-name-design.md),
  which is where a target comes from
