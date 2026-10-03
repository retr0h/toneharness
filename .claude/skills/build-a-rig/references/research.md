# Research a player's gear

The rule: **a url nobody opened is worse than no url**, because it looks like
work. Nine citations here were entered from search-result titles and every one
had to come out again.

## Where to look

Search a list, not the open web. [sources.md](sources.md) is that list, what
each place is good for, the two that look like sources and are not, and the
three ways a citation passes a careless check.

## What a source has to clear

[Sourcing a rig](../../../../CONTRIBUTING.md#sourcing-a-rig) is the standard and
it applies to an agent exactly as to a person. Four things it names:

**Follow a citation to the bottom.** The default rig's amplifier was sourced to
an aggregator quoting Wikipedia citing a print magazine from 2006. Three hops,
and only the last is a source.

**Two sources disagreeing usually answer different questions.** Check whether
one is the studio and the other the stage, or one is dated to the record and
the other to a tour five years later, before deciding either is wrong.

**Correct what you find rather than reporting it.** Noticing a citation is
wrong is the start of the job, not a task for somebody else.

**Quote the sentence you are relying on.** Not the page, the sentence.

## Then check it exists

A name is not a model. Before writing anything:

```bash
mise exec -- go run main.go catalog list --search "Ampeg SVT" --json
```

If the catalog does not carry it, the request says what it wants and the
translation substitutes the nearest, or refuses if the request insisted. Both
are honest; inventing a model identifier is not.

## Choosing the replacement is research, not a shrug

Where the gear is established and the device models nothing by that name, the
rig keeps the real name and a `substitute` says what to put there instead. Which
one is a claim, so it is researched and its reasoning is written down like any
other.

Ask Line 6 first, and know what they do and do not answer. The catalog carries
their own statement in two fields, and nothing else:

- `based_on` is the real gear a model emulates. It is the only link between a
  model and the world, and it is what `catalog list --search` matches.
- `cablink` is the cabinet they voiced an amp with, so it settles **which
  cabinet** without any reasoning of yours. The build already uses it where a rig
  names an amp and no cab.

They publish no equivalence for gear they do not model. There is no "use this
instead" table, so an amplifier they never modelled is decided by the ranking
below rather than by the catalog.

In order of what settles it:

**The company history.** The strongest answer, and the one a search finds. GMT
built the amplifiers that became Gallien-Krueger: Robert Gallien's company sold
amps badged GMT, then GMT over Gallien-Krueger, then Gallien-Krueger alone. So a
GMT 150B's nearest modelled relative is a GK, and that is a documented lineage
rather than a resemblance.

**The same circuit under another badge.** Amplifiers get cloned, rebadged and
licensed. An Acoustic 150 and an Acoustic 360 are one company's preamp two ways;
a Sunn and a Fender of the same years share a circuit more than either shares
with a modern reissue.

**The era and the class.** Valve against solid state first, then the power and
the speaker complement. A 300W valve head into an 8x10 is not answered by a 100W
combo whatever the badge.

**Measurement, last.** Where nothing above decides it, name no amplifier and let
the build choose: `tone build` ranks the instrument's own amps against the
record's figures and says which it took and why. That answer is reproducible,
which a preference is not.

Write the reasoning into the substitute's own `evidence`, with the same rules as
any other claim: a company history is `cited` with the page that establishes it,
a resemblance is `llm`. A substitute chosen on a hunch and recorded as a fact is
the same failure as a guess wearing a citation, one level down.
