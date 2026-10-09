# Getting the signal in and out

**HX Edit must be quit.** It holds the USB interface exclusively and nothing here
can claim it while that is running.

## What is plugged in

Two connections, both at once, and neither is optional:

1. **USB** from the computer to the pedal. This carries the reference recording
   down and the measurement back up. It is also the editor channel.
2. **One 1/4" instrument lead** from the pedal's own output socket into its own
   input jack. This is the only way audio reaches the chain, because
   [USB 5/6 does not arrive](#usb-56-does-not-arrive).

So the pedal is wired to itself, and the computer talks to it over one cable.
Asking whether "the cable" is connected is ambiguous and has wasted somebody's
afternoon: say which of the two.

The lead is what makes the loop, and it is also what makes the loop able to
oscillate. That is the subject of
[the output block](#the-output-block-must-not-put-the-chain-back-on-the-loop).

An HX Stomp is a class-compliant USB audio interface, 8 in and 8 out, fixed at
48kHz unless Line 6's own driver is installed. macOS sees it with nothing
installed.

The loop asks for 48kHz and then checks what the audio backend actually gave
it, because those are different answers. miniaudio meets a device that cannot
run the requested rate by resampling rather than by refusing, so the reading
comes back as a plausible figure instead of an error.

It refuses that rather than converting. `resources/dry/bass-di.wav` is 48kHz and
so is every reading in `resources/sweeps/`, and a figure taken at another rate
cannot be filed beside those: the resampler's own artefacts land in the number.
A run on a machine with Line 6's driver installed, or through an interface
`--hardware` names that runs at something else, stops and says both rates:

```
the audio device is not running at the measuring rate: HX Stomp negotiated
44100Hz in and 48000Hz out, and every committed figure is at 48000Hz
```

The fix is a setting on the machine or a different `--hardware`, not anything
in this repository.

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

## Measure the empty loop first, every time

One reading, before any block, and it costs seconds:

```bash
mise exec -- go run main.go measure blocks --category pitch \
  --hardware "External Headphones,HX Stomp" --volume 38 --out /tmp/baseline.json
```

Both flags, because the rig is part of the reading: one name plays and records on
one device, which is the loop this page exists to warn about, and the volume is
what the committed figures were taken at.

It prints the empty loop before the first block. That line decides whether
anything after it means anything:

| The baseline says    | What to do                               |
| -------------------- | ---------------------------------------- |
| centroid under ~400Hz | the loop is good, carry on              |
| centroid ~12,000 Hz   | **stop.** The loop is not carrying audio |

**Read the centroid, not the level.** A bass through a cable measures low, because
a bass is low, and a dead loop measures its own hiss, which is nothing but high.
Those two are two orders of magnitude apart and no rig moves one into the other.

The level is not a gate, because it is a fact about which rig is in use. The
committed baseline in `resources/sweeps/hx-stomp/fingerprints.json` is **155.27Hz
at -58.55dB**, taken at volume 38 on the two-device rig this page recommends. A
reading of 168.8Hz at -61.22dB on 7 October 2026 matched it and was a good loop.

That -58dB would read as a failure against an older version of this table, which
said a good loop was ~-21dB and a bad one ~-46dB. Those were the one-device rig,
where the pedal played its own output into itself and the signal was forty
decibels hotter. The table went unchanged when the rig did, so it described a good
two-device loop as worse than a dead one-device loop, and it would have stopped a
run that was working.

That broken loop is also why the level cannot be the gate: it read **-46.65dB**,
twelve decibels **louder** than a good two-device loop. Hiss is not quiet, it is
just high.

The failure this prevents ran on 27 September 2026. A baseline of 11,990.0 Hz at
-46.65 dB went unread, eight sweeps were queued behind it, and the first one
finished before anybody noticed that every block was reading the same number.
They were all measuring the loop's own noise. The block being swept cannot tell
you this: `HD2_EQSimple3Band` read 11,990 Hz that day against the 93 Hz its own
committed sweep holds, on identical routing.

## Which leg is broken: a test that takes a minute

A wrong baseline says the loop is bad and nothing about which half. Measure a
block that makes sound without being given any:

```bash
go run main.go measure controls --model HD2_Synth4OSCGenerator \
  --points 2 --seconds 2 --takes 1 --out /tmp/synth.json
```

| What it reads                            | What that means                                  |
| ---------------------------------------- | ------------------------------------------------ |
| loud, tens of dB above the empty baseline | **the return works.** The send is what is broken |
| as quiet as the baseline                  | the return is broken, so the send tells you nothing |

It generates rather than processes, so its output reaches the computer whether or
not anything reaches the pedal's input. Run on 27 September 2026 it read -29dB
against a -64dB baseline, which said the chain reached USB and the reference
recording was not reaching the input jack. That is the send leg: the computer's
playback, the lead from the output socket, and the input jack. No routing value
in the preset changes any of it, and an hour went into the preset before this
test existed.

## Know what a working reading looks like

**The rig needs no routing changes, and this is the current answer.** A preset
built from the blank template carries `@input: 1` and `@output: 1`, and the loop
measures correctly through it. Nothing in the sweep path sets either, and nothing
should: an agent that "corrected" them to the Guitar jack and bare USB turned a
working rig into silence and spent an hour theorising about the protocol.

**Read this together with
[the output block](#the-output-block-must-not-put-the-chain-back-on-the-loop),
and do not stop here.** The sweeps in `resources/sweeps/` record `@input: 1` and
`@output: 1` in their `chain`, which is what the blank template carries, and they
hold correct figures. That does not make those two values right. A destination
that includes the quarter-inch jack puts the chain's output back on the loop, and
whether that oscillates depends on the gain around it, so the same preset reads
correctly one day and squeals the next. If a reading is pinned near 12kHz, that
is the first thing to rule out.

**Never guess a routing index; they differ per device family.** Ask the catalog,
which carries the lists:

```go
cat, _ := catalog.BuiltIn()
cat.SourceAt("Guitar")        // the chain's input
cat.DestinationAt("USB 1/2")  // the chain's output
```

An index written into a page is right for one pedal and silently wrong for the
next. On an HX Stomp today source 1 is `Multi (Guitar, Aux, Variax)` and
destination 1 is `Multi (1/4", XLR, Digital, USB 1/2)`. The second is the one
that closes the loop on itself.

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

## The output block must not put the chain back on the loop

`@output: 1` is not enough, whatever the label says. Entry 1 reads
`Multi (1/4", XLR, Digital, USB 1/2)` and an HX Stomp's Multi does not include
USB, so a preset left on it sends nothing up the cable and the USB return sits at
the converter's noise floor whatever the chain does.

Set `data.tone.dsp0.outputA.@output` to **10**, USB 1/2 by itself. The floor
moves from -123.4 to -118.6 dBFS, which is the amplifier idling rather than
nothing at all.

**Every measuring command does this for you now.** `quieter` in
`pkg/cli/headroom.go` reads the compiled preset back, looks `USB 1/2` up in the
device's own destination list rather than writing 10 down, and rebuilds the
preset before anything is played. So this section is why, not a step to carry
out: a plan compiled here cannot be left on Multi.

It went unfixed for a fortnight because the gain was treated as the fix instead.
Turning `dsp0.outputA.gain` down does drop the loop below unity for most blocks,
and it is still done, but an amplifier built to distort has enough gain of its
own to keep oscillating from -78dB. In one campaign 14 of the first 125 blocks
refused on that, every one an amp or preamp, including both SV Beasts this
project's own pipeline builds with. A guard that refuses the blocks you most
want is the shape of that mistake.

## The destination does not route, and the loop cannot be opened from here

**Settled on 2026-09-29 by measurement, and it contradicts the section above.**
One destination per preset, gain at 0, matt-freeman, the share of energy above
2kHz against a reference carrying 0.01%:

| `@output` | above 2kHz | level |
| ---------------------------------------- | ---------: | --------: |
| 0 `None` | 0.0% | -186.7dB |
| 1 `Multi (1/4", XLR, Digital, USB 1/2)` | 84.4% | -23.6dB |
| **10 `USB 1/2`** | **84.2%** | **-23.6dB** |
| **11 `USB 3/4`** | **84.4%** | **-23.6dB** |

The decisive pair is 10 against 11. These readings are taken on USB 1/2, so a
chain genuinely sent to USB 3/4 alone would read silence. It reads the same.

**On an HX Stomp this enum does not select where the chain goes.** Every non-zero
destination sends it everywhere, the quarter-inch socket included, and only
`None` silences it. So the measuring lead always carries the chain back to the
input, and **no value of `@output` opens the loop**. The headroom is the whole
fix, and `--headroom 0` is never a safe reading.

Two things this leaves unexplained, and both are worth knowing rather than
tidying away. The figure just above, that moving to `USB 1/2` took the floor from
-123.4 to -118.6dBFS, disagrees with this table and nobody has accounted for it:
different firmware, a different preset, or a confounded test. And the catalog's
own note that "an HX Stomp's Multi does not include USB whatever the label says"
cannot be right as stated, because Multi delivers the chain to USB at -23.6dB in
the table above.

The tool still sets the destination. Not because it helps, which is measured, but
because a firmware that started honouring the enum would want it right, and
because that earlier floor measurement has not been explained. It is documented
as doing nothing so nobody spends another afternoon believing in it.

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

## The computer's output device is part of the rig, and the tool sets it

The output level is pinned because an amplifier's distortion depends on how hard
it is driven, so a campaign at a different setting measures every amplifier as a
different amplifier. **The level belongs to a device**, and that is the half that
was missing: pin it while the platform's default output is some other device and
the number is about a device that is not in the signal path.

So a run that plays the reference out of the computer sets the device first and
pins the level on it second, and says both:

```
  the computer plays through External Headphones, which the level below belongs to
  the computer's output level moved to 38, for a consistent rig
```

CoreAudio rather than AppleScript, which owns the slider and has nothing to say
about which device the slider belongs to. `reamp.PlaysThrough` matches the name
the way `--hardware` does, so the same string means the same device in both
places, and it reads the device back afterwards rather than trusting the change.

**What it cost to not have this.** A Mac with its default on a pair of Bluetooth
headphones pinned those to 38 while the jack feeding the pedal sat wherever it
was left. The loop read 66dB of loss, the run reported a dead return, and an hour
went on the cabling, which was correct the whole time. Switching the output by
hand mid-run then did it again from the other side: the device changed, macOS
restored that device's own remembered volume of **0**, and the pin from the start
of the run was a number about a device nobody was listening to.

Both of those are the same mistake twice. A figure that moves with something
nobody is watching is the thing this whole page exists for.

On the one-device rig the pedal plays the reference and the computer's output is
not in the path, so nothing is set and nothing is said.

## Keep the audio when the figures do not add up

`tone tune --keep <dir>` writes what went in and what came back as WAVs, plus the
paths of the bass stems the target was measured from. Off by default, because a
pass is seconds of bass and a campaign is hundreds of them; on, it costs one more
reading at the end of a run.

It is for the fault the figures have no name for. The loop answers in ten numbers
and every one of them can sit inside tolerance while the sound is plainly wrong:
the first run that kept its audio is the section below, where the figures said the
loop was oscillating and the recording was a flat noise floor.

**Kept before a refusal as well as after a run**, because a refusal is when
somebody most wants to listen. A run that writes nothing on the way out is a run
that keeps the audio only when nothing was wrong with it.

What the pair is and is not good for. A record is mixed, mastered and limited and
a chain is one dry note through one amplifier, so the take and the stem do not
match and are not meant to, which is why the solver aims at a displacement rather
than at a record's own figures. The comparison is for an ear catching what no
figure is watching for.

## A dead loop reads as a squeal, and the fix is the opposite

**Check the level before believing any band share.** The squeal guard compares how
much of the returned energy sits above 2kHz against the reference's share. Noise
is broadband, so a return that is nothing but the converter's own floor reads
*brighter* than a bass reference and trips it. The run then says the loop is
oscillating and tells somebody to turn the amplifier's output down, over a lead
that is not carrying.

That happened on this rig, and the figures alone could not tell the two apart:

| | the dry reference in | what came back |
| --------------- | -------------------: | -------------: |
| level | -16.1dB | -81.9dB |
| peak sample | 0.562 | 0.0004 |
| above 2kHz | 0.00% | 7.1% |
| level over 5.8s | -21 to -13dB | -82dB, every window |

**Flat is the tell.** An oscillation builds or rings; a noise floor sits at one
number for the whole take. The 7.1% was the share of nothing.

`Returned` is the guard now, and it runs first, comparing what came back against
what went in rather than against the settled reading. A dead path settles on its
own noise, so every figure measured relative to it agrees with every other one.
The two populations are nowhere near each other: **a working loop here loses 16dB
and the dead one lost 66.**

```
nothing came back through the loop: -16.1dB went in and -81.9dB came back,
which is 66dB of loss and not a chain
      Nothing is returning: check the lead into the interface's input, and
      that the preset's output is the one it is plugged into
```

## A high-gain amplifier oscillates on its own, and the guard is right about it

Twenty of the 661 blocks refused #189's campaign, every one an amplifier built to
distort, plus two drives and a modulation. They refuse at every headroom the
backoff tries, down to -78dB. Measured on `HD2_AmpRevvGenRed` alone at -30dB:

| | bass reference in | silence in |
| --------------------- | ------------------: | ------------------: |
| level | -51.7dB | -52.8dB |
| centroid | 1,679Hz, wander 109 | 2,347Hz, wander 0.2 |
| below 250Hz | 27.8% | 1.1% |
| above 2kHz | 29.3% | 43.2% |

**With nothing going in it puts out the same level as a stable tone.** 1.1dB
down, at 2,347Hz, with the centroid wandering 0.2Hz across takes. That is what
settles it: hiss is broadband and wanders, and a tone locked to one frequency is
feedback. These amplifiers close the loop with their own gain and no trim reaches
them.

A theory that died here, recorded so nobody revives it. `suspect` confirms with a
30dB window on level alone, and enormous gain is noisy at idle, so it looked like
the guard might be reading hiss as oscillation. The spectrum says tone.

What does hold is the gate in front of that test: `now[High] > dry[High] + apart`,
against a bass reference carrying 0.01% above 2kHz. **Any block that saturates at
all opens it.** A guitar reference carries real treble, so on a guitar campaign
this fires far less often, and that is not the guitar amplifiers behaving better.

**The lead is the cause, and the models are fine.** Same block, same silence,
with the quarter-inch lead pulled out of the output socket:

| | lead in | lead out |
| ----------- | ------------------: | ------------------: |
| level | -52.8dB | -81.5dB |
| centroid | 2,347Hz, wander 0.2 | 688Hz, wander 1.3 |
| below 250Hz | 1.1% | 54.7% |
| above 2kHz | 43.2% | 8.5% |

29dB down, the tone gone, the spectrum flipped to low-heavy broadband: a high-gain
amplifier idling, which is what it should look like. The control that makes this a
reading rather than a measurement of nothing is that -81.5dB carries structure
where `@output: None` gave -186.7dB, so the chain reaches USB and is merely quiet.

So **these 23 blocks are measurable and the one-device rig is what cannot measure
them.** And there is a rig that can.

## The rig that opens the loop: play through the computer, record off USB

**Measured 2026-09-29 and it works.** A 3.5mm lead from the computer's own
headphone output into the pedal's quarter-inch input, the pedal's output
connected to nothing, and the reading taken off USB as before:

```bash
mise exec -- go run main.go measure chain   --hardware "External Headphones,HX Stomp"
```

Two names separated by a comma play through the first and record from the second.
One name is one device both ways, which is the old rig.

Why it opens the loop: that cable carries only what the computer plays. The
chain's output never reaches it, so there is no path back at all rather than one
held below unity. `HD2_AmpRevvGenRed`, which the one-device rig refused:

| | one device, -30dB | **two devices, -30dB** | **two devices, no trim** |
| --------------- | ----------------: | ---------------------: | -----------------------: |
| above 2kHz | 29.3% | **1.2%** | **1.2%** |
| below 250Hz | 27.8% | **78.9%** | **78.9%** |
| centroid | 1,679Hz | **257Hz** | **256Hz** |
| level | -51.7dB | -57.5dB | **-27.5dB** |
| level on silence | -52.8dB | **-108.5dB** | |
| signal over silence | **1.1dB** | **51dB** | **81dB** |

Three things that table settles. The oscillation is gone rather than reduced: 1.2%
above 2kHz against the reference's 0.01%, and silence reads the converter floor at
-108.5dB with a centroid wandering 67Hz, which is noise rather than the locked
0.2Hz tone the old rig produced. The spectrum is identical with and without the
headroom trim, so **the trim was only ever fighting the feedback** and costs 30dB
of signal over the floor for nothing here. And a bass through a high-gain
amplifier reads low-heavy, which is what it should have read all along.

**The price is two clocks.** They drift, which `reamp`'s package comment warns
about, and it is harmless for where energy sits and how loud it is. It is not
harmless for deconvolving an impulse response, so `pkg/sdk/cab` wants the one
device rig or its own alignment.

Level: a headphone output is hotter than a guitar input expects, so turn the
computer down rather than up. The pedal's Aux input is built for line level and is
entry 3 of the Sources enum if the guitar jack proves too hot.

## The computer's output level is in the signal path, and the tool pins it

On this rig only. The one-device rig plays out of the pedal, so the computer's
slider reaches nothing; here it is the thing playing the reference, and moving it
changes what every amplifier in a campaign measures as.

It is a tone control rather than a level control. An amplifier's distortion
depends on how hard it is driven, so a louder reference is not a louder reading
of the same tone. It is a different tone.

Every command that pushes audio takes `--volume`, 0 to 100, and sets the level
before measuring rather than trusting whatever it was:

```bash
mise exec -- go run main.go measure blocks   --hardware "External Headphones,HX Stomp"   --volume 38
```

Three things to know about the number.

**38 is where the committed library was taken.** It is the platform's own scale,
which is what a person sees on the slider, so it is neither decibels nor
comparable between machines. That is what the empty-loop baseline is for.

**The level that was actually reached is written into the file**, beside the
headroom, as `volume` on a `Library` and on a `Curves`. So a reading says which
level it was taken at rather than leaving it to be remembered.

**`-1` means the platform would not say.** Only macOS can be asked, through
`osascript`. Elsewhere the run carries on and records `-1`, which is the honest
answer: nothing pinned the level, so the figures are not reproducible without it.
The run warns and does not stop, because on the one-device rig the level is not in
the path and refusing there would be refusing for a reason that does not apply.

## The cable that closes it

One 1/4" lead from the Main out back into the pedal's own input jack.

```
Mac  --USB 1/2-->  Main out  --cable-->  Input jack
                                             |
                                           chain
                                             |
Mac  <--USB 1/2--  USB record  <-------------+
```

Set `@input: 2`, the Guitar jack, and `@output: 10`, USB 1/2 by itself, which
[the tool now does itself](#the-output-block-must-not-put-the-chain-back-on-the-loop).
**The chain must not reach the Main outs, or its own output races back round the
cable.** That is not theoretical: on 27 September 2026 a run on the blank
template's `@output: 1` oscillated, and the reading was a stable tone near
12kHz whose level followed whatever gain the block added. It looks like a
measurement and it is the rig listening to itself. It costs a D/A and
an A/D, and that noise is identical on every take, so it cancels the moment two
settings are compared, which is all this is for.
