# Choose gear by measuring

The half no reasoning about names can do. A recording and a block are measured
by the same code into the same figures, so the ones that land nearest are the
ones worth trying.

## Measure a recording

```bash
mise exec -- go run main.go measure --file take.wav --json
mise exec -- go run main.go measure --dir ~/stems/htdemucs --json
```

WAV only. Convert on the way in with `ffmpeg -i take.mp3 take.wav`. Separate a
bass part out of a finished record with `mise exec -- just stems <in> <out>`.

Point `--corpus` at a tree holding one directory per player and the report says
which words each one's records earn. **A word is earned by sitting clear of the
other players**, so this is the only mode that produces any: one player has
nobody to be clear of. Point it at one instrument, because a bass centroid sits
an octave below a guitar's and a corpus holding both would earn every bassist
"dark" and mean nothing by it.

## Let a recording choose the amplifier

A ToneSpec with `like: { recording: take.wav }` and no named amp gets the
nearest of the device's 224 to what that file measures. No catalog entry and no
citation needed: a recording is evidence of itself.

## What the figures are

Ask, do not assume: `measured.Named()` is the list, and it has changed. The
figures a curve is reported in are not quite the figures a record is described
in, and the difference is documented where they are defined.

Two guards worth knowing, because both produced confident wrong answers before
they existed. **Silence is not a small reading**: two takes of nothing agree to
the last digit, so they clear the noise floor more convincingly than music
does. **A clipped reading is the converters', not the block's**: flat tops make
harmonics that were never in the signal, so it reads as a bright block and is
not one.

Detail: [docs/measuring.md](../../../../docs/measuring.md) and
[docs/algorithm.md](../../../../docs/algorithm.md).
