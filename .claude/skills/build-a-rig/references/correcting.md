# Change a rig after hearing it

The loop that matters: play it, listen, say what is wrong in words, and let
that become an edit.

## A nudge is not a target

"Darker" names no point. It is a direction from wherever the chain currently is,
applied to one axis with the others held. Conversation is made almost entirely of
these, and treating one as a target aims at an absolute position nobody asked
for.

A target is the other kind of instruction, and there are three:

- **a record** is a full point, every axis specified
- **a player** is the median of their records, with the spread across them as a
  tolerance — an axis their records disagree about is one the answer need not be
  precise on
- **a genre** is partial: distinctive on some axes and ordinary on the rest, so
  only the distinctive ones are constrained

That partiality is why a genre is easier to satisfy than it sounds. A full point
in nine dimensions may be unreachable with the controls a device has; a target
pinning three axes and shrugging at six usually is not, and the freedom goes into
satisfying the three.

## Ask before you run

```bash
mise exec -- go run main.go tone reach --id matt-freeman --genre punk
```

One pass of readings and nothing applied: a reading to settle, one for the
baseline, one per control. A minute and a half on a chain of sixteen, against
five minutes for the tuning run it decides whether to spend, and the chain is
left where the compiler put it.

```
  matt-freeman against punk, 16 dials through HX Stomp
  the loop wanders 0.0013 of a band and 3.1Hz, reading 16 controls

    AXIS            OUT BY     ALONE  TOGETHER
    -> high          390.6   reached      45.2  the dials reach it
    -> centroid       52.9   reached       4.2  the dials reach it
    ok low             9.4   reached       0.9  the dials reach it
    ok mid             0.3       met       0.3  already there
```

**ALONE is what that axis's own dials could do for it, ignoring the others.**
It flatters them: it assumes each slope holds across a whole range it was read
locally and that every dial pulls the same way. So it refuses well and promises
badly, and an axis marked OUT OF REACH is out of reach while one inside is only
worth attempting.

**TOGETHER is what is left once every axis is solved at once**, which is the
question somebody actually means. A dial can put the centroid where a target
wants it and can put the low band where the target wants it, and those are two
different positions of one dial. Every axis above is reachable alone and none
of them together.

It is worth trusting because it predicts what the loop then does. Reach said
high would still be 45.2 tolerances out; the tuning run that followed reported
46.4.

### It reads the chain, and this is why

`resources/sweeps/` cannot answer this. Every sweep there was taken with its
block alone, which the file records in `isolated` and `measured/curves.go`
calls the most important field it carries. Measured on an HX Stomp,
matt-freeman's chain, centroid in hertz per full turn:

| control | swept alone | in the chain |
| ------- | ----------: | -----------: |
| Treble  |      12,763 |        3,250 |
| ChVol   |      13,133 |        3,051 |
| Drive   |      12,147 |        1,921 |
| Bass    |      -7,871 |         -831 |
| Mid     |      -6,724 |     **+3,593** |
| Sag     |      +1,393 |       **-239** |

Four to ten times too large everywhere, and four of eleven controls point the
wrong way.

The first version of this command answered from those readings and cost a
second. It said every shipped rig reached every measured genre, which is what a
false premise buys.

So the committed sweeps say **which controls are worth putting in a matrix**,
and nothing more. `measure slopes --id <rig>` prints live beside committed for
any chain, which is how that was found.

### The loop oscillates, and the fix is where the chain is sent

The cable rig sends the pedal's output back into its own input. A preset
compiled from a plan inherits the blank template's routing, which sends the
chain to `Multi (1/4", XLR, Digital, USB 1/2)`, and Multi drives the
quarter-inch jack that lead comes from. So the chain arrives at its own input.
With enough gain around that loop it squeals, and an amplifier is where the gain
is.

**The destination is the fix.** Every measuring command now sends the chain to
`USB 1/2` by itself, which reaches the computer without reaching the socket, so
the lead carries only the reference recording and the loop is open. It is looked
up in the device's own destination list rather than written down as a number,
because the lists are per device family. Nothing has to be asked for: it happens
whatever `--headroom` says.

**`--headroom` is the second half, and it was treated as the whole of it for a
fortnight.** It turns `dsp0.outputA.gain` down before anything is measured and
defaults to -30dB. That gain sits after every block, so it lowers the level
without touching the tone, where turning the amplifier down would work and would
be wrong: ChVol and Master are the sound the rig describes, and the solver would
then solve for a different one. It is kept because it bounds an amplifier's own
hiss and anything left going round the rig, and because every reading committed
here was taken with it.

Why the gain alone was not enough: a block built to distort has enough gain to
close the loop from -78dB, four steps past the default. In one campaign 14 of
the first 125 blocks refused on exactly that, every one an amp or a preamp,
**including both SV Beasts this project's own pipeline builds with**. A guard
that refuses the blocks you most want measured is how a symptom fix announces
itself.

Measured on matt-freeman, a compressor into an Ampeg SVT into an 8x10, sweeping
the headroom on the Multi destination and changing nothing else. These are
readings of the loop as much as of the chain, which is the point of them:

| headroom | below 250Hz | above 2kHz | centroid | level     |
| -------- | ----------: | ---------: | -------: | --------: |
| 0        |       15.4% |      84.3% |  2,197Hz | -23.6dB   |
| -15      |       25.3% |      73.9% |  1,941Hz | -39.2dB   |
| -20      |       58.7% |      39.1% |  1,085Hz | -47.9dB   |
| **-25**  |   **96.1%** |   **0.1%** | **126Hz** | -54.9dB  |
| -30      |       96.0% |       0.1% |    126Hz | -60.0dB   |

It stops running away between -20 and -25, and the default is one step past
that. It costs level, which is the trade: a squeal is loud.

What that is worth, same rig against punk:

```
             squealing     with headroom
  high      0.8433 (390 out)   0.00064 (0.3, met)
  centroid   2,197 (52.9 out)      126 (0.5, met)
  low       0.1543 (9.4 out)      0.96 (0.1, met)
```

and the loop converges to 2.4 tolerances out where it stopped at 46.4 before.

**The guard stays, because headroom is a setting and settings get changed.** A
cabinet is a loudspeaker and a loudspeaker is a low pass: a chain ending in one
can remove energy above 2kHz and cannot add it. `tone tune` and `tone reach`
refuse when it does, `measure chain` says so and prints the numbers anyway.

Neither obvious test works in its place. Level does not separate them, at
-23.6dB squealing against -26.4dB clean. Nor does pushing silence through,
which catches the chain at full gain and not at -15, because an amplifier's
gain rises with what is put into it.

**Every committed reading of an amplifier predates this.** That is 111 of the
661 fingerprints: across the library every family reads the empty loop's own
0.005% in the high band except the full amplifiers, at a median of 46.4% and
12dB louder. The preamps of the same circuits read 0.047%. They want taking
again.

And every reading taken since predates the destination fix, so it wants taking
again too. A campaign run on the Multi destination measured each block with its
own output added back at its input, whether or not the guard caught it: the
refusals are where it ran away, not where it started.

### Read what is loaded, without loading anything

```bash
mise exec -- go run main.go measure chain
```

Nothing is built, nothing is loaded and no control moves. Play a preset, turn a
knob with `device turn`, and read what that did. It is the primitive the rest
of this assumed and did not have, and its absence is why a chain reading 84% of
its energy above 2kHz went a whole session unnoticed: every other command folds
the figure into a distance from a target before anybody sees it.

## How the solve actually works

One amplifier with nine controls at five positions each is 1,953,125
combinations, and a reading takes about eight seconds. Searching that is 226
days. So nothing here searches.

Instead each control is measured on its own. Move one, read what changed, and
that is a slope: so much centroid per turn of Treble, so much low band per turn
of Bass. Measuring is linear in the number of controls, and the readings are
committed under `resources/sweeps/`.

The slopes stack into a matrix and the matrix is solved for the moves that close
the gap between where the chain reads and where the target sits. That is least
squares, by the normal equations, with ridge damping scaled to each control's own
range so a control that moves nothing cannot be handed an enormous move.

Three things follow from it being a model rather than a search.

**Every row is divided by its own tolerance.** A centroid is hundreds of hertz
and a band share is under one, so without that the centroid would be the only
axis that mattered. Scaled, a residual of 2.4 means the same thing on every axis:
2.4 times as far out as the target allows.

**The model is local, so one solve overshoots.** A slope is true near where it
was read and drifts away from it. The answer is another pass from wherever the
last one landed, not a better model. Three to five passes is usual.

**A control that is a list has no slope at all.** A cabinet's twelve microphones
do not lie on a line, and the fourth does not sit between the third and the
fifth in any sense a slope describes. Those are compared rather than solved, and
the next section is how.

## Which model a gear name means

The same gear ships more than once under one name, and the name has to mean one
of them. 355 of this device's 661 models share a name with another of their own
category, so this is the ordinary case rather than a corner.

**The family decides, not the alphabet.** An amplifier before a preamp, because
a rig naming "Ampeg SVT" means the amplifier and 108 names here are both. A
mic'd cabinet before a legacy one, because the same speaker ships as
`HD2_Cab1x15TucknGo` at 7.2 DSP carrying only floats and as
`HD2_CabMicIr_1x15AmpegB15` at 2.5 carrying Mic, Angle and Position: a third of
the cost for three more controls, one of them the twelve-microphone list the
section above exists for.

**The other one is still reachable, by the name Line 6 give it.** They call the
older model `Legacy 1x15" Ampeg B-15` and the catalog records that, so:

```yaml
chain:
  - gear: 1x15 Ampeg B-15          # the mic'd one, 2.5 DSP, twelve microphones
  - gear: Legacy 1x15" Ampeg B-15  # the older one, 7.2 DSP, floats only
```

**A rig cannot name a model and is not meant to.** It names gear a person
recognises, which is what makes it portable, and the above is how that resolves.
A plan names the model outright, which is what `presets compile --plan` takes and
what `tone tune --out` writes, so anything that has to have one exact model says
so there.

## A list is compared, not solved

Twelve microphones is twelve readings. That is cheaper than the sweep of a
single dial in a long chain, and the comparison is exact rather than modelled:
each setting's figures are what that setting actually produced, so there is no
slope to be wrong about.

The scoring is the axis each setting leaves furthest out, which is the measure
the rest of the loop reports and stops on. Summing the axes instead would rank a
setting that is slightly wrong everywhere above one that is right on everything
but the axis the target cares about most, and "2.4 tolerances out" would then
mean two things in one run.

A setting is thrown away rather than scored when it mutes the chain, clips the
converters, or the device refuses it, because in each case the figures describe
something other than the setting. The clipping guard is the one that earns its
keep: one microphone of a cabinet's twelve clipped and read a centroid of
4,471Hz where the other eleven sat between 126 and 147. Scored, it wins every
target asking for a bright sound, and the answer is a cabinet nobody would have
chosen.

The output names the winner and the runner-up:

```
  comparing 1 list(s), every setting of each
    Mic: 11 of 12 settings read, nearest is 4 at 1.8 tolerances out
      next nearest is 9 at 2.1
```

The gap between those two is the useful number. Nothing between them means the
control does not matter for this target; tolerances between them mean it is the
most important thing in the chain.

Settings are numbers rather than names because Line 6 ship no symbol list for
them. The catalog says a cabinet's Mic runs 0 to 11 and nothing says which
microphone each one is, so a report can say which setting won and not what it is
called.

## Choosing and solving interleave

The order is fixed and only one way round works. Choose first, then solve,
because a choice changes every slope: the same Treble in front of two
microphones is two different numbers, so slopes read before the choice describe
a chain that no longer exists. Solving first would be worse, since it spends
every dial answering a target and then moves the thing that moves furthest.

**The nearest setting is not the last word.** It is the nearest with the dials
wherever the compiler left them, and that is a different question from which
setting a solve can finish from. A setting reading a hair nearer while leaving
every dial against a stop is a worse answer than one reading further out with
the whole range in hand, and nothing short of solving from both finds that out.
So a run that falls short backs up to the next-nearest and solves again.

That is not a theory. Matt Freeman's rig aimed at punk on an HX Stomp, one pass
each:

| MidFreq | read  | solved to |
| ------- | ----- | --------- |
| 2       | 390.5 | 52.7      |
| 1       | 420.3 | 5.3       |

The setting that read nearest solved ten times worse than the one that read
second. Without the backing up the run reports 52.7 and stops.

`--tries` is how many of those are worth paying for, and the default is two: the
nearest, and one alternative. Each costs a whole convergence, which on a chain
of a dozen dials over three passes is around forty readings. `--tries 1` skips
the backing up for somebody in a hurry.

Where a chain holds several lists, they are compared one at a time in signal
order, so each is compared with the earlier ones already on their answers. The
backing up then queues every runner-up across all of them together and tries
whichever read nearest, rather than exhausting one control before touching the
next. The best attempt is what gets reported, not the last one tried.

## The noise floor is the unit everything is measured in

Before any dial is turned, the same signal goes through the untouched chain
several times and the spread across those readings is the loop's own wander.
That becomes the lower bound on every tolerance, because an answer closer than
the loop can measure is not an answer.

It is worth understanding because a wrong floor does not fail, it flatters. The
floor is the lower bound on every tolerance, so a floor ten times too large makes
the target ten times easier and the loop reports arriving while it sits nowhere
near. A run once converged while its residual grew from 1.1 tolerances out to
2.4.

Two things make a floor wrong, and both are handled now:

- **The first reading after the audio device opens is the stream settling**, not
  the chain. Six takes of one untouched chain read the low band at 19.35 once and
  then 31.56 to 31.73 five times over. That reading is discarded before any is
  kept.
- **Any one reading can come back odd.** The spread is taken with the reading
  furthest from the middle thrown away, so one disagreement cannot set the floor
  on its own.

A healthy floor on an HX Stomp is around 0.0002 to 0.003 of a band and a few
hertz. Ten times that means something is wrong with the loop, and the thing to
check first is the signal path rather than the block being measured.

## Saying it: nudges live on the ask

A nudge goes in the ToneSpec's `nudges`, not on a flag, because it is a decision
about how something should sound and it has to outlive the session that made it.
`tone tune --ask` reads them.

```yaml
nudges:
  - word: darker
  - word: punchier
    steps: 2
```

The word carries the direction: `dark` is already the highs axis downward, so
there is no "less" to say. Write the comparative if that reads better, since
`darker` and `dark` resolve to the same move.

What comes back says what moved:

```
"darker" moves centroid from 144 to 105.1, 1 of a tolerance
"punchier" moves low from 0.97 to 0.7967, 2 of a tolerance
"punchier" moves transient from 0.76 to 0.9908, 2 of a tolerance
```

Three things that output shows.

**One word can move two figures.** "Punchy" is a tight low end and a hard attack,
and the vocabulary holds it that way because that is how players talk. Both
figures move, and `steps` scales both.

**A word the vocabulary does not know stops the run**, and so does one whose axis
nothing measures. Four of the ten axes have no figure: how much room is on a
part, how loud the strings are under a hand, which pickup was used, whether a
filter is moving. Saying "more space" is a real instruction that this cannot act
on, and it says so rather than running on with it dropped. A nudge ignored in
silence would report a tone nobody asked for.

**A figure the genre leaves free is left free.** A genre pins what its records
agree about and shrugs at the rest, and a nudge cannot move a target that was
never set.

## One step is one tolerance

A person says "a bit darker" and means a noticeable amount. A tolerance is the
spread across a genre's own records on that axis, so a step of one asks for a
sound as far from the target as those records sit from each other. Noticeable,
measured, and not a fraction anybody chose.

It is also the unit the loop already reports in, so "2.4 tolerances out" and
"nudged by one" are the same measure.

## Level is a target like any other

The loop constrains how loud the chain is, anchored at what it measured before
anything moved.

Without that it was the axis nothing defended, and a solve spends what is free.
Asked for punk, it walked an SV Beast's Master from 1.000 to 0.024 and then to
0.000 in two passes, turning the amplifier off to move the band shares slightly.
No corpus states a level, because a record's loudness is a mastering decision
rather than a fact about the sound, so the anchor is the chain's own settled
reading and the tolerance is the one chosen number in the loop.

## When it cannot get there

A bass cabinet cannot produce what a target asks above 5 kHz, because the speaker
stops, and no combination of the controls in front of it changes that.

**That is a result, and it is reported as one**: which axes were met, which were
not, and by how far. Reporting success because a file was written is the failure
this project exists to avoid. A target that cannot be reached is worth keeping,
because *the gear is wrong for the sound* is a real answer to somebody who owns
that gear.

The loop says which of two things happened. It converges, or it stops improving
and says so:

```
stopped improving at 2.7 tolerances out. The chain will not reach this target.
```

Neither is a failure of the tool. The second is the more useful answer, and it
only means anything because the floor is honest: a stretched tolerance would
have called the same chain converged.

## Where the edit goes

Into the **rig**, not the request. A knob position is only meaningful against a
chain. So "the mids are honky" becomes a change to the RigSpec's settings, and
what the session taught about what was wanted goes back into the ToneSpec as a
word, with the round itself recorded in `corrections`. Those three fields are the
`write-a-spec` skill's, including what decides a value and how far a word moves
one.

## Move one control without writing anything

```bash
mise exec -- go run main.go device turn --block 1 --param 3 --value 0.4
mise exec -- go run main.go device current --json
```

`device turn` is a live parameter edit and writes no flash. `device current`
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

## Then keep what worked

`tone tune --out` writes a **plan**, read back off the device rather than
written from what the solver believes it set. Those are two different claims
and only one is checkable: a move the pedal refused, clamped or rounded is a
move the solver still has in its own record.

A plan rather than a rig, and the choice is the point. A rig's `settings` are
seven words shared across every make of amplifier, so exporting one would keep
the chain and throw away the tuning. Every position this loop solved for is a
device parameter at an exact value, and the plan is the only layer with
anywhere to put it. `presets compile --plan` puts it back.

The rig is still worth having, for the other reason. It is portable, so
somebody else can compile it on different hardware. `presets show` prints one,
and `slots export` writes one.

Nothing is written to a slot. A slot is flash and a burst of writes has
corrupted a setlist, so tuning happens in the edit buffer and the answer leaves
as a file.


