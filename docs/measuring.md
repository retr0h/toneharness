# Measuring what a device does

Everything else here measures records. This measures the pedal.

A record is the far end of a signal chain: a bass, an amplifier, a microphone, a
desk and a master, all in one number. It says what the answer should sound like.
It cannot say what any one control did, because every one of them is already in
it and none can be moved.

To learn what a control does, a known signal goes in, and what comes out gets
measured. The difference is the device and nothing else.

## The loop

```
resources/dry/bass-di.wav
        │
        ▼  USB 5/6                    the pedal's Input block
   ┌─────────────┐
   │  HX Stomp   │
   └─────────────┘
        │  USB 1/2                    the processed output
        ▼
  tonestack measure                   the same nine figures records get
        │
        ▼
  compare against the record, adjust, repeat
```

Three of the four boxes work today. The pedal is the one that does not, and
[the state of it](#what-works-and-what-does-not) says how far it got.

## The pedal as an audio interface

An HX Stomp is a class-compliant USB audio interface, 8 in and 8 out, fixed at
48kHz unless Line 6's own driver is installed. macOS sees it without anything
being installed:

```
HX Stomp:
  Input Channels: 8
  Output Channels: 8
  Current SampleRate: 48000
  Transport: USB
```

### Which channel is which

From the HX Stomp manual, confirmed by a Line 6 forum thread where a user quotes
it back:

| Computer sends on | Arrives at                  |
| ----------------- | --------------------------- |
| USB 1/2           | Main L/R, the analogue outs |
| USB 3/4           | the stereo Send             |
| **USB 5/6**       | **the Input block**         |

So re-amping is USB 5/6 out of the computer and USB 1/2 back in. Only USB 5/6
reaches the signal chain; sending to 1/2 goes straight to the sockets on the
back and never touches an amp model.

### The Input block has to be told

USB 5/6 is live "only when the Input block is set to USB 5/6". That is a
per-preset setting, not a global one, so a preset built for measuring carries it
and an ordinary preset does not.

In a `.hlx` it is `data.tone.dsp0.inputA.@input`, an index into an enum the
preset does not name.

## The input and output enums

Nothing in this repository knew these values. They are in HX Edit's own
resources, at
`/Applications/Line6/HX Edit.app/Contents/Resources/HelixControls.json`, under
`input_type` and `output_type`. Reproduced because a machine somewhere will not
have HX Edit installed:

| `@input` | Input block source          |
| -------- | --------------------------- |
| 0        | None                        |
| 1        | Multi (Guitar, Aux, Variax) |
| 2        | Guitar                      |
| 3        | Aux                         |
| 4        | Variax                      |
| 5        | Variax Magnetics            |
| 6        | Mic                         |
| 7-10     | Return 1 to Return 4        |
| 11       | Return 1/2                  |
| 12       | Return 3/4                  |
| 13       | S/PDIF                      |
| 14       | USB 3/4                     |
| **15**   | **USB 5/6**                 |
| 16       | USB 7/8                     |

| `@output` | Output block destination              |
| --------- | ------------------------------------- |
| 0         | None                                  |
| 1         | Multi (1/4", XLR, Digital, USB 1/2)   |
| 2-4       | Path 2A, Path 2B, Path 2A+B           |
| 5         | 1/4"                                  |
| 6         | XLR                                   |
| 7, 8      | Send 1/2, Send 3/4                    |
| 9         | Digital (S/PDIF, AES/EBU, or L6 LINK) |
| 10        | USB 1/2                               |
| 11        | USB 3/4                               |
| 12        | USB 5/6                               |

The file carries `input_type_lt` and `output_type_lt` for a Helix LT and
`_native` for the plugin. HX Stomp uses the unsuffixed pair, which still lists
four Returns it has no sockets for; the indices are the Helix family's, not the
device's.

**`@output: 1` is enough.** Multi includes USB 1/2, so an ordinary preset
already sends its processed output back up the cable. Only the input needs
changing.

## Building a measuring preset

Take a preset off the device, change one number, put it back:

```bash
tonestack presets export --slot 42C --as hlx --out 42C-before.hlx
# set data.tone.dsp0.inputA.@input to 15
tonestack presets import --slot 42C --preset 42C-reamp.hlx
tonestack presets select --slot 42C
```

`import` keeps its own copy of whatever the slot held, under the state
directory, and says where:

```
kept ~/.local/state/tonestack/presets/42C-s0-20260919-035242.505232000.hlx
```

Read it back rather than trusting the write. `presets export` again, and check
the number is 15 and the name is what was sent.

### Which slot

Whichever slot a person names. `TONESTACK_SCRATCH_SLOT` is the convention the
device tests use, and nothing should overwrite a slot nobody offered.
[The protocol](protocol.md) says what a careless write costs: a mispaced one
stalls the endpoint and the pedal needs a power cycle. That is a bad thing to
discover while its owner is out.

## What works and what does not

**Working.** The loop runs. A dry file goes into the pedal, comes back
processed, and `tonestack measure` reads the result. Presets can be switched
unattended over MIDI. What is not working is changing one parameter, and the
reason is in
[The pedal ignores what it is written](#the-pedal-ignores-what-it-is-written).

**Not working.** One parameter cannot be changed. That is the only thing left,
and the reason is below.

### The output block does not mean what its label says

An HX Stomp's output enum is the Helix family's, and its entry 1 reads "Multi
(1/4", XLR, Digital, USB 1/2)". An HX Stomp has no XLR, and its Multi does
**not** include USB. A preset left on Multi sends nothing up the cable, and its
USB return sits at the converter's noise floor whatever the chain does.

Set `data.tone.dsp0.outputA.@output` to **10**, USB 1/2, explicitly. The
difference is visible immediately: the return's floor moved from -123.4dBFS to
-118.6dBFS, which is the amplifier idling rather than nothing at all.

This cost most of an evening. Every test before it looked like an input problem,
because the output was silent for a reason that had nothing to do with the
input.

### USB 5/6 still does not arrive

With the output fixed, USB 5/6 was swept across every input value the enum
holds, 0 to 16, and none of them carried audio.

What is established either side of it: the computer can send the pedal audio,
because a tone played on USB 1/2 comes out of its Main outs and was heard. The
pedal can send the computer audio, because the chain reaches USB 1/2. Only the
link between the computer's USB 5/6 and the Input block is missing.

### The cable that closes it

One 1/4" lead from the pedal's Main out back into its own input jack.

```
Mac  --USB 1/2-->  Main out  --cable-->  Input jack
                                             |
                                           chain
                                             |
Mac  <--USB 1/2--  USB record  <-------------+
```

Set the Input block to the Guitar jack, `@input: 2`, and the output to USB 1/2
only, `@output: 10`. The chain must not reach the Main outs, or its own output
races back round the cable.

It costs a digital-to-analogue and an analogue-to-digital conversion, which adds
noise. That noise is identical on every take, so it cancels the moment two
settings are compared, which is all this is for.

Measured through it, an amplifier and cabinet on a dry bass:

|           | dry             | through the pedal |
| --------- | --------------- | ----------------- |
| energy    | 91% low, 9% mid | 95% low, 5% mid   |
| centroid  | 145 Hz          | 117 Hz            |
| transient | 0.63            | 0.58              |
| decay     | 1.49 s          | 1.54 s            |
| dynamics  | 6.7 dB          | 7.1 dB            |

The cabinet pulled the centre of gravity down 28Hz and took four points of
energy out of the mids, which is what a speaker does to a signal. The numbers
behave like physics rather than like noise.

### The pedal ignores what it is written

**A preset written over USB does not become what the pedal plays.**

Slot 42C was given a copy of 01A, a preset measuring 97% of its energy between 1
and 6kHz. Reading 42C back over USB returns the copy. Playing 42C returns the
old preset, 0.3% over the same band. Selecting it by the vendor protocol and by
MIDI program change both do it.

So the device holds presets in memory and a USB write reaches its storage
without disturbing that. Everything written tonight is in the slots and none of
it is in the sound.

This is why sweeping a control by writing presets cannot work, and it is worth
knowing on its own: `presets import` does not do what somebody would reasonably
assume it does.

### What does work unattended

MIDI, over the same cable, with no writes at all:

```
PC   0  ->  01A   rms -17.5 dBFS   1-6kHz 97.054%
PC 125  ->  42C   rms -24.5 dBFS   1-6kHz  0.306%
PC   2  ->  01C   rms  -3.3 dBFS   1-6kHz  0.003%
```

Three presets, three measurements, nobody in the room. Program 0 is 01A and each
bank holds three, so a slot is `(bank - 1) * 3 + letter`.

What is missing is the message HX Edit sends when somebody drags a knob, which
changes a parameter in the running preset rather than in storage. That message
exists, because HX Edit does it. It is not implemented here.

## Why a synthesised signal will not do

A generated tone is repeatable, perfectly even across the range, and wrong for
most of this.

- **No dynamics.** Compression and drive are level-dependent. A signal at one
  level says nothing about either, however many notes it holds. The dry bass
  here carries 8.2dB between its typical and its loudest, and that spread is the
  measurement.
- **Three harmonics is not a bass.** Distortion works by creating
  intermodulation between partials. A three-partial tone and a real instrument
  come out of the same drive block sounding nothing alike.
- **No attack.** A synthesised note that starts at full amplitude has no pluck
  in it, and `transient` is one of the nine figures.
- **One decay for every note.** Real strings ring longer low than high, and the
  top dies before the fundamental.

Where a synthesised signal is better is the linear part. A logarithmic sine
sweep gives a complete frequency response in one pass, and separates harmonic
distortion out of the same recording. That is worth building and is not a
substitute for the bass.

[resources/dry/README.md](../resources/dry/README.md) says what the dry signal
is and where to get it.

## What this is for

A character word today moves a control by an amount somebody chose. `dark` takes
Treble down by a quarter of its range because a quarter was written in a file.
Nothing has ever checked that a quarter is what `dark` means, or that moving
Treble moves the figure `dark` was earned from.

Once this loop runs, a word can name a region in the nine figures instead of a
direction on a knob, and a preset either lands in it or does not.
[The design note](superpowers/specs/2026-09-18-nothing-here-has-ever-heard-anything-design.md)
argues that out.
