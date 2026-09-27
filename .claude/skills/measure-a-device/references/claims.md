# Which claim a reading supports

Four claims, and they are not the same:

1. the rig validates against the catalog
2. the device accepted it
3. the hardware loaded it
4. **the sound measures where it was aimed**

The fourth is why this exists, and it is the one the other three cannot stand in
for. **The first two have passed while the fourth failed.**

## Every chain this tool once wrote was empty

A chain was put in front of the device, read back verbatim, and rendered as
nothing. Same preset, two writers, measured through the loop:

```
written by HX Edit          centroid 4034.33 Hz
built by this tool          centroid  147.67 Hz
```

**147.67 Hz is a bass going down a cable through nothing.** Validation passed.
The write was accepted. It read back correct. The chain was not there.

The cause was the order of five keys inside a block body: the device writes the
model reference first and this tool wrote it last, so the device seeked to where
it keeps a model and found a class. Fixed on 27 September 2026.

Measuring the audio is what caught it, and nothing else could have. Reading walks
the MessagePack and finds a key wherever it sits, so the wrong order passed every
check this repository can make.

## And then the opposite failure, the same night

With the chain rendering, an amp read 50dB below the others. Not an empty chain:
a chain whose Master and channel volume arrived as zero, because a plan that
names a model and sets nothing sent every absent parameter as 0 rather than the
catalog's default.

Two wrong readings, two different causes, and telling them apart is a number:

| Reading                       | Means                                       |
| ----------------------------- | ------------------------------------------- |
| ~120-150 Hz at any level      | the chain did not render                    |
| kilohertz, 40-50dB down       | it rendered with a volume control at zero   |

Both are fixed. The reason to keep them is that the second looks like the first
to anybody who reads only the level, and a night went on three "fixes" aimed at
the wrong one.

## What that invalidates

Any measurement taken through a chain this tool built before 27 September 2026
describes an empty chain and a pair of converters. Those figures are kept because
the method was right and the conclusion drawn from them was wrong, which is worth
being able to see. They are not comparable to anything taken since.

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
