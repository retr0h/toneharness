# Research a player's gear

The rule: **a url nobody opened is worse than no url**, because it looks like
work. Nine citations here were entered from search-result titles and every one
had to come out again.

## Where to look

Search a list, not the open web.
[Where to look, and what not to accept](../../../../docs/workflows/create-a-rig-for-a-player.md#where-to-look-and-what-not-to-accept)
is that list, in rough order of how often each settles something. The short
version: `web.archive.org` for dead magazines, forums for people who were
there, and rig rundowns before video.

Read the forums with the tooling, which handles what each site refuses:

```bash
mise exec -- just forum-search "mike dirnt american idiot bass rig"
mise exec -- just forum "<a thread url>"
```

**A 429 from Reddit means wait, not that the thread is empty.** Those look
identical and confusing them is the failure this project keeps having.

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
