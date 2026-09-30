# Hold the records to the era the rig claims

```bash
mise exec -- go run main.go rigs records --corpus resources/music/bass --json
```

A rig says which years its gear describes. A manifest says when each measured
record was made. Both can be honest on their own and the join between them still
wrong: a record cut before the amplifier existed measures a different rig, and the
words derived from it describe gear the rig does not name.

One rig here names an Acoustic 360, its ask names a window in the late seventies,
and its corpus once held a record cut two years before Acoustic built one. All
three statements were honest and the joins between them were wrong.

## It reports rather than refuses

Which half is wrong is a judgement, and only somebody who knows the player can
make it. The ask may name the wrong period, or the records may be the wrong
records.

- **The records are wrong** where the gear evidence is specific about a period.
  One rig cites an amplifier for one album and names the others used in other
  years, so the records from those other years are the half to replace.
- **The years are wrong** where the gear evidence covers a wider period than the
  ask claims. Widening the ask to the window the evidence actually covers is then
  the honest fix and costs nothing.

An ask stating no years reads "no era to hold them to" rather than passing
quietly, because an unstated period is not a period every record falls inside.

It also names a directory no rig answers to. The directory carrying the rig's
identifier is the only thing joining a rig to its records, so a directory spelled
any other way is a rig nobody measured, which looks identical to the ordinary
case.

## Replacing the records

This is the ordinary case rather than an exception. One player here did not pass
until the records were replaced: measured from two earlier albums while the ask
describes a later one, on amplifiers the rig's own evidence names for other years.
Measured from the right album, the mid band reads 1% where those records read 8%,
and the word earned is the opposite of the one carried before.

1. Fetch the replacements into the same directory, per
   [fetching.md](fetching.md).
2. Take the out-of-era entries out of the manifest. **Leave their audio and stems
   on disk.** Nothing here deletes a record, the file costs nothing, and anybody
   re-scoping the rig later may want it back.
3. Re-measure the whole corpus rather than this player, and update every rig whose
   figures moved. A word is earned against the others, so adding here can take one
   away from somebody nobody touched. See
   [Changing a source is never one file](../../../../CONTRIBUTING.md#changing-a-source-is-never-one-file).
4. Read the era check again, then carry the words back into the ask, per
   [words.md](words.md). If the ask has no `played`, fill it first.
