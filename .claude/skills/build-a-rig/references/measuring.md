# Choose gear by measuring

The half no reasoning about names can do. A recording and a block are measured
by the same code into the same figures, so the ones that land nearest are the
ones worth trying.

## Measure a recording

```bash
mise exec -- go run main.go measure recordings --file take.wav --json
mise exec -- go run main.go measure recordings --dir ~/stems/htdemucs --json
```

WAV only. Convert on the way in with `ffmpeg -i take.mp3 take.wav`. Separate a
bass part out of a finished record with `mise exec -- just stems <in> <out>`.

Point `--corpus` at a tree holding one directory per player and the report says
which words each one's records earn. **Point it at one instrument**, never at the
whole tree: a bass centroid sits an octave below a guitar's, and a corpus holding
both earns every bassist "dark" and means nothing by it. What a word is earned
against, and how a corpus is grown, is the `measure-music` skill's.

## Let a recording choose the amplifier

A ToneSpec with `like: { recording: take.wav }` and no named amp gets the
nearest of the device's bass or guitar amplifiers to what that file measures. No
catalog entry and no citation needed: a recording is evidence of itself.

**The instrument narrows the shortlist, and it did not used to.** The catalog
groups amplifiers `Guitar` and `Bass`, 173 against 27 on an HX Stomp, and nothing
read it: a bass build ranked all 224 and a request aimed at a dry bass recording
chose a Fender Super Reverb. It now draws from the 27. Twenty-three amplifiers
carry no grouping at all and are left out of a ranking, because this is the tool
choosing rather than you naming, and a block nobody placed is one it declines to
guess at. Name one by hand and it resolves as it always did.

## Or let the genre choose it

An ask with no gear, no recording and no player still builds, from the genre it
already carries:

```yaml
genre: [punk]
instrument: bass
words:
  - term: mid-forward
  - term: tight-low-end
```

```text
did  amp  Cali 400 Ch1 is the closest of 27 measured to punk, 150 Hz against 165
```

**It is a displacement, not a position, and that distinction is the whole of it.**
A genre is measured off finished records and a block off a dry signal pushed
through it. Punk reads 97.1% of its energy low where the dry signal holds 90.8%
before any block touches it, so asking which block reaches 97.1% asks for bottom
that is not in the input: every candidate is out of range and the nearest is
whichever is darkest. That is how a request for punk once chose an Ampeg B-15NF,
a Motown flip-top.

So the genre is read as how far it sits from the records of players who hold none
of it, and that shift is applied to the signal the blocks were measured with.
Punk sits 0.021 of the energy lower and 9.6Hz brighter than those, so the target
is 92.9% low at 164.9Hz, which a block can reach. Both sides become "how far from
its own normal", which is the comparison a genre already earns its words from.

Two refusals rather than a weak answer. A genre under the threshold is reported
and never computed from, because eight records from three players is the bar and
fewer is one band's sound wearing a genre's name. And a genre measured on another
instrument is refused rather than used: a bass centroid sits an octave below a
guitar's, and the nearest amplifier to the wrong octave is a different question
rather than a worse answer.

A recording still wins where both are named. One performance measured exactly
beats the middle of a population.

## Or name the player, and get the rig somebody researched

```yaml
like:
  artist: Mike Dirnt
```

That resolves to his cited chain rather than to a measurement, and the notes say
so: `2 blocks came from the rig researched for them`. It is the stronger claim,
because every piece of that chain carries a source and a measurement carries
none.

`artist`, `band` and `song` all work, and the narrowest wins where an ask names
more than one: a song is one recording, an artist a body of work, a band several
people's. An alias counts, so `primus` reaches Les Claypool where no slug of it
would.

It needs somebody to have done the research. A player with records in the corpus
and no rig is refused with `no rig has been researched for that player`, naming
them, and `rigs list` says who does. Measuring a named player's records is not
built; a recording is the route when nobody has written their rig.

**`subject:` resolves nothing, on purpose.** It says who the ask is for and
`like:` says what to aim at. An ask carrying only a subject is told which field
it wanted rather than refused in silence.

## What the figures are

Ask, do not assume: `measured.Named()` is the list, and it has changed. The
figures a curve is reported in are not quite the figures a record is described
in, and the difference is documented where they are defined.

Two guards catch readings that look ordinary and are not, and the
`measure-a-device` skill owns both along with the rest of the sweep discipline.
If a figure here looks implausible, that is the skill to reach for rather than a
reason to re-run this command.

## What is not corrected for

**The strings.** If the request says the record was played on flatwounds and the
setup holds roundwounds, the answer says so and changes nothing. That difference
is larger than most pedals make, and nothing here has measured what it does to
the figures, so applying a correction would be inventing a number. Read the note
and decide.

**The instrument itself.** A Setup names what is in the room and that decides
which instrument the rig is for, which feeds block ordering. It does not correct
for a Precision against a Jazz.

Both wait on evidence rather than on code. Either measure a bass with each set
of strings through the loop, or tag the corpus records with what they were
played on and derive the difference across players.
