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
reason is in [The pedal ignores what it is written](#the-pedal-ignores-what-it-is-written).

**Not working.** Nothing sent to the pedal comes back. Playing a tone down each
of the eight output channels in turn and recording all eight inputs gives the
same reading whether a tone is playing or not:

```
out\in        1       2       3       4       5       6       7       8
  1     -125.4  -125.4  -999.0  -999.0  -123.3  -123.3  -999.0  -999.0
  ...
  -     -125.4  -125.4  -999.0  -999.0  -123.3  -123.3  -999.0  -999.0   (nothing played)
```

Two things in that table are worth keeping. Inputs 1/2 and 5/6 sit at about
-123dBFS, which is a live stream carrying dither, while 3/4 and 7/8 read as
digital zero and are not connected at all. So the pedal is sending, and what it
sends is silence. And the row with nothing playing is identical to every other
row, so nothing the computer sends is arriving.

Ruled out along the way:

- **macOS privacy blocking the input.** The built-in microphone records the room
  at -62dBFS from the same process, so input is permitted.
- **The preset.** Read back from the device after writing: `@input` is 15 and
  `@output` is 1.
- **Sample rate.** Both ends at 48kHz, the only rate class-compliant mode
  offers.
- **A wrong channel.** Every output channel was tried, not just 5/6.

**The open suspect** is that no Line 6 driver is installed, so the pedal is in
class-compliant mode. `/Library/Extensions`, the system extension list and
`kextstat` have nothing from Line 6. Line 6 support's own words in that forum
thread, that the pedal "is taking over the sound card like an audio interface",
describe monitoring rather than re-amping, and nobody in the thread was trying
to re-amp. Whether class-compliant mode carries all eight output channels or
only the first pair is not established here and is the next thing to find out.

Installing a driver needs a person and probably a restart, so it stopped here.

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
