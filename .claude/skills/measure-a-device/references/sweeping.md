# Running a sweep

Moving a control and measuring the result is easy. Getting a number that means
anything is where the work is, and four things have to be true. Each was got
wrong first, and each produced numbers that looked fine.

## A slope is not a property of a control

The same Treble into a 4x12 and into a 1x15 are two different numbers. Put a
drive in front and they change again. So there are two artifacts and they are
not interchangeable:

|                  | measured on                                     | reusable                 | cost                                   |
| ---------------- | ----------------------------------------------- | ------------------------ | -------------------------------------- |
| a block curve    | one block, nothing else in the chain            | yes, this is the library | one sweep per control, once            |
| a chain Jacobian | the preset being tuned, at its current settings | no                       | one measurement per control, per solve |

The size of the difference: measured alone, one amplifier's Treble moves the
centroid 10,872 Hz. Measured through its cabinet, in a chain the previous sweeps
had left maxed, **717 Hz**. Both are correct about different things.

```bash
mise exec -- go run main.go measure controls --model HD2_AmpUSDripmanNorm --json
```

That builds the first: a rig holding one block, compiled and played. **Every
sweep records the chain it ran through**, read back off the device with
`device current`, and marks whether that chain held anything else.

## Sweeping a block with another behind it

`--after` names a model to put after the one being measured, so the second
artifact in that table can be measured deliberately rather than only as a
by-product of a solve:

```bash
mise exec -- go run main.go measure controls \
        --model HD2_AmpSVBeastNrm --after HD2_Cab8x10SVBeast
```

A model identifier, not a gear name, for the same reason the sweep itself takes
one: 661 models share 468 names, so "Ampeg SVT" matches both of that
amplifier's channels. The catalog is checked before anything is compiled, and a
block cannot follow itself, because both would answer to the same name and a
curve filed under a control would be a curve of whichever of the two the sweep
reached.

Three things change in the file, and all three are the same point. `isolated`
goes to `false`, so nothing downstream reads the result as a fact about the
block. The pre-flight checks the device loaded **both** models rather than one,
because a chain the device could not build is a chain it answers by playing
what it could, and a dropped cabinet would leave the amplifier measured alone
while the file claimed otherwise. And the readings land in
`<model>-into-<after>.json` rather than `<model>.json`, because what reads
these back asks for one file per model: a chain sweep under the block's own
name is handed to the solver as that block's isolated reading, and the filename
is the only thing standing between those two.

What it is worth measuring: an amplifier swept with nothing in front of it has
no speaker rolloff, so what comes back is the buzz a cabinet exists to absorb.
43 of the 121 usable curves hold a position reading more than half its energy
above 2kHz and all 43 are amplifiers.

## A sweep leaves its control where it finished

A live edit writes nothing back, so a control stays at the top of its range.
Sweep a second control after that and it is measured on a chain the first one
skewed; sweep eleven and the eleventh runs on an amplifier with four controls
pinned at maximum.

Playing the same file again replaces the edit buffer and undoes every move, so
the chain goes back in front of the device before every control. Through
`device play` rather than by reselecting a slot: twelve sweeps would otherwise
spend twelve flash writes putting a chain back. See
[device-care.md](device-care.md).

## Silence is repeatable, so the noise floor cannot catch it

The floor catches a figure that wandered. Two takes of nothing agree to the last
digit, so silence clears it more convincingly than music does.

What comes out is not a null result, it is a confident one. The centroid of hiss
is broadband and reads high, so a control that mutes the chain at one end
reports an enormous move. **2,959 Hz was filed as an amplifier's Drive moving
the centroid; 2,959 Hz is the difference between hiss and sound.** A reading more
than 30 dB under the settled level is now marked `silent` and left out of the
totals, and a sweep with fewer than two positions left refuses to report a curve.

## A reading that moved less than the rig's wander moved nothing

`--takes` is how many takes the floor is measured from, and the floor travels
with the curve. A sweep carries the rig's own repeatability, so anything under it
is reported as no movement. Without that, the last digit of a float looks like a
finding.

**A control's range comes from the catalog**, which is not a nicety: 1,452 of the
device's 4,835 float controls do not run zero to one, and a Simple EQ's Mid Freq
swept 0..1 never leaves its bottom stop and reports as inert.

## A reading of a block the device never loaded looks exactly like a reading

Playing a chain is not the device holding it. The pedal accepts a preset, draws
something else, and the loop measures whatever is actually in the slot, so what
comes back is a number with the right shape and the wrong subject.

The committed library shipped 46 cabinets that way, every mic'd cabinet with a
Pan control. All 46 read within 0.4Hz of each other, with identical band shares
and a level identical to two decimal places, while the 87 plain cabinets spread
across 92.8Hz. They were not readings of 46 cabinets: they were 46 readings of
the chain with nothing in the slot, which on this rig is -56.56dB with 93.7% of
its energy low.

What makes it findable is that the same reading is correct for a block that
really is transparent. 34 others share it — the mixers, the splits, the sends
and returns, an equaliser at its flat defaults, Volume and Gain at unity, a
Pitch Wham heel down — and for those it is the truth. **A cabinet is never
transparent.** A cabinet measuring identically to a Mixer did not render.

Both sweep commands now play the chain and read it back before trusting a
figure, and record a refusal naming the model where the device is holding
something else. A figure taken before that guard existed is a figure to check
against its own neighbours: if several different blocks agree to two decimal
places, they are one reading.

## A chain that dies partway through takes the silence floor with it

The floor under which a reading is hiss rather than signal is thirty decibels
below what the chain settles at, and that settled level is read again before
every control. So a chain that stops passing signal midway drags the floor down
after it, and every reading from then on clears a floor that is itself silence.

`6x10 Cali Power` shipped that way. Its Mic swept first, settled at -54.08dB and
read 105 to 134Hz, which is a cabinet. Then:

    control    settled   its floor   lowest reading   centroid
    Mic         -54.08      -84.08           -54.09   105..134
    Position   -129.20     -159.20          -131.63   908..1891
    LowCut     -130.82     -160.82          -133.42  1862..3083
    Level      -130.86     -160.86          -157.82  1814..1849

Six controls of hiss on a bass cabinet, up to 77% of their energy above 2kHz,
every one clearing its own floor by thirty decibels.

Each control is now held to what the chain settled at before the **first** one,
which is the chain the seated pre-flight has just watched load. A later control
settling further under that is refused, not noted: there is nothing to learn
from the rest of a sweep whose chain has stopped, and the block is worth
retrying whole.

**Half of the `FM4` models do this.** Seven of the fourteen, across the filter
and pitch categories, against one block in roughly a hundred everywhere else.
It is not deterministic — five different controls on five blocks, every one
clean on the next attempt — so **retry is the remedy and a campaign should give
that family extra attempts** rather than treating a first failure as a block
that cannot be measured.

## A level that came back wrong is a different rig

`--volume` pins the computer's own output before anything is measured, and it is
a tone control rather than a level control: an amplifier's distortion depends on
how hard it is driven, so two runs at different levels measure two different
amplifiers.

It is set and then read back, because macOS takes the instruction and rounds it
on some hardware. Twice in about 130 sweeps it came back at **0** — the output
muted — and the figures that followed read -122dB with centroids between 1630
and 2390Hz. Hiss, recorded as a filter.

A level that comes back as something other than the one asked for is a signal
path that is not the one the campaign is filing its readings under, so the sweep
stops. A platform that will not say what its level is carries on, because
nothing can be checked there and the file records that rather than a number.

## What a sweep's own file lets you check afterwards

Four things, and each caught something the others did not:

| check | what it catches |
| --- | --- |
| every control holds readings | a sweep that stopped early |
| `instrument`, `volume`, `isolated` are set | a reading from another rig |
| every control settled within 30dB of the first | a chain that died midway |
| blocks of different gear share a reading | a block that never rendered |

Run all four before committing a batch. On this campaign the third rejected a
filter the first two passed, and the fourth is what found the 46 cabinets whose
readings were of an empty slot.

### `instrument` is now checked, not just recorded

The second check used to be hygiene. It is not any more: a reading is refused
for a chain built for another instrument, and a reading that cannot say which
instrument it was taken on is refused too. Five places ask, and the refusal
reaches the target, the block ranking, the pair library, the control tuning and
a cabinet's microphone.

So a sweep that lands without the field is a sweep whose readings nothing will
use. Check it before committing rather than after.

There is one fallback and it is not a guess. Where the field is absent the
reference recording's filename answers, because that was the only trace of it
before the field existed and 8 of the 215 readings committed here predate it.
Every one of those names `bass-di.wav`. Refusing them outright took an EQ's
controls off a preset and put the catalog's defaults back, which is a worse
answer than the one the file can still give. Name a reference recording after
the instrument it holds and that fallback keeps working; name it `take-3.wav`
and it does not.

## The fingerprint library says which run each reading came from

A curve file describes one sweep, so one header describes every reading in it.
The fingerprint library does not: `measure blocks --resume` skips what is
already in the output and adds to it, and the fields that decide what a figure
is of all lived on the library. So a file assembled over several runs described
every block in it with the **last** run's rig, volume, pinned output and
baseline, including blocks measured through none of them.

Each reading now carries `run`, a key into the library's `runs`, and the key is
a digest of what makes readings comparable: the reference, the instrument, the
headroom, the volume, the rig, whether the loop was open, the device the level
was pinned on, and the baseline. Two sessions alike on all of those collapse
into one entry, which is right, because downstream they are one run.

`--resume` freezes the earlier file's own header onto the readings it carries,
rather than letting this run's header claim them. It keeps a claim the file
already made instead of replacing it, and `measure blocks` says at the end when
a file has come to hold more than one run, since not every block in such a file
may be ranked against every other.

**The 614 committed fingerprints are deliberately left unstamped.** They predate
the field, and the header describes the run that wrote the file last rather than
all of them, so stamping them would turn a known silence into a definite-looking
claim. `Library.RunOf` answers for them from the header and returns false, which
is how a caller tells a record from a fallback.

A `--resume` that mixes two rigs is still refused outright, because a library
whose blocks were measured on two pedals is one nothing may rank inside at all.
The stamp is for the case that refusal cannot see: a file that says nothing
about its own rig.

**The fourth needs stating carefully, because two models of one pedal are
supposed to read alike.** `HD2_AmpAguaSledge` and `HD2_PreampAguaSledge` are the
amplifier and the preamplifier of one Aguilar Tone Hammer, and they agree to
within 0.01Hz on every control, which is correct and not a fault. Rounding the
comparison to two decimals calls them identical and flags them.

What the signature actually is: **many blocks of unrelated gear agreeing**. The
46 cabinets spanned 0.4Hz where their mono twins spanned 92.8Hz, and 80 blocks
across six categories shared one reading. Two models of one pedal agreeing in
the third decimal is the device being consistent.

## A control cannot be guessed from its name, and triage saves nothing

The question a campaign asks when a firmware release adds models: now that so
much is swept, can the new blocks be predicted rather than measured?

No. Tested by leave-one-out across the 55 control names appearing on four or
more swept blocks — predict each block's control from the median of the others:

| control | blocks | span across them | a guess is off by |
| --- | --- | --- | --- |
| `Drive` | 49 | 0.1–352 Hz | 78% |
| `Gain` | 27 | 0.0–254 Hz | 76% |
| `Mix` | 21 | 2.4–8292 Hz | 71% |
| `Treble` | 42 | 0.7–593 Hz | 61% |
| `Mid` | 24 | 3.9–269 Hz | 60% |

Eighteen of the 55 predict within 25%, and that number flatters the method:
most of those are controls whose median span is **0.0 Hz**, so what is being
predicted correctly is that the control does nothing.

**That is a trap rather than a shortcut.** `Level` reads a median of 0.0 Hz
across 76 blocks and seven of them move it as far as 774 Hz. `Output` has one at
221 Hz among seventeen. Nothing in the name says which block is the exception,
so a guess is right until it is badly wrong, and the blocks it is wrong about
are the interesting ones.

The only non-zero controls that predict well are the `Ch1 *` family at 3 to 10%,
and all four of those come from one amplifier's channels. That is four readings
of related gear, not four independent ones.

**And a cheap triage pass is not worth running.** Three points over everything,
then nine on whatever moved, costs 16.5 hours against 17.0 for nine points on
everything — a 3% saving — because **64% of controls (510 of 800) move the
centroid by more than ten times the noise floor.** Most of the device is a lever,
so the pass that finds the levers nearly measures everything anyway.

**So sweep new blocks, at nine points.** It is cheap per block: a five-control
synth is about seven minutes.

What the library does buy is knowing **which** blocks are worth sweeping, and
that is where the saving is. A bass build can reach 293 of the device's 661
blocks, but the 43 shipped rigs between them hold only **21 distinct blocks**,
and sweeping those 21 covers 87% of the controls in every shipped chain. Sweep
what the rigs hold before sweeping what a build could choose.
