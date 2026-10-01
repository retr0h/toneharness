# A genre is a corpus with a name

2026-09-19

**Status: implemented.** `audio.GenresMeasured` pools records into genres and
`corpus music genres` is what prints them. Three genres are measured.

## The request

"Make me a punk bass tone."

Nothing here can answer that, and the reason is the same one that made every
character word a guess: punk is a word somebody would have to define, and
whatever they typed would be their opinion wearing a data structure.

## What punk is

It is where punk records sit.

Measure a pile of them in the nine figures everything else is measured in, and
the region they occupy is what the word means. Nobody defines it and nobody
argues about it in the abstract: it is a fact about a set of recordings, and
anybody holding the same recordings gets the same answer.

So a genre needs no new machinery. It is a selector over the corpus, and the
corpus already exists.

### The average punk record is not punk

This is the trap, and the project already knows the way out of it.

The middle of a cluster is the least distinctive thing in it. Average every punk
bassline and you get something that sounds like nothing, because whatever makes
punk recognisable is what separates it from everything else, not what its
members have in common with the general population of recorded bass.

Which is exactly how a character word is already earned here: a word is not
awarded for a high reading, it is awarded for sitting **outside the middle half
of the other players**. Pino is `dark` because his centroid is 96Hz against
172Hz for everybody else, not because 96Hz is intrinsically dark.

A genre is defined the same way. Punk is the axes on which punk records sit
apart from non-punk records, by how far, and the target is that displacement
rather than the cluster's centre.

**What follows from that:** a genre cannot be measured from its own records
alone. It needs the rest of the corpus to stand against, the same way a player
does. A corpus of nothing but punk defines nothing.

## How to hold it

**Tag the records, do not build a second corpus.**

A directory per genre would force every record into one box, and a record is not
in one box. *Longview* is punk, and it is 1994, and it is a pick played with a
Precision into an SVT. Each of those is a thing somebody might ask for.

So genre goes in the manifest beside the year:

```yaml
- track: longview
  year: 1994
  genres: [punk, pop-punk]
```

and a genre becomes a query: every record tagged punk, measured against every
record not tagged punk. One record serves as many genres as it belongs to, and a
genre nobody has tagged yet costs nothing to add later.

The same shape answers questions nobody has asked yet. Records carry a year, so
"a nineties sound" is the same query with a different predicate. `played`
carries the instrument, so "a Precision sound" is too.

## Which genres, and when one counts

**Not a list in the contract.** Enumerating genres in an OpenAPI schema means a
release to add one, and the thing that decides whether a genre works is not the
schema. The field takes a free string.

What decides it is whether enough records carry the tag to have a distribution.
The threshold:

|                                        |                                                      |
| -------------------------------------- | ---------------------------------------------------- |
| **8 records, from at least 3 artists** | usable                                               |
| **15 or more**                         | trustworthy                                          |
| fewer                                  | reported as not yet a genre, and refused as a target |

Three artists is the part that matters. One artist gives you that band's sound
wearing a genre's name; two gives you a scene. The whole point of a genre is
that it is what several people have in common.

Where the corpus stands against that:

```
31 records, 9 artists, one artist per sound
punk: 3 records, 1 artist   -> not a genre, it is Mike Dirnt
```

So today the honest count of supported genres is **zero**, and the tool should
say so rather than compute a target from three records.

### The three to start with

**punk**, **pop-punk** and **grunge**. Chosen because they are what somebody
asked for, they are adjacent enough to be worth telling apart, and 1990s
material is easy to source and easy to date.

Adjacent is the useful part: a genre is defined against everything it is not,
and three neighbours make a harder and more meaningful test than three genres
that sound nothing like each other. If punk and pop-punk cannot be separated in
the nine figures, that is worth knowing early.

## Who says a record is punk

A required field when a record is added, and the answer carries how it was
decided.

**A model labels it by default.** That is the fast path, and it is exactly what
the `llm` evidence kind already means here: asserted by a model and checked by
nobody. A tag carrying that kind is a tag somebody may overrule, and the tool
lists them for review rather than hiding them among the sourced ones.

**A person labels it when they say so**, or when they disagree with what the
model said. That is the end of the argument; nothing recomputes it.

The reason to write this down is that it is a workflow rule rather than a schema
rule. Adding a record means: the track, the year, the url, the source, and now
the genres. A record arriving without them is incomplete in the same way one
arriving without a year is, and
[the era check](../../../CONTRIBUTING.md#changing-a-source-is-never-one-file)
exists because that kind of gap is invisible until something depends on it.

## What a person actually asks for

The genre request is one of a family, and building only for it would be a
mistake. What a person says, and what each needs:

| They say                              | It needs                                                  |
| ------------------------------------- | --------------------------------------------------------- |
| "a punk bass tone"                    | a genre's displacement from the rest of the corpus        |
| "like Mike Dirnt"                     | one player's, which already works                         |
| "like Dirnt but darker"               | a target, then a nudge along one axis                     |
| "chunkier"                            | a nudge with no target at all, relative to what is loaded |
| "like this song" *(hands over audio)* | measure it and aim at the result                          |
| "I have a Jazz, not a Precision"      | the difference between their instrument and the record's  |

Only the second of those works today.

Three things fall out of the table:

**A target and a nudge are different verbs.** "Like Dirnt" picks a point to aim
at. "Darker" moves along an axis from wherever you are. A system that only
understands targets cannot take the second instruction, and conversation is made
almost entirely of the second.

**Some requests have no record behind them.** "Chunkier" is not a claim about
anybody's playing. It is a direction, and the honest thing is to treat it as one
rather than invent a citation for it.

**The last row is the one that makes a preset usable.** Every figure in the
corpus is measured off a record made with somebody else's instrument. A preset
aims the amplifier at that record while the person is holding a different bass
with different strings. The difference between the two is a knob position, and
nothing here has ever had anywhere to put it.

## Tweaking is the normal case

Nobody gets what they want first time, so the interesting state is the one
between asking and saving.

```
  ask ──▶ rig ──▶ listen ──▶ "more of that" ──▶ rig ──▶ ... ──▶ save
                    ▲                                              │
                    └──────────────── or abandon ──────────────────┘
```

Everything in the middle is a working state: a rig that exists, is being moved,
and is not yet anything anybody would publish. Today there is no such state. A
rig is a file on disk and every change is an edit to it, so the twelfth attempt
overwrites the first and nothing records that eleven were tried.

That working state wants three things a file does not give:

- **It remembers the moves.** "Darker, then a bit less dark" should be able to
  go back one step, which means the moves are a list and not just their result.
- **It knows what is asserted and what is being tried.** The evidence in a rig
  is the part somebody researched. A nudge is not evidence and must never become
  it by sitting in the same file long enough.
- **It can be thrown away.** Most of them should be.

Saving is then a deliberate act that turns a working state into a rig, and the
moves that got there become the record of how it was arrived at. The format
already has a place for that: `mutations`, which records an ask, what moved, and
why. Nothing writes it yet.

## Now that something can hear

The loop closed on 2026-09-18, and it changes what can be offered.

A rig can be **rendered**: pushed through the pedal, recorded, and handed back
as audio. So "here is your punk tone" can come with the sound of it, on the
person's own hardware, rather than a file and a hope.

It also lets the tool check its own work. Build the rig, render it, measure the
render, and compare that to the target it was aiming at. If it missed, say so
and by how much, rather than reporting success because the file was written.

That is the difference between "the rig validates against the catalog" and "the
rig sounds like what was asked for", which this project has been careful to keep
apart and has never been able to close.

## What this needs, in order

1. **Genres on records.** A field in the manifest, tagged across the corpus.
2. **A genre's displacement**, measured the way a player's is, against
   everything not in it.
3. **Nudges as a verb.** A move along an axis from where a rig currently is,
   with no target and no pretence of evidence.
4. **A working state** that holds a rig mid-conversation, remembers the moves,
   and keeps them apart from what was researched.
5. **Render and check.** Push a built rig through the pedal, measure what comes
   back, and report the distance from the target.

The first two need no hardware. The last needs what
[measuring.md](../../measuring.md) describes, which now works.

### Items 1 and 2 are built, corrected 2026-09-27

Genres are on records, with `genres_by` beside them saying whether a model or a
person decided each set. That field is not in the plan above and should have
been: a tag nobody checked and a tag somebody overruled are worth different
amounts and look identical once both are a word in a list.

The displacement is built, and what it ships in is worth recording. The audio
cannot travel, so `pkg/sdk/audio/data/genres.json` carries what each genre
measured as, written by a generator `just generate` runs and read by `go:embed`.
Asking for a genre therefore costs no audio, which is the same split the preset
corpus already used.

### A genre cannot use the rule a player uses, corrected 2026-09-27

Item 2 above says "measured the way a player's is". That does not work, and the
reason is structural rather than a threshold anybody can tune.

A player earns a word where their whole spread sits outside the middle half of
the others. That suits three or four records by one person. A genre pools
several players, so its spread runs two to three times wider: measured on this
corpus, 64 to 92Hz of centroid against 17 to 41Hz for a player. A spread that
wide cannot sit clear of anything, so the player's rule earned every genre
nothing whatever the figures said.

John settled the replacement on 2026-09-27: a genre's **middle** against the
others' middle half, carrying the margin. Still displacement, and it no longer
demands that every punk record be extreme.

### What the three genres actually earn

The section above hoped adjacency would make a harder test. It did, and the
answer is more interesting than expected:

| genre    | records | players | earns                                          |
| -------- | ------- | ------- | ---------------------------------------------- |
| grunge   | 9       | 3       | `scooped` (0.01 v 0.06), `clean` (0.13 v 0.24) |
| pop-punk | 9       | 3       | `clean` (0.17 v 0.23)                          |
| punk     | 12      | 4       | nothing                                        |

Punk has the most records and the most players of the three and earns nothing:
it sits inside the middle half of non-punk bass on every axis. This record asked
for that finding early, in "if punk and pop-punk cannot be separated in the nine
figures, that is worth knowing early", and the answer is that punk cannot be
separated from bass in general at all.

So the threshold says a genre has enough behind it, never that it sounds like
anything in particular. Those are two claims and the tool now reports both
separately: `measure genres` prints "a genre" against grunge and "sets nothing
apart" against punk.

pop-punk's `clean` sits 0.01 past the line, which is the weak case the margin
exists to expose rather than a result to lean on.

## Related

- [measuring.md](../../measuring.md), the loop this leans on
- [a recipe compiles to a rigspec](2026-09-18-a-recipe-compiles-to-a-rigspec-design.md),
  which is where a saved working state would land
- [nothing here has ever heard anything](2026-09-18-nothing-here-has-ever-heard-anything-design.md),
  which argues why a word has to be measured rather than declared
