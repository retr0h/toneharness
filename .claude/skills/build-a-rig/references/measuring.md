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

## A genre is a region, and its middle is what every genre shares

The displacement above is right in principle and far too small in practice.
Totalled across the three band shares, for all 11 genres the bass corpus can
aim at:

    punk                 18.90
    nu-metal              6.40
    metal                 5.30
    alternative-metal     4.20
    grunge                4.20
    blues-rock            2.40
    rock                  2.40
    pop-punk              2.00
    hard-rock             0.90
    southern-rock         0.90
    alternative-rock      0.20

On the same scale the amp fingerprints span mid 0.5 to 50.4. So ten of the
eleven targets sit within 6 points of one baseline inside a field 50 points
wide, and 9 of the 11 shipped genre rigs carry a MESA/Boogie Bass 400+ into a
6x10 Power House. Across the five that share identical gear, 3 of 35 controls
move at all, the largest difference being Treble 0.439 against 0.500. Six
hundredths of travel is not a sound anybody can tell from another.

Punk is the exception on that table and the exception everywhere else: the one
genre with its own chain, the one whose knobs moved, and the one that has been
played and liked. What made it good was that its aim was decisive.

`high` displaces by exactly 0.00 for every genre, because a bass stem holds
almost nothing above 2kHz. That axis cannot separate anything and should carry
no weight for bass.

So aim at the edge of the genre's own range rather than its middle:

**Gear comes from the players.** Punk means an Ampeg SVT into an 8x10 because
four of its seven documented bassists resolve to exactly that. Nearest
fingerprint is the wrong instrument for this question and it is what gives nine
genres one amplifier.

**Settings go to the edge of the range, in the direction of the words the
records earn.** `measure genres --json` already publishes the range: every
figure under `across` carries a `low`, `mid` and `high`, so this needs no new
measurement. Punk earns mid-forward and bright, and its own records span mid
0.013 to 0.237 and centroid 109Hz to 252Hz. Aim at 0.237 and 252, not at the
0.114 and 174Hz in the middle. That is still a real record of that genre, the
most genre-sounding one on that axis, rather than an average of all of them.

**Read the sweeps for which control gets there**, because the obvious one often
does not. On an Ampeg SVT it is `Bass` that raises the mid share, reaching 24.2
at its minimum against punk's range top of 23.7, while `Mid` is what moves the
centroid, 128Hz to 353Hz. So a mid-forward aim cuts the bass, which is the
opposite of what the name suggests and the opposite of where a median target
lands.

**The chain follows the character, not the corpus's habits.** Every shipped
genre rig carries a compressor justified by "a chain of this kind almost always
holds one, and the corpus statistics are what put it here". Punk's transient
range is 0.664 to 0.839, which is high, and a compressor lowers transient. The
statistics are not evidence about the genre.

**Extra blocks arrive bypassed, each on its own switch.** A genre's records
cover a range no single setting reaches, so give the range to the player's feet
instead of picking one point in it: the base sound is the amplifier, and the
blocks that reach further into the genre sit switched off until somebody steps
on one. `compile.Assign` does the assigning and the colours.

The corpus is reference material for understanding a genre, not a population to
average. Averaging is what produces one preset eleven times.

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
