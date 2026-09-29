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
fifth in any sense a slope describes. Those are chosen by comparison rather than
solved, which is not built yet.

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

## Then export what worked

Read the device back as a rig and keep it. That is the artifact worth having,
because it is deterministic and somebody else can compile it.


