# Getting the signal in and out

**HX Edit must be quit.** It holds the USB interface exclusively and nothing here
can claim it while that is running.

An HX Stomp is a class-compliant USB audio interface, 8 in and 8 out, fixed at
48kHz unless Line 6's own driver is installed. macOS sees it with nothing
installed.

## Which channel is which

| The computer sends on | Arrives at                  |
| --------------------- | --------------------------- |
| USB 1/2               | Main L/R, the analogue outs |
| USB 3/4               | the stereo Send             |
| **USB 5/6**           | **the Input block**         |

USB 5/6 is the route that *should* reach the signal chain. **It does not on this
hardware**, and the rig here is a cable instead: read
[the cable that closes it](#the-cable-that-closes-it) and
[know what a working reading looks like](#know-what-a-working-reading-looks-like)
before writing any of this into code. This table is what the enum claims.

The claim still explains the shape of the cable rig: sending to USB 1/2 goes to
the sockets on the back rather than to an amp model, which is exactly why a lead
is needed from those sockets back into the input jack.

## Know what a working reading looks like

**The rig needs no routing changes.** A preset built from the blank template
carries `@input: 1` and `@output: 1`, and the loop measures correctly through it.
Nothing in the sweep path sets either, and nothing should: an agent that
"corrected" them to the Guitar jack and bare USB turned a working rig into
silence and spent an hour theorising about the protocol.

Compare against the readings in `resources/sweeps/`, which are the only figures
here anybody has confirmed. One block alone, at the first point of its first
control:

| Block           | centroid  | level  |
| --------------- | --------- | ------ |
| Tuck n Go       | 2,988 Hz  | -13.75 |
| Cali 400        | 3,918 Hz  | -13.95 |
| G Cougar 800    | 5,245 Hz  | -6.66  |
| SV Beast Brt    | 8,472 Hz  | -15.39 |
| US Dripman Norm | 12,823 Hz | -25.49 |

**A working amplifier reads high and loud**: kilohertz of centroid at better than
-26 dBFS. That is not intuitive for a bass signal, and it is the thing to hold on
to, because the intuitive reading is the broken one.

| What comes back                 | What it is                                              |
| ------------------------------- | ------------------------------------------------------- |
| kilohertz, better than -26 dBFS | the chain, working                                      |
| ~120-150 Hz at any level        | **an empty chain**: a bass down a cable through nothing  |
| ~2 Hz at ~-105 dBFS             | silence, nothing arriving at all                        |
| kilohertz but 40 dB too quiet   | the chain is there and the signal is not                |

The second row has cost more than the rest together. It looks exactly like what a
bass ought to measure, so it reads as success. It was
[a block body's keys in the wrong order](../../../../pkg/sdk/internal/wire/README.md#a-block-bodys-keys-go-in-the-devices-order):
the device seeks to where it keeps a model, found something else, and rendered
nothing, while the preset read back byte for byte. Fixed, and the test that holds
it compares bytes rather than reading the chain back.

An agent read 123 Hz as a working amplifier, wrote it into this page as the
known-good figure, and built three rounds of fixes on top of it. Check a new
figure against the table above before believing it.

The fourth row is the one to check next, because it is the near miss. A chain
whose volume control arrived at zero reads in the kilohertz and 40 to 50dB down,
which is a chain that rendered, not one that did not. Read the chain back with
`device current` and look at what the parameters actually say: the level is a
setting, and the centroid says the blocks are there.

## When opening the audio device fails

`miniaudio: Invalid argument` on opening the pedal is **a stalled USB endpoint,
not a channel count**. Power cycle the pedal, exactly as
[a write that stops answering](device-care.md) says, and try again. It was read
as a channel-count problem once and cost an evening: the code asked for two
channels, the device presents eight, and changing that to match broke a loop
that had been working for a week.

Two channels is correct. The signal leaves on the first pair, which is where the
Main out listens, and comes back on the first pair as a recording.

## Before deciding the protocol is wrong

Check when the thing last worked. `git log` on the sweep files under
`resources/sweeps/` dates every reading that has ever been taken, so a path that
produced one yesterday is a path whose setup is fine and whose caller changed.
That check is ten seconds and it is the one that was skipped.

**USB 5/6 is live only when the Input block is set to it, and that is per
preset rather than global.** A preset built for measuring carries it and an
ordinary preset does not. In a `.hlx` it is `data.tone.dsp0.inputA.@input`, an
index into [an enum the preset does not name](enums.md).

## The output block does not mean what its label says

`@output: 1` is not enough, whatever the label says. Entry 1 reads
`Multi (1/4", XLR, Digital, USB 1/2)` and an HX Stomp's Multi does not include
USB, so a preset left on it sends nothing up the cable and the USB return sits at
the converter's noise floor whatever the chain does.

Set `data.tone.dsp0.outputA.@output` to **10**, USB 1/2 by itself. The floor
moves from -123.4 to -118.6 dBFS, which is the amplifier idling rather than
nothing at all.

This is the trap worth recognising: **every test before it looked like an input
problem, because the output was silent for a reason that had nothing to do with
the input.** Establish the return before blaming the send.

## USB 5/6 does not arrive

**The finding that decides how the loop is built.** Read it before the table
above, not after: an agent that took the table as the answer wrote a sweep
sending on USB 5/6, measured the amplifier's own hiss, and reported figures for
a control it had never reached.

Swept across every input value the enum holds, 0 to 16, none of them carried
audio in. What is established either side of it: the computer can send the pedal
audio, because a tone on USB 1/2 comes out of the Main outs and was heard, and
the pedal can send the computer audio, because the chain reaches USB 1/2. Only
that one link is missing.

## The cable that closes it

One 1/4" lead from the Main out back into the pedal's own input jack.

```
Mac  --USB 1/2-->  Main out  --cable-->  Input jack
                                             |
                                           chain
                                             |
Mac  <--USB 1/2--  USB record  <-------------+
```

Set `@input: 2`, the Guitar jack, and `@output: 10`. **The chain must not reach
the Main outs, or its own output races back round the cable.** It costs a D/A and
an A/D, and that noise is identical on every take, so it cancels the moment two
settings are compared, which is all this is for.
