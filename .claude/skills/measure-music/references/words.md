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

**The quarter was not taste, and is now a choice.** Measured across all 21
genres on mid, centroid and harmonics, the margins as a share of the others'
spread fell in two groups with nothing between them:

```
0.00 0.00 0.01 0.02 0.03 0.04 0.05 0.08 0.11 0.12 0.14 0.16 0.18 0.19 0.21
0.38 0.41 0.44 0.46 0.49 0.51 0.57 0.58 0.69 0.83 1.27 2.41
```

Any threshold between 0.21 and 0.38 answered the same way, so a quarter was the
round number inside the gap. A tenth was available and refused: it would have
separated southern rock from pop punk, which is the pair this corpus most wants
separated, on margins of 6Hz. Choosing a threshold for the answer it gives is
how a measurement stops being one.

**Adding decay as a fourth axis closed that gap.** The same measurement over
four axes runs straight through it, 0.20 0.20 0.20 0.20 0.25 0.27 0.29 0.29
0.30, with no break anywhere between 0.02 and 1.00 wider than 0.05. Sixty-three
points had a gap and seventy-three do not, which is what usually happens to a
gap found in a small sample.

So the quarter is a chosen threshold now rather than one the data picks out, and
it is kept because moving it moves every word in the corpus on no evidence. Say
that when quoting it: the number is the same and what stands behind it is
weaker.

Point it at one instrument. A guitar's centre of gravity sits an octave above a
bass guitar's, so pooling both would earn every bassist `dark` and every guitarist
`bright` and mean nothing by either.

`holds` is the margin past the middle of the others, in the measure's own unit,
which is what tells one word from another. A word with a wide margin survives
the next player being measured; one just over the line is the next thing to
move. Two players may share a word and not be alike, and this is where that
shows.

## Nothing earns `clean`, and the axis it came from has one pole

`saturated` is earned from the share of energy above the fundamental. Its
opposite is not, and the corpus is why.

Four of the five players earning `saturated` have a drive device written into
their own rig: Chuck Dukowski, Tim Commerford, Matt Freeman, and Tobin Esperance
with a Malekko B:Assmaster on his board. The high end of that measure tracks
something real.

The low end tracked nothing. Three of the four players that used to earn `clean`
have documented gain. Chi Cheng read 0.081, and his own rig quotes Premier
Guitar on the Tech 21 SansAmp GT2 he used "for extra balls". Troy Sanders read
0.059, the cleanest figure in the corpus, and runs a Wren and Cuff Tall Font
Russian distortion. Klaus Flouride read 0.077 on Dead Kennedys records nobody
would call clean. Grunge earned `clean` as a genre, which is the one thing
grunge is not.

**The separation is the reason, not the arithmetic.** Distortion products are
broadband and sit where the guitars sit, so splitting a mix assigns them to the
guitar stem, and the dirtier and denser the record the more of them leave.
Grunge, Deftones and Mastodon are the densest mixes here and they read the
lowest. A measure reporting its own blind spot as a positive finding is worse
than one that says nothing.

It is not brightness in disguise, which was the other suspicion worth ruling
out. Centroid and harmonics correlate +0.43, the four former `clean` players
span 97Hz to 191Hz of centroid, and Tim Commerford earns `saturated` at 109Hz.
Dark and saturated is reachable, so the axis is not reading the top end twice.

`clean` is still a word a person may write in an ask, and it still moves Drive.
What changed is that no measurement asserts it. Restoring the pole needs a
measurement of breakup rather than of harmonic content, taken before separation
or on a signal that never needed it. The word means "no breakup at any dynamic",
and a median share of energy cannot speak about any dynamic: somebody who breaks
up only when hit hard reads low, and `grit-on-attack` and `minimal-drive` exist
for that.

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

## Decay earns a word now, and still moves nothing

It was refused twice before, and the measure was wrong both times. What it timed
was the arrangement: a note was followed until its level fell by 12dB, and in a
dense mix that is a section change rather than a string stopping, so God Save
the Queen read 21.01 seconds a note against 1.44 on another track by the same
player on the same rig.

Two rules fixed it. Onsets are peak-picked and no closer than 120ms, where
before every frame that got louder started a note: 6,131 of them in 199
seconds, thirty a second. And a note is given two seconds to die and dropped if
it does not, rather than timed to wherever it eventually falls. The worst
drifter in the corpus went from 19.57 seconds of spread across one player's
three records to 0.62.

**It passes the same test the other three pass.** Drift inside one player over
the spread between players: centroid 0.60, decay 0.69, harmonics 0.77, mid 0.79.
So `short-decay` and `long-decay` are earnable, around half the players earn one,
and every aimable genre now earns at least one word where four earned none.

**And no preset changed.** Decay describes the hands and the strings, and no
amplifier, compressor or reverb on this device has a control for it, so a word
earned on it is recorded and moves nothing. That is why hard rock, pop punk and
southern rock still resolve to one chain: the figure that separates them is real
and there is nothing in a Helix to spend it on. Knowing it is still worth
having, because it is what somebody compares the preset against by ear.

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
