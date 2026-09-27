# Grow a player's corpus

The download is the easy part. Most of what goes wrong is a file that arrives
cleanly and is the wrong recording.

Do all of it from the main checkout. A git worktree has its own
`resources/music/`, which git ignores and which goes when the worktree does, so
records fetched there are records nobody keeps. Nine were lost that way once, and
refetching them took longer than separating them had.

## When a corpus needs more

Four signals, and each names what to add rather than only that something is
missing:

```bash
mise exec -- go run main.go corpus music records --corpus resources/music/bass --json
mise exec -- go run main.go corpus music players --corpus resources/music/bass --json
```

- A record named in the manifest with no stems beside it. It is measured by
  nothing, and nothing else says so: the manifest looks complete.
- An entry with no `url`. The record cannot be fetched again and nobody can check
  what was measured, so it needs a link before it needs anything else.
- Fewer than three records, or all of them from one production.
- A width too wide to call a habit, with one record alone at one end. A fourth
  record says whether that one is the outlier.

Say which of these you see and name the records you would add. Download when
asked, not before.

## Choosing them

Three or four each, with the bass prominent enough to separate cleanly, and from
**different productions**. One album's tracks share a room, an engineer and a
master, so what those records have in common may be the studio rather than the
player. Spreading them is what leaves the player as the thing they share.

## Check the player played the bass on that recording

Before anything else, because no later step catches it. A song credited to a band
is not proof of who played its bass, or that a bass guitar is on it at all.

"Higher Ground" has no bass guitar. The line is a Moog, played by Stevie Wonder.
Downloaded into Flea's corpus by title, it would have measured a synthesiser as a
bassist; the record wanted there is the Chili Peppers cover. Check the credits,
and take most care with covers, where one title names two recordings.

Then check the record was made when the rig was, which is [era.md](era.md). A
record from another period measures another rig, and the figures come out looking
like the gear that was in the room instead of the gear the rig names.

## The layout is not tidiness

[resources/music/README.md](../../../../resources/music/README.md) is the rule
for what goes where: why the directory is the instrument, why a band and a genre
are fields on a record, and which two ordinary commands delete the whole corpus
without announcing it. Read it rather than inferring the shape from what is
already on disk.
