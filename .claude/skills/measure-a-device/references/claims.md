# Which claim a reading supports

Four claims, and they are not the same:

1. the rig validates against the catalog
2. the device accepted it
3. the hardware loaded it
4. **the sound measures where it was aimed**

The fourth is why this exists, and it is the one the other three cannot stand in
for. **The first two have passed while the fourth failed.**

## Every preset this tool once wrote was empty

A preset was written into a slot, read back verbatim, and rendered by the device
as an empty chain. Same file, two writers, measured through the loop:

```
written by HX Edit          centroid 4034.33 Hz
a byte-for-byte import      centroid  147.67 Hz
```

**147.67 Hz is a bass going down a cable through nothing.** Validation passed.
The write was accepted. The slot read back correct. The chain was not there.

Measuring the audio is what caught it, and nothing else could have. Reading
ignored an offset table the device's renderer uses, so a document whose table did
not match its bytes passed every check this repository can make.

## What that invalidates

Any measurement taken through a preset this tool wrote before the encoder was
fixed describes an empty chain and a pair of converters. Those figures are kept
because the method was right and the conclusion drawn from them was wrong, which
is worth being able to see. They are not comparable to anything taken since.

The same trap is why [the four structural slots answering on an empty
preset](addressing.md#the-block-is-its-slot-not-its-place-in-the-chain) cost an
evening: a preset that renders as nothing still answers on 0, 9, 10 and 19.

## So establish the path before believing a figure

Before reporting any reading, know and say:

- what the chain actually was, read back with `device current` rather than
  assumed from what was sent
- that something arrived, against the empty-loop baseline rather than against
  zero
- which positions came back `silent` or refused, and with which of
  [the five causes](addressing.md#what-a-refusal-means)

```bash
mise exec -- go run main.go device current --json
```

**Reporting success because a file was written is the failure this whole project
exists to avoid.** If the audio was not measured, the fourth claim is not
available, and saying so is the answer rather than a caveat on a different one.
