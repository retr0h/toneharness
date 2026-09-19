# sweeps

What each control actually does, measured rather than asserted.

One file per block. Each names the control, every position it was measured at,
the nine figures at each, and the rig's own repeatability at the time, so a
reading that moved less than the noise can be told from one that moved.

This is the data the guessing was standing in for. A character word used to move
a control by an amount somebody typed, and
[docs/algorithm.md](../../docs/algorithm.md) says what replaces that: stack
these curves into a matrix of slopes, and a request becomes a small linear
system rather than a search.

## Reading one

`us-dripman-norm.json` holds the first, four of the amplifier's controls,
measured through the loop in [docs/measuring.md](../../docs/measuring.md):

```
  figure       Norm Drive         Bass          Mid       Treble
  centroid        -289.30     -2151.07      -312.68       592.61
  level             46.43        -2.60         0.04         1.17
  low               -0.24        64.48         9.46       -17.96
  mid               10.13        12.78         1.68        -2.10
  high              -9.89       -77.26       -11.14        20.06
```

Per full turn of the control. A column is what one control does; a row is every
way to move one figure.

It reads like the physics: Bass moves energy down the spectrum and Treble moves
it up, both of them far more than Mid, and Drive is the only one that does much
to level. Nothing here was written down in advance, which is the point.

## What a number is worth

Each control carries the `noise` measured beside it — how far the figure
wandered across repeat takes with nothing touched. A move smaller than that is
reported as no move at all.

It is not a formality. Sweeping the Mid control moved the level by 0.063 against
a floor of 0.050, which is the difference between "Mid affects loudness" and
"the converters were breathing".

## These are measurements, not constants

A figure here describes **this block, through this rig, on this reference
signal**. Two things change it:

- **The reference signal.** Every measurement is a comparison against one fixed
  input, identified by hash in each file. A different one invalidates every
  number taken against the old one, the same way
  [changing a record invalidates the words derived from it](../../CONTRIBUTING.md#changing-a-source-is-never-one-file).
- **The path.** These came through a cable from the pedal's output back to its
  own input, so a digital-to-analogue and an analogue-to-digital conversion are
  in every reading. That is constant across a sweep and cancels when two
  settings are compared, which is what a sweep is for, and it is not zero.

So re-measuring on another rig will not reproduce these exactly. It should
reproduce their shape, and the shape is what gets used.

## Adding to them

```bash
just sweep 1 3                 # block at slot 1, its parameter 3
just sweep 1 3 --points 17 --out swept.json
```

A block's slot is its position in the chain plus one;
[docs/measuring.md](../../docs/measuring.md#addressing-a-control) has the layout
and the trap in it.
