# Change a rig after hearing it

The loop that matters: play it, listen, say what is wrong in words, and let
that become an edit.

## Where the edit goes

Into the **rig**, not the request. A knob position is only meaningful against a
chain, and the chain is in the plan. So "the mids are honky" becomes a change
to the RigSpec's settings, and what the session taught about what was wanted
goes back into the ToneSpec as a word.

## Move one control without writing anything

```bash
mise exec -- go run main.go presets turn --block 1 --param 3 --value 0.4
mise exec -- go run main.go presets current --json
```

`presets turn` is a live parameter edit and writes no flash. `presets current`
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

More: [docs/workflows/correct-a-rig-you-have-heard.md](../../../../docs/workflows/correct-a-rig-you-have-heard.md).
