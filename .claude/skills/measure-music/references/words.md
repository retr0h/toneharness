# What the figures earn

```bash
mise exec -- go run main.go measure players --corpus resources/music/bass --json
mise exec -- go run main.go measure players --corpus resources/music/bass --evidence
```

The only mode that produces words, because a word is earned against the other
players and one player has nobody to be measured against. `nothing` is the
ordinary answer rather than a failure: a player whose own records disagree with
each other earns nothing, and so does one sitting where everybody else sits.

## A word describes a sound, it does not prove it unique

A word is earned by sitting on one side of the **middle** of the others, by at
least a quarter of their spread. Not outside their quartiles, which is what it
used to be, and the difference is that **words overlap**.

Any number of players or genres may be darker than typical. Nothing takes the
word off one of them because another arrived carrying it too, and what a genre
is, is the combination it earns rather than the one word nobody else has. Six
genres on this corpus read `dark` and four read `scooped`.

That is a change from a rule that kept quietly removing words. Grunge earned
`scooped`, `dark` and `clean`; twenty-five records of rock, metal and nu-metal
joined the corpus, the metal players are dark, and grunge earned nothing.
Grunge had not changed. Darkness had stopped being unusual, and because these
words move controls, a genre's preset was depending on which unrelated genres
happened to have been measured.

Distinctiveness did not stop mattering. It moved to the margin, which is what
`holds` prints: the word says what the sound is, and the margin says how far
from typical it sits.

**The quarter is not taste.** Measured across all 21 genres on all three axes,
the margins as a share of the others' spread fall in two groups with nothing
between them:

```
0.00 0.00 0.01 0.02 0.03 0.04 0.05 0.08 0.11 0.12 0.14 0.16 0.18 0.19 0.21
0.38 0.41 0.44 0.46 0.49 0.51 0.57 0.58 0.69 0.83 1.27 2.41
```

Any threshold between 0.21 and 0.38 answers the same way, so a quarter is the
round number inside the gap. A tenth was available and refused: it would have
separated southern rock from pop punk, which is the pair this corpus most wants
separated, on margins of 6Hz. Choosing a threshold for the answer it gives is
how a measurement stops being one.

Point it at one instrument. A guitar's centre of gravity sits an octave above a
bass guitar's, so pooling both would earn every bassist `dark` and every guitarist
`bright` and mean nothing by either.

`holds` is the margin past the middle of the others, in the measure's own unit,
which is what tells one word from another. A word with a wide margin survives
the next player being measured; one just over the line is the next thing to
move. Two players may share a word and not be alike, and this is where that
shows.

## `scattered` is the other reason nothing was earned

Two players earn nothing for opposite reasons, and the table tells them apart.
One sits where everybody else sits, which is a fact about them. The other's own
records disagree with each other, which is a fact about which records were
chosen: their figures are an average of several different sounds, so the average
describes none of them.

`earns` reads `nothing: scattered` for the second, and `holds` says on which
measure and by how much:

```
chuck-dukowski  3  nothing: scattered  own centroid spans 211 Hz, 1.2x the others' spread
```

One figure per record, and the span is how far those sit from each other,
against how far the other players sit from each other end to end. Span against
span, so the ratio compares like with like. Above one is the line, and it is
derived rather than chosen: a player whose three records sit as far apart as the
entire corpus of players does is a player whose average is an average of
different sounds. Under it the corpus is more varied than they are, and the
average stands for something.

It appears beside a word as well as instead of one, because the axes are
separate. A player can be clear on the bands and all over the place on the
centroid, and the second is why a word the first predicted never arrived. Jaco
Pastorius earns `mid-forward` and `bright` with his own mid spanning 1.9x the
others' spread, so both words rest on four records across three albums and five
years.

Nothing is wrong with a scattered player, and what to do is a judgement with
three answers:

| what it is                                    | what to do                                                  |
| --------------------------------------------- | ----------------------------------------------------------- |
| one record is from another era                | drop it. [era.md](era.md) is the check that catches this    |
| the career genuinely holds two sounds         | a rig per era, each with its own records. `extends` chains them |
| the records are right and the player is varied | leave it. Say `scattered` in the caveat of any word they earn |

The third is the common one and it is not a defect to fix. What it forbids is
quoting the average as though it described a sound.

## A lost word is not a regression

Every player added changes what all the others earn, and nothing about those
players changed. Three arrivals here took two words away between them, and a
ninth took one away and gave one back. That is the population doing its job
rather than a wobble in it.

So do not chase a word that moved when a record was added. Read which side of the
comparison moved and say that. Dropping one of a player's own records cannot take
a word away at all, because the test reads their range as a tenth and a ninetieth
percentile. It can change whether they read as `scattered`, which counts records
rather than windows and is the one figure a single record moves.

## Attribute the figure to what caused it

The ask carries `played`, which resolves to no block and changes no preset, and it
still earns its place. A fretless played near the bridge is bright before an
amplifier is involved, and one player here reads 259Hz of centroid against a
middle of 131Hz on exactly that. The darkest player measured is on flatwounds.
Without
the field those figures read as the amplifier's doing, and a word derived from
them moves an amplifier control that was never responsible.

The same job on the rig side is an instrument's `records:`, naming which measured
tracks that instrument made. **Leave it out where nobody knows.** One player
recorded on two basses of the same model and never said which took which track, so
neither entry carries one. Naming two of four records is what stops the other two
being explained by an instrument that was not on them.

Getting it wrong is not free. A rig here once explained a 162Hz spread across one
session as a four-string against a six-string, and the source it already cited put
both records on the six.

## Where the word goes

A word this earns belongs in the ask's `words`, and `--evidence` writes the block
with both sides of the comparison, because the gap between them is what decides
how far the word moves a control. Half of what everybody else reads is half a
step, not a knob at zero.

Nothing writes it into the ask for you. A term is an argument, and putting one in
a file is still somebody deciding to believe it.
