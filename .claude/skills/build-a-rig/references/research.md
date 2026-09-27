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
