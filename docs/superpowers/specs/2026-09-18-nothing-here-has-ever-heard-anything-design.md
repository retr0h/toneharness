# Nothing here has ever heard anything

2026-09-18

**Status: implemented.** `pkg/sdk/reamp` pushes a signal through the pedal and
keeps what comes back, and the `measure` commands are what read it.

This supersedes
[everything in a rig reaches the pedal](2026-09-18-everything-in-a-rig-reaches-the-pedal-design.md),
written this morning, whose second half should not be built.

## What happened

A day of work on making every field in a rig reach the device ended with a
question that undid it: if nothing here can hear, how does anybody know what any
of these words mean?

The answer is that nobody does. The words were reasoned about, not measured.

## What is solid

Measured, reproducible, and checkable by anyone holding the same files:

- **Records to numbers.** `pkg/sdk/audio` reads audio and returns nine figures:
  `low`, `mid`, `high`, `centroid`, `transient`, `decay`, `dynamics`,
  `harmonics`, `lean`. Pino Palladino's centroid is 96Hz against 172Hz for the
  other eight players. That is a fact about those files.
- **The sourcing.** Who played what, when, and on whose authority, with the page
  cited and read.
- **The era check.** A figure derived from a record made on other gear is caught
  rather than believed.
- **Gear to models.** A name a person uses resolves to something the device
  carries, or says it cannot.

## What is guessed

Everything between a word and a knob.

- `dark` moves Treble down by one step. The step is a quarter of the range, or
  the corpus spread where there is one. Nobody established that a quarter of the
  range is what `dark` means.
- `tight-low-end` moves Sag, on the strength of Line 6's own description of the
  control. That is a manufacturer's prose, not a measurement.
- The ten axes and their words are a vocabulary somebody wrote down. No figure
  defines any of them.

**Nothing has ever verified that moving a knob moves the figure it was moved
for.** That is the whole problem, and it sat in the task list for weeks marked
"needs pedal", as though it were a detail rather than the foundation.

## What is missing, precisely

One link:

```
a dry bass  →  the device  →  measure the output  →  compare to the record
                   ↑                                        │
                   └────────── adjust and repeat ───────────┘
```

The right-hand side exists. The left-hand side exists. Nothing has ever pushed a
signal through the device and measured what came out.

## What a word should be

A word should name a region in the nine figures, not a direction on a knob.

"Chunky" is not "Bass up, Treble down". It is something like: high `low`, low
`high`, slow `transient`, long `decay`. Once that is written as numbers it can
be argued with — a preset either lands in the region or it does not, and if it
lands there and sounds wrong then the region is wrong and gets moved.

That is the feedback loop. Without it a word is a label nobody can check.

## The measuring rig

1. A dry reference recording of one bass, direct, no amplifier and no pedals.
   Chromatic up each string covers the range the instrument actually produces.
2. Push it through a known set of parameters. Capture the output.
3. Measure the output with the same nine figures the records are measured with.
4. Change one parameter. Repeat.

The result is what every control actually does, in the same units the records
are described in. After that, a word can be defined against it.

### Hardware or software

An HX Stomp is a USB audio interface, so the loop runs through the hardware in
real time, with it plugged in and a person nearby.

Helix Native is the same models as a plugin. If it can be driven from a script
the loop runs offline, thousands of combinations overnight, no pedal.
`pedalboard` loads VST3 plugins from Python and would be the host.

Two things decide it and neither is known yet: whether Native exposes its
per-block parameters to a host, rather than only a handful of generic ones, and
whether its licensing permits headless use. Both are answerable with a trial
before any money is spent.

If Native cannot be scripted the work still happens, on the hardware, slower.

## What this costs

RigSpec is too complicated, and most of what makes it complicated is the
vocabulary machinery: ten axes, twenty-four terms, a map of which control each
one reaches and how far. If a word becomes a measured region, that machinery is
replaced rather than extended.

What survives is the part that says what to aim at: the records, their
measurements, the gear, and the evidence for all of it.

## Decisions made today that should not be built

- **Deleting `decay`, `string-noise` and `pickup` from the vocabulary.** The
  argument was that no device models them. That was wrong for decay at least:
  `decay` is one of the nine figures already measured. The terms were never
  unmeasurable, only unwired.
- **Deleting `played[].strings`.** Strings are actionable, as a difference
  rather than as a fact: a preset exists to make one person's bass sound like
  somebody else's record, and if the record was cut on flatwounds and the player
  has roundwounds then the difference is a knob position. What is missing is the
  other half, which is what the person running the preset has. Nothing in this
  format describes them, because the format describes artists.

Neither was committed.

## Next

Establish whether Helix Native can be scripted. That single answer decides
whether the measuring rig is an overnight job or a hands-on one, and nothing
else should be built until the loop exists.
