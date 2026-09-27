# What a chain carries besides blocks

A device lays a chain out on a fixed grid of 20 positions. The first holds the
input, two in the middle hold the split and the join, the last holds the output,
and the device decides where those sit. The other sixteen hold blocks or nothing.

So the position in that array is the block's number and **it is not its place in
the chain**: a preset holding four blocks can have them at 5, 6, 8 and 13. Nor is
it the number the wire answers to, which is one higher again, because the grid
keeps the input at 0. `device turn --block` takes the position and does that
arithmetic itself.

Footswitch assignments address blocks by that number, so renumbering them to 0
through 3 would break the only link between a switch and the block it works on.

Two things travel with the chain that nobody chooses, and leaving either out
produces a preset that looks right and behaves wrongly.

## The snapshots are part of the chain

Each of the three snapshots carries an array of 20 in step with the grid, holding
whether each position is switched on. **A chain written without them recalls the
wrong blocks the moment anybody presses a snapshot.**

Writing the chain's own state across all three is right for a chain nobody has
snapshots for and wrong for a preset that arrived with three different sounds,
which would come back holding one under the names the blank shipped with. What
gets written instead is what the source recalls: the name, the tempo, the colour,
and which positions each snapshot switches on.

The four routing positions are left alone, because a snapshot records those too
and switching one off recalls the chain with its split bypassed.

## The routing is part of the chain too

A chain is written into the other sixteen positions, so a preset built from a
file keeps the routing of the slot it was built into: an output gain somebody set
arrives at zero, and a split somebody balanced arrives as whatever the blank
carried. What the source says is written instead, each entry found by the kind it
declares rather than by where one blank happens to keep it.

**The values are capped at the length the device itself wrote**, because a device
sends fewer parameters than a model names. An input names seven and sends three,
an output names three and sends two. Writing the model's full list would hand the
device a longer array than it produced.

## One thing that looks wrong and is not

A snapshot's tempo is stored and read back correctly while HX Edit shows the
preset's tempo for all three. That is not a lost value: the device has a global
Tempo Select setting deciding whether tempo follows a snapshot at all. Check that
before going looking for a bug.

## Reading the chain back before trusting an index

```bash
mise exec -- go run main.go presets show --slot 42C --json
mise exec -- go run main.go catalog show --model HD2_AmpSVBeastNrm --json
```

`presets show` lists a chain in order and `catalog show` lists a model's
parameters in theirs, which is what `device turn` addresses. A parameter has no
name on the wire, only a position in the model's own list, so counting down a
table printed for a reader mislabels every control, and the numbers stay
plausible while it does.

An amplifier and its cabinet are one block to a device and two entries in a
preset, which is why `device turn --model 1` reaches a cabinet fused into an
amplifier's slot.
