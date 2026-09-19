# dry

The dry signals pushed through a device to find out what it does.

Everything else here measures records, which are the far end of a signal chain:
a bass, an amplifier, a microphone, a desk and a master. A record says what the
answer should sound like. It cannot say what any one control did, because every
one of them is already in it and none can be moved.

These are the near end. One bass, straight to the converter, no amplifier and no
effects. Play one through a device, measure what comes back, and the difference
is the device and nothing else.

A record cannot be used for this. Running a finished record through an amplifier
models an amplifier into an amplifier, and the answer describes neither.

## What is here

| File                |                                            |
| ------------------- | ------------------------------------------ |
| `bass-di.wav`       | 3:35, the whole take                       |
| `bass-di-short.wav` | 0:34, a different session of the same song |

Both 24-bit, mono, 44.1kHz.

As `tonestack measure` reads the full take:

```
energy      93% low · 7% mid · 0% high
centroid    138 Hz
transient   0.75
decay       1.73 s
dynamics    8.2 dB
harmonics   30%
```

The 8.2dB of dynamics is the part that matters and the part a synthesised file
does not have. Compression and drive are level-dependent, so a signal at one
level says nothing about either, however many notes it contains.

## Where they came from

The Cambridge-MT "Mixing Secrets" Multitrack Download Library, MTK001, real
studio multitracks given away for teaching.

- `https://mtkdata.cambridgemusictechnology.co.uk/MTK001/JohnMcKay_DaisyDaisy_Full.zip`,
  entry `JohnMcKay_DaisyDaisy_Full/06_BassDI.wav`, saved here as `bass-di.wav`
- `https://mtkdata.cambridgemusictechnology.co.uk/MTK001/JohnMcKay_DaisyDaisy.zip`,
  entry `JohnMcKay_DaisyDaisy/04_BassDI.wav`, saved here as `bass-di-short.wav`

```
8849a69ccf25a26d0ddd33956a2eb903ce452714298da039e542436f755d99d1  bass-di-short.wav
859209f091f1e92db2ffdfccf9b586e348005aafcf7c7ed8eee08f235087af4b  bass-di.wav
```

`cambridge-mt.com` itself sits behind a challenge that refuses an automated
reader. The data host above does not, and serves range requests, so a single
file can be pulled out of a 208MB zip without fetching the zip.

## Redistribution

**No.** The library's own terms, from the readme in the zip: "provided for
educational purposes only, and the material contained in them should not be used
for any commercial purpose without the express permission of the copyright
holders."

So the audio is ignored by git, as the records under `music/` are, and this file
is what travels instead. Anybody can fetch the same bytes from the urls above
and check them against the hashes.

## Why this one

The filename says DI and a filename is not evidence. The same session ships
`07_BassAmp.wav`, the miked cabinet of the same performance, and the two were
measured against each other:

- Between 2-4kHz and 4-6kHz the amp track drops 38dB. This one drops 14dB and
  goes on decaying smoothly to 22kHz. A cabinet is a brick wall at the top and a
  direct signal has no such corner.
- In the 20-60Hz band this one sits 23dB higher. An amplifier and cabinet cut
  the bottom; a direct signal keeps it.

Peak is -1.5dBFS with a crest factor near 16dB, so nothing was limited or
clipped on the way in.

What cannot be ruled out is the direct box itself, which colours the signal by
existing, and a gentle EQ printed at tracking. Neither is an amplifier.
