# sweeps

What each control actually does, measured rather than asserted.

One file per block. Each names the control, every position it was measured at,
the five figures at each, the rig's own repeatability at the time, and the whole
signal chain it ran through.

This is the data the guessing was standing in for. A character word used to move
a control by an amount somebody typed, and
[docs/algorithm.md](../../docs/algorithm.md) says what replaces that: stack
these curves into a matrix of slopes, and a request becomes a small linear
system rather than a search.

## A curve belongs to a chain, not to a control

The most important field in one of these files is `isolated`.

A slope is not a property of a control. The same Treble into a 4x12 and into a
1x15 are two different numbers, and so is the same amplifier with a drive pedal
ahead of it. So there are two kinds of measurement here and they are not
interchangeable:

- **A block curve**, taken with one block in the chain and nothing else. This is
  what belongs in a library, because it describes the block.
- **A chain reading**, taken in a preset that holds several blocks. Useful while
  tuning that preset, worthless afterwards.

`isolated: true` says which. The whole rig travels in the file beside it, so a
later reader can rebuild the chain rather than trust a sentence about it.

The size of the difference is not small. Measured alone, this amplifier's Treble
moves the centroid by 10,872Hz. Measured through its cabinet, in a chain the
previous sweeps had left with four controls at maximum, the same knob moved it
by 717Hz.

## Reading one

`us-dripman-norm.json` holds eight of this amplifier's twelve controls, each
alone in the chain, measured through the loop in
[docs/measuring.md](../../docs/measuring.md).

Per full turn of the control, with how much of the movement a straight line
accounts for:

```
  control       centroid Hz         level dB          low %         high %
  Norm Drive    -14663  0.59       +0.3  0.12      +78.4 0.58    -95.2 0.64
  Bass          -13649  0.88       +0.6  0.93      +95.8 0.91    -98.5 0.91
  Mid             -155  0.18       -0.1  0.71       -1.0 0.08     -0.8 0.08
  Treble        +12672  0.93       -0.4  0.27     -105.2 0.92   +102.6 0.92
  ChVol          +9561  0.90      +38.2  0.92      -75.7 0.90    +78.6 0.90
  Master         -2111  0.03       +2.7  0.02      +17.5 0.03    -20.2 0.04
  Sag            -4316  0.95       -4.2  0.85      +33.6 0.95    -34.7 0.95
  Hum              -69  0.39       -0.0  0.19       -0.0 0.00     +0.3 0.20
```

A column is what one control does. A row is every way to move one figure, which
is what the solver picks from.

Nothing in that table was written down in advance.

### What it says

**Bass and Treble are a tone stack.** Opposite signs, near-equal size, both
straight: `-13649Hz` against `+12672Hz`. One moves energy down and the other
moves it up.

**ChVol is the only volume control.** It moves level by 38.2dB per turn where
every other control moves it by under 5. Nothing was told that; it is the one
column where `level` is large.

**Mid barely acts, and acts precisely.** It moves the mid band by 2.2 points per
turn at a straightness of `1.00`, a perfect line, and does nothing else large
enough for a line to describe. That is a narrower and more useful description
than a character word could carry.

**Sag darkens and quietens, a lot.** The centroid falls 4,316Hz per turn and the
level 4.2dB, straight at `0.95`. Line 6 document it only as *"tighter
responsiveness for metal and djent"* against *"more touch dynamics & sustain"*,
with no direction and no size.
[docs/knowledge.md](../../docs/knowledge.md#which-way-a-knob-moves) counted it
among the largest controls in the catalog nothing could give a direction for.
This is the direction.

**Hum reaches nothing.** Every figure moved less than the rig's own wander:
0.048dB of level against a floor of 0.124. A control that cannot be aimed at is
worth recording as such, rather than being moved and hoped over.

**Master and Norm Drive are not lines, and the numbers say so.** Both sit at
0.03 to 0.6. Both are gain stages and both have the same shape: silent, then the
amp wakes up bright, then it thickens as the control goes further.

```
  Master   0.000   centroid    131  level -55.39   silent
           0.125   centroid    130  level -36.23
           0.250   centroid  10795  level -16.57   awake
           1.000   centroid   3769  level -24.88   thick
```

No single slope is true anywhere along that. `-2111Hz per turn` is the average
of a rise and a fall, and it describes neither. The straightness figure is there
so nothing uses it as though it were a slope, and every measured point travels
in the file so a solver can take the slope where the chain actually sits.

## What a number is worth

Each control carries the `noise` measured beside it, which is how far the figure
wandered across repeat takes with nothing touched. A move smaller than that is
reported as no move at all.

It is not a formality. Hum's largest reading is a centroid move of 107Hz against
a floor of 167Hz, which is the difference between "Hum darkens the amp" and "the
converters were breathing".

### Silence is not a small reading

A figure computed on silence is worse than a null result, because it is a
confident one. Two takes of silence agree to the last digit, so they clear the
noise floor more convincingly than music does, and the centroid of hiss is
broadband and reads high. A control that mutes the chain at one end of its
travel therefore reports an enormous, repeatable move.

The file this replaced said exactly that. It credited `Norm Drive` with a
46.43dB level move and concluded that Drive was the only control doing much to
level. The 46.43dB was the amp going silent at zero. Measured properly, Drive
moves level by 0.3dB per turn and `ChVol` moves it by 38.2.

Every position more than 30dB under the settled level is now marked `silent`,
left out of the fit, and listed in `muted_at` so the exclusion is visible rather
than a hole in a curve. Three controls here have one: `Norm Drive`, `ChVol` and
`Master`, all at 0.00, and all three are controls that can genuinely mute an
amplifier.

## These are measurements, not constants

A figure here describes **this block, through this rig, on this reference
signal**. Three things change it:

- **The reference signal.** Every measurement is a comparison against one fixed
  input, identified by hash in each file. A different one invalidates every
  number taken against the old one, the same way
  [changing a record invalidates the words derived from it](../../CONTRIBUTING.md#changing-a-source-is-never-one-file).
- **The chain.** Covered above, and recorded in every file.
- **The path.** These came through a cable from the pedal's output back to its
  own input, so a digital-to-analogue and an analogue-to-digital conversion sit
  in every reading. That is constant across a sweep and cancels when two
  settings are compared, which is what a sweep is for, and it is not zero.

So re-measuring on another rig will not reproduce these exactly. It should
reproduce their shape, and the shape is what gets used.

## What is missing here

Four of this amplifier's twelve controls. `Bright` is a switch and refuses a
float. `Ripple`, `Bias` and `Bias X` were queued and did not run: the pedal
dropped off the USB bus partway through the campaign and the last three sweeps
found no device. They need the pedal reconnected and the campaign re-run for
those three indices.

## Adding to them

```bash
just isolate "US Dripman" amp          # one block, alone, in slot 40
just identify 1 12                     # check the catalog's order against it
just sweep 1 3 --isolated --slot 40 --out /tmp/sweeps/p3.json
just fold /tmp/sweeps resources/sweeps/hx-stomp/us-dripman-norm.json
```

`--slot` reloads the preset before measuring, which matters more than it looks:
a live edit writes nothing back, so without it each sweep runs on a chain the
previous one left skewed.

A block's slot is its position in the chain plus one;
[docs/measuring.md](../../docs/measuring.md#addressing-a-control) has the layout
and the trap in it. A parameter's index is its position in the device's own
order, which is not the order `catalog show` prints;
[the same page](../../docs/measuring.md#which-control-it-actually-was) says why
that matters and `just identify` settles it against the hardware.
