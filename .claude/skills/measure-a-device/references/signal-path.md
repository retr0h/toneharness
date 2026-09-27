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

Only USB 5/6 reaches the signal chain. Sending to 1/2 goes straight to the
sockets on the back and never touches an amp model, so a test that plays there
and measures the return has measured a pair of converters.

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
