# sweeps

What the device actually does, measured rather than asserted.

This is the data the guessing was standing in for. A character word used to
move a control by an amount somebody typed, and
[docs/algorithm.md](../../docs/algorithm.md) says what replaces that: stack
measured curves into a matrix of slopes, and a request becomes a small linear
system rather than a search.

Everything here is generated. Nothing in it is hand-written and nothing in it
belongs in Go source: a slope typed into a constant is the thing this whole
exercise removed. The files are the artifact, the same way the catalog is.

## Two files, answering two questions

`fingerprints.json` says **what each block sounds like**. One reading per
block, sitting at its own defaults, for all 665 of them. It is what picking a
block out of 665 needs: ranking 224 amplifiers by how close each sits to a
target turns "sound like this record" into a shortlist.

`<model>.json` says **what one block's controls do**. Every float control,
swept across the range the catalog gives it, folded into a matrix of slopes.
It is what setting a chain needs once the chain has been picked.

The split is a cost one. A control is about two minutes and a twelve-control
amplifier most of an hour, so the device's 4,835 float controls are a hundred
and sixty hours. A fingerprint is one reading. So every block gets a
fingerprint and the blocks a chain reaches for get a matrix.

It is also the right split. [algorithm.md](../../docs/algorithm.md) measures
its matrix fresh for whatever chain is being tuned, because a slope belongs to
its chain, so a stored matrix for every block would be rebuilt before anything
used it. What cannot be worked out at solve time is which blocks belong in the
chain at all.

### Reading a fingerprint

Every figure is a difference from the empty loop, which is measured first and
stored as `baseline`. Without it a number says nothing: 95 Hz is not what an
equaliser does to a bass, it is what the bass already was, and an equaliser
flat at its defaults passes it straight through.

Two flags travel with each block:

- `clipped` is a reading that hit the converters' ceiling. Its spectrum is the
  clipping's rather than the block's, because flat tops make harmonics that
  were never in the signal, so it reads as a bright block and is not one.
- `refused` is a block that would not load alone, with what the device or the
  compiler said. Some of what the catalog lists means nothing on its own.

**A fingerprint is not a verdict.** It says where a block sits with nobody
touching it, and a block that does nothing at its defaults may still do a
great deal once its controls are moved. The equalisers are the clear case:
every one of them measures as the baseline, because flat is what they ship at.
Their curves are the only thing that describes them, and those cost a
campaign.

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
just fingerprint --resume              # every block, one reading each
just campaign HD2_AmpUSDripmanNorm amp # one block, every control
just fold /tmp/campaign/HD2_AmpUSDripmanNorm \
    resources/sweeps/hx-stomp/us-dripman-norm.json
```

One control at a time, when a campaign is the wrong granularity:

```bash
just isolate "US Dripman" amp          # one block, alone, playing
just identify 1 12                     # check the catalog's order against it
just sweep 1 3 --isolated --preset /tmp/isolated.hlx --out /tmp/sweeps/p3.json
```

`--preset` plays the chain again before measuring, which matters more than it
looks: a live edit writes nothing back, so without it each sweep runs on a
chain the previous one left skewed. It goes through `presets play` rather than
a slot, because a slot is flash and a campaign loads a chain once per control.

A control's range comes from the catalog unless `--low` and `--high` say
otherwise. That is not a nicety either: 1,452 of the device's 4,835 float
controls do not run zero to one, and a Simple EQ's Mid Freq swept 0..1 never
leaves its bottom stop and reports as a control that does nothing.

A block's slot is its position in the chain plus one;
[docs/measuring.md](../../docs/measuring.md#addressing-a-control) has the layout
and the trap in it. A parameter's index is its position in the device's own
order, which is not the order `catalog show` prints;
[the same page](../../docs/measuring.md#which-control-it-actually-was) says why
that matters and `just identify` settles it against the hardware.
