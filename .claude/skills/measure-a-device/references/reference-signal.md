# The reference signal

Every measurement here is a comparison, and a comparison needs one thing held
still. That thing is `resources/dry/bass-di.wav`, passed as `--dry`.

The same bass, the same notes, the same playing, every time. Whatever differs in
the answer is the device, because nothing else was allowed to move.

## Why it is a recording of somebody playing

A generated signal is repeatable too. It is not a bass, and what an amplifier
does to three sine waves is not what it does to an instrument. Four reasons the
synthesised version will not do:

- **No dynamics.** Compression and drive are level-dependent, so a signal at one
  level says nothing about either however many notes it holds. The dry bass
  carries 8.2 dB between its typical and its loudest, and that spread is the
  measurement.
- **Three harmonics is not a bass.** Distortion works by creating
  intermodulation between partials, so a three-partial tone and a real
  instrument come out of the same drive block sounding nothing alike.
- **No attack.** A note that starts at full amplitude has no pluck in it, and
  `transient` is one of the figures.
- **One decay for every note.** Real strings ring longer low than high, and the
  top dies before the fundamental.

Where a synthesised signal is better is the linear part. A logarithmic sine
sweep gives a complete frequency response in one pass and separates harmonic
distortion out of the same recording. Worth building, and not a substitute for
the bass.

## The hash is not bookkeeping

`resources/dry/README.md` carries the url, the zip entry and the hash. The hash
is what makes a figure measured next month comparable to one measured today.

**A different reference file silently invalidates every number ever taken
against the old one.** Nothing in the figures marks the moment it happened, and
two generations of readings sit side by side looking comparable. So pin the
reference, hash it, and if it ever has to change, re-measure everything that was
measured against it rather than keeping both.

Same discipline as changing a record in the corpus, and the same reason.

## What a reading is worth against it

Comparability is the whole point, so a reading taken with a different `--dry`,
a different `--seconds`, or through a different cable is a different experiment
and belongs in a different file. Say which when reporting one.

```bash
mise exec -- go run main.go measure controls \
  --model HD2_AmpUSDripmanNorm \
  --dry resources/dry/bass-di.wav \
  --points 9 --json
```
