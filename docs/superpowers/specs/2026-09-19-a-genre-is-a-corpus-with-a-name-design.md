# A genre is a corpus with a name

2026-09-19

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

## Related

- [measuring.md](../../measuring.md), the loop this leans on
- [a recipe compiles to a rigspec](2026-09-18-a-recipe-compiles-to-a-rigspec-design.md),
  which is where a saved working state would land
- [nothing here has ever heard anything](2026-09-18-nothing-here-has-ever-heard-anything-design.md),
  which argues why a word has to be measured rather than declared
