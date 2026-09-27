# Measure a genre

Two different questions, and only one of them needs audio:

```bash
mise exec -- go run main.go corpus music genres --corpus resources/music/bass --json
mise exec -- go run main.go measure genres --corpus resources/music/bass --json
```

One instrument, the same as `measure players`. A genre is measured against the
players who do not play it, and a guitar's centre of gravity sits an octave above
a bass guitar's, so pointing this at the whole tree would compare bass genres
against guitarists and earn every one of them `dark`.

The first counts what somebody wrote into the manifests, which is a file read. The
second measures the recordings and takes minutes per record. What each genre
measured as is packed into the binary by `mise exec -- just generate`, so asking
for a genre afterwards costs no audio.

## A genre is measured by a different rule than a player

A player earns a word where their whole spread sits outside the middle half of the
others, which suits three records by one person. A genre pools several players, so
its spread runs two to three times wider, 64 to 92Hz of centroid against 17 to
41Hz for a player, and nothing that wide can sit clear of anything. So a genre is
measured by its **middle** against the others' middle half, carrying the margin,
and the others are the players who play none of it.

Finding that out cost a rewrite. Applying the player rule to a genre is not
stricter, it is unanswerable: it returns `nothing` for everything, which reads as
a corpus problem and is a rule problem.

## Clearing the threshold is not sounding like anything

Eight records from at least three players before anything may aim at a genre.
Under that, three records by one band is that band's sound wearing a genre's name,
and a request for the genre would get the band. The figures cannot tell those
apart, so the count is the only thing that can.

Over it, a genre can still earn nothing, and that is worth reporting rather than
worth fixing. The genre here with the most records and the most players sets
nothing apart, sitting inside the middle half of everything else on every axis. So
clearing the threshold says a genre has enough behind it, not that it sounds like
anything in particular.

Read the margin the way a player's is read. A word a hundredth past its line is
the kind one more player measured could take away, and saying so is more useful
than reporting the word alone.

## What to download next

A genre one record short of the threshold wants one record. A genre with eleven
records from one band wants a second and a third player, not a fourth album. The
player count is the half that is usually short, and it is the half a fourth album
cannot fix.

A record naming no genre is invisible to every genre, so it counts towards none of
them. `corpus music players` shows who has untagged records.

## The checked column is the other half

A genre can reach the threshold entirely on tags a model guessed, which still
reaches it, and somebody aiming at it should know nobody has looked. That is what
`genres_by` is for; see [fetching.md](fetching.md).
