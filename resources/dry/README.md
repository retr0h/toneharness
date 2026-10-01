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

| Path                |                                            |
| ------------------- | ------------------------------------------ |
| `bass-di.wav`       | 3:35, the whole take                       |
| `bass-di-short.wav` | 0:34, a different session of the same song |
| `fingerstyle/`      | 469 single notes played with fingers       |
| `picked/`           | 468 single notes played with a plectrum    |

All 24-bit, mono, 44.1kHz.

The two takes and the two directories answer different questions. A take is one
performance and carries the dynamics a measurement of compression or drive
needs. The directories are the same notes played two ways, which is the only
thing that can say how much of a sound is the right hand rather than the rig.

As `toneharness measure` reads the full take:

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

## Where the two takes came from

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

## Where the two directories came from

IDMT-SMT-Bass, from Fraunhofer IDMT: 5,328 single notes recorded to teach
machines to read a bass line, across three four-string basses at three pickup
settings each, covering E1 to G3.

- `https://zenodo.org/records/7188892/files/IDMT-SMT-BASS.zip`, 1.5GB, whose
  `PS/FS/` and `PS/PK/` became `fingerstyle/` and `picked/` here. The rest of
  the archive varies expression rather than the hand, and is fingerstyle
  throughout, so it says nothing about a plectrum and was not kept.

```
47a19041ad49182dff2b3d85927051bab93fc5c31bc820c623840c9dfc4fce74  IDMT-SMT-BASS.zip
1df98833dccc8fcfbf47292b0c5774421e636a6d173517d3938aca2b12e2ae95  fingerstyle/
1f2a3232a5cb51d08489fe381d89f5e6d3bb9e6c9077a4b1e02367b2867cda8b  picked/
```

A directory's hash is over its filenames and their bytes, sorted, since there is
no single file to name.

Cite it as Jakob Abeßer, Fraunhofer IDMT, and read the licence below before
using it for anything.

**468 of them are pairs.** A name says what it is: `BS_2_EQ_2_PK_NO_2_8.wav` is
bass 2, pickup setting 2, picked, normal expression, and then the note. Every
picked note has the same note on the same bass at the same pickup in
`fingerstyle/`, with one fingerstyle note left over. So the difference between a
plectrum and fingers can be taken per note and averaged, rather than taken
between two averages of a note range, and the spread that comes out is the
spread of the effect instead of the spread of the pitches.

## Redistribution

**None of it is ours to redistribute, and the two licences differ.**

Cambridge-MT, from the readme in the zip: "provided for educational purposes
only, and the material contained in them should not be used for any commercial
purpose without the express permission of the copyright holders."

IDMT-SMT-Bass is CC BY-NC-ND 4.0: attribution, no commercial use, no
derivatives. Stricter, and the no-derivatives clause is the one to read before
anything measured from it ships.

`fingerstyle/` and `picked/` are git-ignored, as the records under `music/` are,
and these hashes are what travels instead. Anybody can fetch the same bytes from
the url above and check them.

**`bass-di.wav` and `bass-di-short.wav` are not git-ignored.** `.gitignore`
excludes `resources/dry/**` and then un-ignores `resources/dry/*.wav`, so both
are committed through git LFS and go out with every clone. That is
redistribution of the material the paragraph above says not to redistribute, and
this file said the opposite until 2026-10-01. Said plainly here rather than left
as a contradiction between a sentence and a pointer file: deciding what to do
about it is a licensing question, not a tidying one.

Note the asymmetry in the rule. `!resources/dry/*.wav` reaches only the top
level, because git does not descend into an excluded directory, so a `.wav` in a
subdirectory stays ignored without anybody writing a rule for it. That is why
the 937 notes added beside them needed no `.gitignore` change, and it is luck
rather than design.

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
