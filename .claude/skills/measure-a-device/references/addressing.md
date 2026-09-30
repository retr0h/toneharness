# Addressing a control

A live edit names a block by a number and a parameter by a number, and neither
is the one a reader would guess.

```bash
mise exec -- go run main.go device turn --block 1 --param 3 --value 0.4
mise exec -- go run main.go device current --json
```

`--value` is a dial, `--choice` is one of a list such as a cabinet's microphone,
and `--switch` is on or off such as an amplifier's Bright. A device does not
coerce, so the wrong one is refused.

## The parameter is its position in the device's own list

Not the order a `.hlx` writes its keys, and not anything alphabetical. One
amplifier's wire order begins Norm Drive, Bass, Mid, Treble where its preset JSON
is sorted and begins Bass, Bias, BiasX, Bright.

`catalog show` prints them sorted for a reader, so the two disagree on nearly
every model. **Counting down the printed one mislabels every curve, and the
numbers stay entirely plausible while it does.** Hold the catalog to the
hardware instead:

```bash
mise exec -- go run main.go measure names --model HD2_AmpUSDripmanNorm --json
```

It moves each index, reads back which named parameter changed, and says so. Two
values are tried per index, because a control already resting on the first would
show no change and be reported as an index that reaches nothing. An index it
could not test is untested rather than wrong.

## The block is its slot, not its place in the chain

An HX Stomp's layout is fixed: 0 the input block, 1-8 path A where a chain's
blocks go, 9 the output block, 10 the split, 11-18 path B, 19 the join.

So **a block's slot is its position plus one** on the first path: an amplifier at
`@position: 0` is slot 1. On the second path it is the position plus the split's
slot plus one. Slots 20 and above do not exist and always refuse.

Measured, on a preset whose blocks sat at positions 0, 1, 3 and 4: addressing 5
moved the block at position 4, addressing 4 moved the one at position 3, and
addressing 0 was refused with `-3` because the input is not a block. **`device
turn --block` takes the position, not the slot**, and adds the one itself, so the
number `presets show` prints is the number to type. The SDK's `Address.Block` is
still the slot, which is what the wire wants and what a sweep passes.

**The four structural slots answer even on an empty preset**, and that misleads.
A preset rendering as nothing still answers on 0, 9, 10 and 19 and refuses
everything else, which reads exactly like "chain blocks cannot be addressed" and
is really "there are no chain blocks". Read the chain back with
`presets show` or `device current` before concluding anything from a refusal.

## What a refusal means

`-3` is a bad block or parameter reference, and it has five causes:

- a slot the loaded preset has nothing in
- a slot past 19 on this device
- the wrong wire type: a switch takes a bool and refuses the same number as a
  float or an int, and a cabinet's `Mic` is an enum and takes an int
- a parameter past the model's own list, which needs the addressing mode false
  with the index zero
- a split's `bypass`, which takes a different opcode

**Nothing refuses because of what kind of block it is.** Amplifiers and cabinets
take a live edit like anything else, against a noise floor of 6.9 Hz on the
centroid, which is the whole point of
[measuring the noise floor first](sweeping.md#silence-is-repeatable-so-the-noise-floor-cannot-catch-it).
