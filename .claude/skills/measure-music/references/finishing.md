# Finishing the job

Adding records is not done when the files are on disk. Nothing downstream sees a
new record until the measurements are re-run and the packed file is rebuilt, and
nobody else sees any of it until a pull request carries the right half of it.

Run this whole sequence yourself. The person who asked said "add Justin
Chancellor to the bass corpus"; they did not ask to be walked through five
commands, and a step they have to remember is a step that gets skipped.

## The order, and why it is an order

```bash
# 1. The manifest first, before any audio is fetched. The record measured has to
#    be the one the evidence names, and `just record` reads the manifest to know
#    what to fetch.
#    resources/music/<instrument>/<player-slug>/corpus.yaml

# 2. Fetch by the link in the manifest, one record at a time.
mise exec -- just record resources/music/bass/<player> <track> <url>

# 3. Separate the instrument. A mix measures the band, so no figure describes the
#    player until the bass is out of it. This is the slow step: minutes a record.
mise exec -- just stems resources/music/bass/<player> \
  resources/music/bass/<player>/stems bass

# 4. Measure the tree, not the player. A word is earned by sitting clear of the
#    others, so one player measured alone earns nothing and the comparison is the
#    answer.
mise exec -- go run main.go measure players --corpus resources/music/bass
mise exec -- go run main.go measure genres  --corpus resources/music/bass

# 5. Pack what ships. Until this runs, the binary still holds the old figures and
#    `tone build` aims at them.
mise exec -- just generate     # writes pkg/sdk/audio/data/genres.json
```

Step 5 is the one that gets forgotten, and the measuring commands give no hint:
they print the new numbers while the embedded file still holds the old ones, so a
build aims where the corpus used to be.

**A test catches it now.** `data/genres.json` records a fingerprint of every
manifest it was measured over, and
`TestTheShippedFiguresNameTheCorpusTheyWereMeasuredFrom` compares that against
the manifests in the checkout. It needs no audio, so it fails in continuous
integration as well as locally, and it says what to do: run `just generate` on a
machine holding the audio and commit what it writes.

It is the only claim about the measurements anything can check. The recordings
are not in the repository, so no test can re-derive a figure; what it can say is
that the figures were taken over the records committed beside them.

**It cannot tell you a branch is behind.** A branch missing a merge has a pack
matching its own manifests, so this passes while the figures describe a record
somebody has already removed. That is a fact about the branch rather than the
tree, and only git knows it: fetch and branch from `origin/main` rather than from
a local `main` nobody has pulled. One branch cut that way measured against a stem
reading 947Hz when nothing else in the corpus reads above 264Hz, and every check
agreed with it.

## What to re-run when something changes

| what changed                    | re-run                                    |
| ------------------------------- | ----------------------------------------- |
| a record added or replaced      | 3, 4, 5 for that player, then 4 and 5 for the tree |
| a manifest edited, no new audio | 4 and 5                                   |
| genres on a record changed      | 4 and 5                                   |
| nothing but prose               | nothing                                   |

Steps 4 and 5 always go together. A measurement that is not packed is a
measurement nothing reads.

## Then check the job is actually finished

Two things to read back rather than assume:

`corpus music players` has a `rig` column. A player whose records are measured
and who has no rig in the marketplace contributes figures nothing can
act on, which [growing.md](growing.md) is blunt about: adding records is half a
job. Say which half you did.

`corpus music genres` says which genres cross the threshold. A genre under eight
records from three players is not aimable, and saying "added" without saying that
leaves somebody to find out when `tone build` refuses them.

## The pull request

Open one. The measurements are committed so nobody re-runs any of this, and that
only works if they land.

**What it carries:** the manifest, and `pkg/sdk/audio/data/genres.json`.

**What it must not carry:** the audio, and the stems. They are somebody else's
records. `.gitignore` refuses them and
[resources/README.md](../../../../resources/README.md) says neither may be
redistributed, so the measurements travel and the recordings stay on the machine
that fetched them.

Check before you push, because a `git add -A` in the wrong directory is all it
takes:

```bash
git status --short            # no .mp3, .wav, .flac, no stems/
```

Run `just ready` before committing, like any other change. The description says
which records were added, what the genre earns now, and whether any player is
still owed a rig.
