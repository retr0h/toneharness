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

## A block that changes the notes is refused, not ranked

A fingerprint is four figures off one block measured alone, so two pedals that
land in the same place are interchangeable to it however differently they
behave. An octave fuzz is the case that matters. Of the 61 drives an HX Stomp
models, Octave Fuzz reads a centroid of 213Hz with 22.0% of its energy in
harmonics, and Ballistic Fuzz reads 226Hz and 22.6%. Nothing in those numbers
tells a fuzz from a fuzz that adds a note nobody played.

So the pool is narrowed before it is ranked rather than the ranking being
taught something it cannot measure. Every one of the 31 models Line 6 group
under `pitch` is out, and so are ten they file elsewhere: five drives that are
octave fuzzes, four ring modulators and frequency shifters under `mod`, and one
harmonised delay. `catalog.Block.ChangesTheNotes` owns which blocks those are
and the argument for each.

It bites. Across the eleven genres the drive pool is 52 where it was 57, and
the refused blocks were not far down the list: pop-punk's Sub Oct Fuzz read
0.69 from the target against the winner's 0.21, and rock's Octave Fuzz 0.87
against the Metal Zone's 0.68. Five of the eleven had one in second or third
place, on margins a single record moves.

Only where the tool is choosing. Gear an ask names by hand is a decision
somebody made and resolves as it always did, which is why Flea's rig still
names a Maestro Bass Brassmaster: an octave fuzz is the right answer when it is
the pedal somebody is documented with.

Where the refusal changed which block got chosen, the build says so:

```text
did  drive  Tycobrahe Octavia read nearest and was refused: it changes the notes rather than the tone
```

**What this does not fix.** Two of the three things a fingerprint cannot see
are still there. It does not know how a block behaves in front of an
amplifier, because every sweep is of a block alone. And it does not know what
a pedal is for: a Big Muff is a fuzz with a mid scoop players reach for
deliberately, and measurement picked one for blues-rock. Preferring a pedal a
genre's own players are documented with would answer that, and today there is
nothing to prefer. The Pro Co RAT that Krist Novoselic played reads 39th of
grunge's 52 drives, 28.7 from the target against the nearest 0.92, and the
Brassmaster reads last of alternative-rock's.

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

**The cabinet gets its microphone, and that is the only lever it has.** Measured
over every swept mic'd cabinet, the microphone moves the centroid a median
29.2Hz and nothing else on the box moves it 5: Distance 4.2, Position 2.7,
HighCut 2.5, Angle 0.8, Level 0.0. `LowCut` moves 190, and is bounded to the
instrument's lowest note because every one of those hertz is above it, which
leaves the microphone alone.

It is chosen by displacement rather than by position, because twelve impulse
responses have no order to run along: a microphone 11Hz brighter than the
others on its cabinet answers a target 20Hz brighter than the other records.
Comparing absolutes sent all eleven genres to microphone 0, and letting the
same reading set the rest of the cabinet sent LowCut to 79.9Hz.

**A cabinet the compiler adds is chosen for too**, which took two goes. A rig
naming its own cabinet has it chosen where the gear is; a rig that borrows its
amplifier's own pairing, and every artist rig, has it added later and reached
neither. 22 of the 43 shipped rigs sat on whatever microphone the corpus median
gave their model, which is one answer for everybody: 14 on one cabinet all read
microphone 3. They now spread across four.

### What computes and what a document shows are not the same figures

A measurement is reported to the places it can honestly claim. A centroid goes
to the hertz because the bins are about 10Hz apart, and a share to two places
because the third moves with which windows the playing fell in.
`audio.Across.Measured` and `audio.Profile.Measured` do that rounding, and they
are the maps a rig's evidence carries.

**The arithmetic read them too, and it cost three genres their displacement.**
A genre's figure is a median over its records, each itself a median over
windows, and averaging is what buys the places back: rounding a median over
sixteen records to one take's precision throws away precision the averaging
earned. hard-rock's records sit 0.5Hz and 0.0008 of a share from the players
who hold none of it, which read as a centroid of 132 against 132 and a mid
share of 0.03 against 0.03. Nothing to aim at, on a figure whose whole range
across the eleven genres is 0.02 to 0.11, where two places leave ten
distinguishable values.

So `audio.Across.Figures` and `audio.Profile.Figures` are the same middles
unrounded, and everything that subtracts two figures or spans several of them
reads those: `middles`, `elsewhere`, `apart` and `displacedTo`. `Measured`
stays what goes into a document, so nothing is claimed to more places anywhere
a person sees a figure.

**The floor moves to the difference, because it was never really in the
inputs.** Dropping the rounding without saying so is worse than keeping it: a
genre sitting half a hertz from everybody else then asks for half a hertz, and
a Deluxe Comp's Threshold has 0.14 points of midrange of authority, so it
reaches that ask and goes to a stop delivering noise. `audio.Resolves(key, by)`
is the floor stated on the difference instead, the same one `Axis.resolves`
holds a word to. hard-rock asks for nothing again, southern-rock asks for 7.05
hertz of centroid and nothing on the midrange, and rock asks for 11.44 rather
than the 11.00 the rounding left it.

The same floor had to be stated in `audio.scattered`, which compares a player's
own span against the whole corpus's. Both sides came off `Measured`, so a
difference below the measure's precision could not exist, and the moment the
inputs stopped being rounded a float's tail read as a player whose records
disagree: three sine tones a hundred hertz apart reported a scattered mid band,
where every one of them holds the same nothing in it.

Where a target leans nowhere on any weighed axis the median stands, which is
what it was there for. Five of the eleven genres earn no term at all, so their
displacement is the middle of their own range, and one of the five leans
nowhere.

**The chain follows the character, not the corpus's habits.** Every shipped
genre rig carries a compressor justified by "a chain of this kind almost always
holds one, and the corpus statistics are what put it here". Punk's transient
range is 0.664 to 0.839, which is high, and a compressor lowers transient. The
statistics are not evidence about the genre.

**Extra blocks arrive bypassed, each on its own switch.** A genre's records
cover a range no single setting reaches, so give the range to the player's feet
instead of picking one point in it: the base sound is the amplifier, and the
blocks that reach further into the genre sit switched off until somebody steps
on one. `compile.Assign` does the assigning and the colours, and which kinds a
rig gets is the next section.

The corpus is reference material for understanding a genre, not a population to
average. Averaging is what produces one preset eleven times.

## A bass is weighed on three figures, and harmonics is the third

`measured.SpectralFor("bass")` drops the low and high bands and leaves the
centroid at a thousandth and the midrange at one. For a long time that was the
whole of it, and the sweeps hold ten figures per position.

**Harmonics is the one of the other eight that can be weighed.** It is a share
of the energy above the fundamental, a record and a block go through the same
`audio.Measure` to get it, and the measuring loop's second of padding cannot
move it: a share does not change when silence is added to either end. That is
exactly the argument that disqualifies transient and decay, and it does not
apply here.

What it is worth, measured. The blocks span harmonics from 0 to 86 of a
hundred, and four of the eleven genres displace it past what the figure can
claim:

| genre | harmonics it asks for |
| ----- | --------------------- |
| grunge | -6.48 |
| blues-rock | -6.33 |
| metal | -1.76 |
| rock | +1.41 |

The other seven displace less than a hundredth of a share and ask for nothing,
which the floor on a difference already handles.

Weighed at one, like a band share, because it is reported on the same scale: a
block's reading and a genre's displacement are both hundredths of the energy.

**It broke the two-answer microphone.** A list is chosen by direction, the
midrange column of the eleven displacements held two signs, and 22 of the 43
rigs reached the same microphone. With harmonics in the ranking the 43 spread
across six microphones rather than five, and the 22 become 13 on one and 10 on
another.

**And it cost a Metal Zone.** rock's drive was a BOSS Metal Zone MT-2, which is
the headline of the complaint about measurement choosing badly; it is now a
Carvin VLD1 Legacy Drive. pop-punk's amplifier and cabinet move from a
MESA/Boogie Bass 400+ into a 6x10 to an Ampeg SVT into an 8x10, which is what
punk bass is. Tim Commerford's drive moves from a Roland Jet Phaser, which the
device files as a drive and is a phaser, to an Arbiter Fuzz Face; he earns
`saturated`. Tobin Esperance's moves off a Bitcrusher.

**Two went the other way.** blues-rock's amplifier moves from an Ampeg SVT
(bright channel) to a Pearce BC-1 preamp (jumped), which is a worse answer by
ear on a genre whose amplifier was always a measured guess: 2 of its 3 players
have a researched rig and no amplifier is in most of them. And Greg T. Walker's
drive moves from a Z.Vex Fuzz Factory to a Xotic EP Booster although he earns
`saturated`, because the Fuzz Factory's own harmonics reading is 0.000 and the
booster's 38.3 is nearer the 41.6 he asks for. The arithmetic is right and the
reading is the thing to doubt.

**Ten rigs move from a Simple Tilt to an MXR 10-Band Graphic EQ**, which is the
change worth understanding rather than worrying about. The eight bass
equalisers span harmonics from 9.1 to 37.8, and the 10-band sits at 37.795
against a baseline of 37.794: it is the most transparent of them. A genre that
asks for no change in harmonics gets the equaliser that does not change them.
Its bands come out as a curve rather than against the stops, which is what the
reach rule fixed.

### What is still not weighed, and why

`level` stays at zero on purpose: a block that is right and quiet is right, and
the next block's level control fixes it for nothing.

`transient` and `decay` stay at zero because the measuring loop pads its
recording with a second of silence to catch a reverb's tail. That moves both,
the same way for every block and not at all for a record, so a block may be
compared with a block on them and never with a record. Undoing it would mean
measuring the padding's own contribution and subtracting it, which is a sweep
nobody has run.

`dynamics` and `lean` are measured on both sides and nothing has established
whether the padding moves them. They are the next two to look at.

### Being nearest is not the same as having been chosen

`measured.Library.Nearest` always answers. For most kinds of block the answer
means something, and for one it does not: of the eight equalisers this device
models for a bass, punk's nearest sits 24.49 from its target and the median of
the eight sits 24.51. Two hundredths, on a scale where a hundredth of a share
of the midrange is worth one.

So a choice has to clear a bar, and the bar is the smallest distance the
comparison can tell apart: `measured.Weights.Resolves`, which is the distance a
one-unit difference on every weighed figure produces. A hertz of centroid and a
hundredth of a share each, which for a bass is 1.41.

| kind | how far the nearest beats the median, across the eleven genres |
| ---- | ------------------------------------------------------------- |
| amplifier | 9.4 to 23.9 |
| cabinet | 9.0 to 17.4 |
| drive | 12.4 to 20.7 |
| compressor | 2.2 to 16.4 |
| reverb | 2.5 to 3.9 |
| **equaliser** | **0.02 to 3.15, and ten of the eleven miss the bar** |

Only grunge's equaliser is a choice. The other ten are the first of eight
blocks nothing can separate, broken by the model's own identifier so the answer
is at least the same every run.

**The block stays and the claim goes.** An equaliser nobody could choose is
still an equaliser, and its controls are set from its own sweep, which is where
the shaping an equaliser does actually happens. What changes is the evidence:

```text
Not chosen by measurement. None of the 8 bass equalisers this device models is
measurably nearer punk than the others, so MXR 10-Band Graphic EQ is the first
of them rather than a choice. The model is arbitrary and the controls on it are
not: those are set from its own sweep.
```

That is task 267's own complaint answered as far as it can be without
hardware. Its words were that a genre's drive "is the nearest measured block
and the document says so in its evidence, which is honest but is not a
recommendation anybody should trust by ear". For ten of the eleven equalisers
it was not even honest.

**The median rather than the runner-up.** Two blocks may differ while the other
six are alike, and what decides whether being first meant anything is the field
rather than the second place.

## A rig gets the effects its own records ask for

Every chain the corpus built carried the same two, a drive and an equaliser, on
all eleven genres. None of the 32 artists carried any, because the list was
reached only where the ask named one of the eleven genres the bass corpus can
aim at, and an artist rig's chain is authored and never passes through that
path. The eight artists that had a switch had it because their researched rig
names a pedal of its own: Flea's Maestro Bass Brassmaster, Krist Novoselic's
Pro Co RAT, Bootsy Collins's Mu-Tron III.

So the kinds come off the records instead. A target earns a word where one of
its figures sits outside what everybody else measures, and that word is the
only thing that says a block of some kind is wanted rather than merely
available. Four figures ever earn one across the 43 rigs here: decay 25 times,
the mid band 20, the centroid 17 and harmonics 4.

| the word | the figure | what it gets | why |
| -------- | ---------- | ------------ | --- |
| bright, dark | centroid | an equaliser | where the energy sits is what an equaliser moves |
| mid-forward, scooped | mid | an equaliser | the same control of the same thing, so a target earning both gets one |
| saturated | harmonics | a drive | energy above the fundamental is what distortion makes |
| long-decay | decay | a reverb | what the figure measures is a tail, and a tail is what a reverb adds |
| short-decay | decay | nothing | see below |

`translate.Stompable` owns it and `presets.Make` calls it with the document's
own identifier, which is the route task 268 established: a rig for somebody
aiming at their own records is not circular, and the identifier is what reaches
it without reading `ask.subject` as a target.

Four rules around the choosing:

**Ordered by how far past everybody else the records sit**, as a share of what
the others measure. A margin is in its figure's own units, so the raw numbers
rank by unit rather than by distinctiveness: the centroid margins here run from
10 to 78 hertz and the harmonics margins from 0.03 to 0.08 of a share. As a
share of the others' median, Jaco Pastorius's 78Hz of centroid is 0.6 and Matt
Freeman's 0.08 of harmonics is 0.27, which are two numbers about the same
thing.

**A kind the chain already holds is not doubled.** Matt Freeman earns `bright`
and `saturated` and gets both; a rig that already names a drive gets only the
equaliser. Two blocks of one kind would also light the same, because
`compile.assign`'s colour table is keyed by category.

**Capped by the switches the device has.** A block past the last switch is one
no foot can reach, and the foot is the whole reason for it. An HX Stomp has
three, and a researched pedal already on the floor spends one of them.

**Switched off, every one.** A block chosen because a figure sits outside
everybody else's is for reaching the far end of what those records cover, not
for sitting in the signal path.

Each one lands in the computed half of the document with its own evidence: what
it measured nearest, and the word that asked for a block of its kind at all.
The authored half is untouched, because a block chosen from a word is not
something somebody wrote.

**A shorter decay has no answer, and that is most of what is left.** Fourteen
of the 43 are distinctive only on it and seven more earn no word at all, so
eight rigs end up with nothing after what their chains already name. A gate
would shorten a note by cutting it off and a compressor by holding the front,
and neither is a thing to put under a foot on a bass, which is why
`compile.stomped` excludes the compressor in the first place. Nothing is the
honest answer: reaching for a block to fill a switch is how a Metal Zone ended
up on rock bass, which is task 267.

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
them, and `rigs list` says who does. Where their records are measured, those
figures are what the controls on that chain are set to, so the gear is cited and
the knobs are a reading.

**`subject:` resolves nothing, on purpose.** It says who the ask is for and
`like:` says what to aim at. An ask carrying only a subject is told which field
it wanted rather than refused in silence.

## An artist rig is set to their own records

A rig whose gear is authored still has every knob to set, and the records listed
on it are what sets them. Before this each of the 32 named the amplifier a
source puts somebody on and then carried the catalog's defaults: Dee Dee Ramone
sat at Bass 0.41, Bias 0.74, ChVol 0.8 and Drive 0.6, which is the factory
setting to the digit, while his three records sat on the document unread. Across
the 14 rigs sharing an Ampeg SVT, every control read the same on all 14.

The rig's own identifier is who it is for, and it is also the directory the
corpus files their records under, which is what makes this answerable without a
field for it. So `rigs resolve --id <player>` aims, and it needs `--sweeps`:
only the fingerprints are packed into the binary, so a release sets nothing and
the controls stay where the corpus medians put them.

**Their middle, not an edge, which is the opposite of a genre.** A genre aims at
its edge because the middle is what every genre shares and ten of the eleven
displace less than 6.4 points from one baseline. One player's records describe
one person, so their middle is the thing being aimed at rather than the thing
that hides them. Where their own records disagree more than players disagree
with each other, the report says so and that figure describes none of their
sounds.

**Still a displacement.** A player is measured off finished records and a block
off a dry signal through one block, so the two do not subtract. What is
comparable is how far each sits from its own normal, and the pack carries what
the other players measure as for exactly this.

**The cabinet is not aimed.** A sweep reads one control at a time and cannot see
that the cut taking the most energy out of the low band is also the one taking
the signal with it: letting this set a whole cabinet sent LowCut to 79.9Hz,
which is above the fundamental of a bass's open E. The cabinet stays the one its
sources or its amplifier's own pairing name.

It moves real distance. The 14 Ampeg SVT rigs now span Drive 0.5 to 0.875 and
Mid 0.085 to 1.0. Several land on an endpoint of their control's sweep, which is
what a monotone control does when it is read alone, and it is the limit of
reading a sweep rather than a fault in the aim.

**`built:` is where it lands and `rig:` is untouched.** Computed controls written
into the authored half become an override the next resolve honours, which is how
eleven genre rigs went on reading as four presets after the compiler had been
taught to tell them apart. A resolve replaces `built:` whole and never writes
where the gear and the citations live.

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
