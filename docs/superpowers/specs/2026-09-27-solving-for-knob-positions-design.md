# Solving for knob positions

**Status: designed, not built.** The sweeps exist for one amplifier and the
solver does not exist at all. Kept as a record of the method so the next session
does not re-derive it, and so nobody mistakes it for something the tool does
today.

How a request like "make it punk" becomes knob positions, without trying every
combination.
[Nothing here has ever heard anything](2026-09-18-nothing-here-has-ever-heard-anything-design.md)
argues why it has to exist at all. The parts of this that are true today live in
the skills: sweep discipline in `measure-a-device`'s `references/sweeping.md`,
what a target is and what a nudge is in `build-a-rig`'s
`references/correcting.md`.

## The space is too big to search

One amplifier, nine controls, five positions each:

```
  every combination :    1,953,125 measurements   226 days
  one at a time     :           45 measurements   7.5 minutes
```

Forty-three thousand to one. And five positions is coarse, nine controls is one
block, and a chain has several. Trying combinations is not slow, it is
impossible, and no amount of patience fixes it.

**So nothing here searches.** It measures what each control does, builds a
model, solves the model for the settings a target needs, and checks the answer.
The measuring is linear in the number of controls; the solving is arithmetic.

## What a sweep produces

One control, moved through its range, with everything else held and the same
[reference signal](../../../resources/dry/README.md) every time. At each
position, the nine figures.

That is a **curve per figure per control**: what Bass does to the low band, what
it does to the centroid, what it does to level. Not a single number — the whole
shape, which matters because the shape is rarely a straight line and because the
slope at one end is not the slope at the other.

A sweep also carries the rig's own repeatability, so a reading that moved less
than the loop's wander is reported as having moved nothing. Without that, the
last digit of a float looks like a finding.

**"Everything else held" is the whole difficulty.** Three things break it and
none announce themselves, because each produces numbers that look ordinary.
[measuring.md](../../../.claude/skills/measure-a-device/SKILL.md) has them in
full; the short version is:

- **The rest of the chain.** A slope is not a property of a control. The same
  Treble into a 4x12 and into a 1x15 are two different numbers, so a matrix
  belongs to the chain it was measured in. Measured alone, this amplifier's
  Treble moves the centroid by 10,872 Hz; measured through its cabinet in a
  chain the previous sweeps had left maxed, 717 Hz.
- **The previous sweep.** A live edit writes nothing back, so a control stays
  where the last sweep left it. The chain has to be reloaded between them.
- **Silence.** The repeatability check cannot catch a figure computed on
  nothing, because two takes of silence agree exactly. A control that mutes the
  chain at one end reads as an enormous move, and the move is the difference
  between hiss and sound.

## The model

Stack those curves and you have a matrix: how much each figure moves when each
control moves.

```
        Bass   Mid   Treble  Drive  ...
low     +0.8  -0.1    -0.2   +0.3
mid     -0.2  +0.9    +0.1   +0.2
high     0.0  -0.1    +0.7   +0.4
centroid -12    +3     +31    +18
...
```

Each column is one sweep. Each entry is a slope, read off the curve **at the
setting the chain is currently on** rather than averaged over the range, which
is what makes a curve worth keeping instead of a single derivative.

## Solving, rather than searching

A target is a point in those same nine figures. The current sound is another
point. The difference is what has to be closed.

```
   target − current  =  matrix × (how far to move each control)
```

Which is a small linear system, solved for the moves. Nine figures and nine
controls, so it is solved in the least-squares sense, preferring small moves
where several answers would do and refusing to move a control past its range.

That is the whole trick. **A search asks "what does this combination sound like"
a million times. This asks "which direction reduces the distance" once, and then
checks.**

### Then correct, because the model is local

The slopes are true near where they were measured and drift away from it, so the
first solve overshoots or undershoots. The fix is not a better model, it is
another step:

```
  measure → solve → apply → measure → solve → apply → ...
```

Each pass starts from where the last one landed and uses the slopes read at that
point. Three to five passes is usual, and each is one measurement plus
arithmetic. That is Gauss-Newton, and it is why a curved response does not need
a curved model.

**Stop when the residual falls inside the noise floor.** Not when it reaches
zero, which it never does, and not after a fixed number of passes. If the loop
cannot tell the current sound from the target, it has arrived.

## Where a straight line is the wrong shape

Two things break it, and both break in a knowable way.

**Drive and compression depend on level.** A tone control does roughly the same
thing whatever comes in; a drive does not, because it is the input level against
a threshold that decides everything. So those controls are swept at three input
levels rather than one, and their part of the model carries which level it came
from.

**A control before a non-linearity changes what the non-linearity is fed.**
Turning the bass up ahead of an overdriven amp is not an EQ move, it is a drive
move. The two interact, and no matrix of single-control slopes says so.

The cheap answer to the second is to **not measure it up front**. The correction
loop copes with mild interaction on its own: the slopes are re-read each pass at
the current point, which is exactly where the interaction has already happened.
Only when the loop fails to converge is an interaction worth measuring, and then
only the pair that failed. Measuring every pair in advance costs the
combinatorial explosion again to answer a question that usually does not arise.

## What it costs

For one block, nine controls, five positions, plus a noise floor: **fifty
measurements, about eight minutes.** For the fifteen blocks the curated rigs
actually use, about two hours. Overnight, unattended, once.

After that the model is on disk and answering a request costs the correction
loop alone: three to five measurements, under a minute.

Re-sweeping is only needed when the reference signal changes, which
[must not happen casually](../../../.claude/skills/measure-a-device/SKILL.md),
or when the device's firmware changes what a block does.

## What a target is

Three kinds of request, and each becomes a point or a partial point in the nine
figures. Each is also a field on the ask: `like` for a record or a player,
`genre` for a genre, and `nudges` for the fourth thing below, which is not a
target at all.

**A record.** "Like *Longview*" is the figures of that recording. Every one is
specified, and the target is a full point.

**A player.** "Like Mike Dirnt" is the median of his records, with the spread
across them as a tolerance: an axis his records disagree about is one the answer
need not be precise on.

**A genre.** "Punk" is
[where punk records sit against everything else](2026-09-19-a-genre-is-a-corpus-with-a-name-design.md).
And crucially it is a **partial** target: punk is distinctive on some axes and
ordinary on the rest, so only the distinctive ones are constrained and the
others are free.

That last point is what makes genre requests easier to satisfy than they sound.
A full point in nine dimensions may be unreachable with the controls a device
has. A target that pins three axes and shrugs at six usually is not, and the
freedom goes into satisfying the three.

### A nudge is not a target

"Darker" names no point. It is a direction from wherever the chain currently is,
applied to one axis, with the other eight held. Same solver, different
right-hand side: instead of `target − current`, the difference is a step along
one axis and zero everywhere else.

Conversation is made almost entirely of these, and a system that only
understands targets cannot take the instruction.

A word on the ask is the same shape of thing, said once rather than in reply:
`dark` is a step along the high axis from wherever the corpus left the control.
Both live on the ask because both are somebody describing a result, and both are
resolved against the chain the compiler built, since which control a word can
reach depends on which blocks are in the chain. What this loop replaces is the
size of the step. Today a word nobody measured moves a control by a fixed
fraction somebody chose, and once the sweeps are in the step is whatever closes
the gap in the figures.

## When it cannot get there

Sometimes the answer is that the device will not do it. A bass cabinet cannot
produce what a target asks for above 5kHz because the speaker stops, and no
combination of the controls in front of it changes that.

The loop finds this by converging to a residual larger than the noise floor and
then not improving. **That is a result, and it is reported as one:** which axes
were met, which were not, and by how far. The alternative — reporting success
because a file was written — is the failure this whole project exists to avoid.

A target that cannot be reached is also worth keeping, because it says something
true: the gear is wrong for the sound, which is a real answer to a person who
owns that gear.

## Checking the answer

The last step is not optional and is cheap: build the preset, run the reference
signal through it, measure what comes back, and compare to the target.

Three claims, and only the third is worth anything:

- *the rig validates against the catalog* — the file is well-formed
- *the device accepted it* — it is stored and readable
- *the sound measures where it was aimed* — it works

[The first two have passed while the third failed.](../../../pkg/sdk/internal/wire/README.md)
A preset written by this tool once rendered as an empty chain on the pedal while
every check the repository could make said it was fine. Measuring the audio is
what caught it, and measuring the audio is what closes every request from now
on.

## What is still missing

**Nothing, on the device side.** A live edit reaches every block, including
amplifiers and cabinets, so a sweep is one message and a measurement rather than
a preset write. That was open for an evening and was
[a self-inflicted wound](../../../.claude/skills/measure-a-device/SKILL.md): the
probing ran against presets a broken encoder had emptied, and an empty preset
still answers on the four structural slots, which reads exactly like chain
blocks being unreachable.

**Curves for anything but one amplifier.** The method is settled and the
measuring is the slow part: about eight seconds a reading, so a control at nine
positions with a repeatability check is a couple of minutes and a twelve-control
amplifier is most of an hour. Every block wanted in the library costs that once.

**The corpus side is done.** Three genres are tagged and measured, and
`measure genres` reports what each earns. It needed no hardware, and the result
is on [architecture.md](../../architecture.md): punk has the most records of the
three and earns nothing, so a threshold says a genre has enough behind it rather
than that it sounds like anything in particular.
