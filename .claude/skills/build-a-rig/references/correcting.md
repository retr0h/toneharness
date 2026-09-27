# Change a rig after hearing it

The loop that matters: play it, listen, say what is wrong in words, and let
that become an edit.

## A nudge is not a target

"Darker" names no point. It is a direction from wherever the chain currently is,
applied to one axis with the others held. Conversation is made almost entirely of
these, and treating one as a target aims at an absolute position nobody asked
for.

A target is the other kind of instruction, and there are three:

- **a record** is a full point, every axis specified
- **a player** is the median of their records, with the spread across them as a
  tolerance — an axis their records disagree about is one the answer need not be
  precise on
- **a genre** is partial: distinctive on some axes and ordinary on the rest, so
  only the distinctive ones are constrained

That partiality is why a genre is easier to satisfy than it sounds. A full point
in nine dimensions may be unreachable with the controls a device has; a target
pinning three axes and shrugging at six usually is not, and the freedom goes into
satisfying the three.

## When it cannot get there

A bass cabinet cannot produce what a target asks above 5 kHz, because the speaker
stops, and no combination of the controls in front of it changes that.

**That is a result, and it is reported as one** — which axes were met, which were
not, and by how far. Reporting success because a file was written is the failure
this project exists to avoid. A target that cannot be reached is worth keeping:
*the gear is wrong for the sound* is a real answer to somebody who owns that gear.

## Where the edit goes

Into the **rig**, not the request. A knob position is only meaningful against a
chain. So "the mids are honky" becomes a change to the RigSpec's settings, and
what the session taught about what was wanted goes back into the ToneSpec as a
word, with the round itself recorded in `corrections`. Those three fields are the
`write-a-spec` skill's, including what decides a value and how far a word moves
one.

## Move one control without writing anything

```bash
mise exec -- go run main.go device turn --block 1 --param 3 --value 0.4
mise exec -- go run main.go device current --json
```

`device turn` is a live parameter edit and writes no flash. `device current`
reads back what the device actually holds, which is the only way to know a
move landed: a live edit is not stored, so the next preset selection undoes it.

**Read the chain back before trusting an index.** A parameter has no name on
the wire, only a position in the model's own list, and `catalog show` prints
them sorted for a reader rather than in wire order. Counting down the printed
one mislabels every control and the numbers stay plausible while it does.

```bash
mise exec -- go run main.go measure names --model HD2_AmpUSDripmanNorm
```

That holds the catalog's order to the hardware. An index it could not test is
untested rather than wrong, which matters: a switch correctly refuses a number
on a dial and that says nothing about naming.

## Then export what worked

Read the device back as a rig and keep it. That is the artifact worth having,
because it is deterministic and somebody else can compile it.


