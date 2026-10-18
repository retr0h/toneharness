# Writing the rig down, and shipping it

The research is the hard half and it is not the whole job. A rig nobody committed
helps the person in front of you once; a rig in the marketplace is there for
everybody.

Run this yourself. The person asked for a Justin Chancellor rig, not to be told
which command scaffolds one.

## Which tier it goes in, and which directory

[marketplace/](../../../../marketplace/README.md) has two tiers, and the
difference is what the research reached.

`community/` unless every claim has a source somebody opened. That is the normal
answer for a rig written today.

`core/` when it does, because core ships in the binary and a wrong claim there
reaches everybody who installs this. A rig that starts in community and gets
sourced later moves across.

Inside either tier, the directory is what the subject is:

| the subject                                      | directory  |
| ------------------------------------------------ | ---------- |
| a player, a band, an era                         | `artists/` |
| a genre, with nobody named                       | `genres/`  |
| a sound, where the gear could not be identified  | `sounds/`  |

A sound goes in the third because it is attributed to nobody: it is what a set
of records measures as, resolved to gear, written where the research into who
played what bottomed out. See
[evidence.md](evidence.md), "Where the gear cannot be identified, offer a
sound-alike", which also says when not to write one.

**A person is never filed under a genre.** Most play several, the ask's `genre:`
already names which, and a directory would have to pick one and be wrong about
the rest. The reverse holds too: a genre rig names no player, so it does not
belong beside people who can be cited.

A genre rig is built rather than researched, because there is nobody to ask. Let
`tone build` rank the device's amplifiers against what the genre measures as and
take the nearest, then record it as `kind: audio` evidence naming how many
records and players it rests on. It also moves as the corpus grows, so say so in
the caveat rather than presenting it as settled.

**Never write a rig into `pkg/sdk/shipped/`.** It is a generated copy of
`marketplace/core/`, so a rig put there is deleted by the next `just generate` and
the work is gone with no error. After changing anything in `core/`, run
`mise exec -- just generate` and commit both, which is what packs it into the
binary.

## Two bars for core, and where each one falls

`core/` ships in the binary, so a rig there reaches everybody who installs this.
Two things have to be true.

**Bar one: every gear claim has a source somebody opened.** No exceptions. `llm`
alone means a model asserted the gear and nobody confirmed it, which is the same
thing as made up. An audit of 122 gear claims across the marketplace found six
like that on four rigs: an Ampeg SVT for Charlie Hayward with nothing behind it,
the same for Tommy Caldwell and Cone McCaslin, and a Marshall stack for Dusty
Hill that Premier Guitar's ZZ Top gallery and his obituary were both opened
looking for and neither carries.

**Bar two: the amplifier resolves to the amplifier named.** A `gear:` name
resolves through the gear map to the nearest model when the real thing is not
emulated, which is right for somebody building a sound and wrong to ship as a
citation. Berry Oakley's Fender PS400 resolving to an Ampeg SVT is not a Berry
Oakley rig, it is an Ampeg SVT with his name on it. The amplifier is the block
that decides the sound, so this bar is drawn there.

**A cabinet or a pedal may substitute, where the claim is cited and the caveat
says so.** This device models fourteen bass amplifiers and fewer cabinets, and
the best-sourced claims in this marketplace are cabinet claims the device cannot
honour: Geddy Lee's two Ampeg V-4B cabinets are cited from Canadian Musician in
1979 and 1981 and Guitar Player in 1980, three print sources in his own words,
and nothing here models one. Deleting that to satisfy a rule aimed at invented
gear would throw away the research and leave the rig no more accurate. Name the
substitution in the caveat and keep it.

**A source that hedges is still a source.** Somebody saying they think they use
an Ampeg is evidence about an Ampeg, weak and honestly labelled, which is a
different thing from a model inventing one. That rig goes in `community/` with
the hedge quoted rather than smoothed out, because "I think" is not a settled
claim and is not nothing either.

Check both bars before filing anything in core:

```bash
# every gear claim and the evidence under it
mise exec -- go run main.go rigs show --id <id> --json

# what each name resolved to, which is the only place a substitution shows
mise exec -- go run main.go rigs resolve --id <id> --json
```

A substitution is invisible in the document. The file says `Fender 400 PS` and
reads as settled; only the resolve says an Ampeg came out.

## Do not replace a player to keep a genre alive

A genre needs eight records from three players before anything may aim at it, so
moving one rig out of core can drop a genre under the line. That is not a reason
to keep a rig whose gear nobody sourced, and not a reason to swap in whoever
happens to map.

The order:

1. **Research deeper first.** A famous player with unsourced gear in the file is
   usually a research failure rather than a bad choice of player. Klaus
   Flouride's amplifier sat on `llm` while the answer was a MusicRadar interview
   with him, cited in the footnote under the Wikipedia entry nobody had
   followed. Wikipedia is never the source and its references are.
2. **Keep the player and move the file** where the gear is real and cited and
   this device models no amplifier like it. The research is good and only the
   mapping failed, so it goes to `community/` with the substitution named.
   Dropping the player throws away the one part that was right.
3. **Replace only with somebody who clears both bars**, before a single record is
   downloaded.
4. **Let a genre go dead rather than prop it up.** A genre nothing can aim at
   reports that it is short and by how much. A genre carried by a player whose
   amplifier was invented reports that it is ready, and every rig built from it
   inherits the invention.

Nothing is deleted to reach a threshold and nothing is deleted to satisfy this
page. A rig moves tier, and the research moves with it. Somebody holding the
pedal can override any of it by hand; what ships by default has to be checkable.

## Scaffold rather than write from scratch

```bash
mise exec -- go run main.go rigs new --help
mise exec -- go run main.go rigs new --dir marketplace/community \
  --id justin-chancellor --amp "Mesa Boogie Dual Rectifier" \
  --band Tool --genre prog-metal
```

`--dir` is the tier, and it is the flag to get right: without it the scaffold
lands wherever a rig of yours would go rather than in the marketplace. It writes
one document, the ask and the rig that answers it, with the fields and the
comments already there, and appends `artists/` itself. Writing one by hand means
rediscovering which fields exist, and the contract is the thing that knows.

The two halves are separate on purpose: the rig is the gear, the ask is what
somebody wanted of it. [write-a-spec](../../write-a-spec/SKILL.md) owns every
field and which half a fact belongs in. Read it rather than guessing from a
neighbouring file.

## Before you commit it

Build it. A rig that does not resolve is not a rig yet:

```bash
mise exec -- go run main.go presets make --id <slug> \
  --rigs marketplace/community --out /tmp/check.hlx
```

Read what it printed. Every block it added that the rig did not name, every word
that moved nothing, every substitution it made. Those are decisions taken on the
player's behalf and this is the moment to disagree with them.

`rigs records` says whether the records behind the rig are from the era it
claims. A rig for *Lateralus* measured against records from two albums later
describes a different sound, which [era.md](../../measure-music/references/era.md)
owns.

## Resolve it, so the file says what the preset holds

A committed rig carries every control the chain ends up with, and a test refuses
one that does not:

```bash
mise exec -- go run main.go rigs resolve --dir marketplace/core --id <slug>
```

A researched rig names an amplifier and perhaps a cabinet. Resolving it writes
the compressor the corpus adds, the cabinet the amplifier is voiced with, and the
dozen controls each of those arrives at, so the document describes the preset
rather than the research. Both tiers were half resolved and half not, nineteen
against twelve in core alone, with nothing marking which, and a reader could not
tell a rig nobody had resolved from one whose gear this device cannot do.

**Write over the file rather than to `--out`.** The prose at the top of a file is
put back by a rewrite; `--out` to a new path has no file to read it from, so the
header is simply gone. A comment lower down is refused rather than dropped, by
name and line, because a rewrite cannot know where it belongs: move what it says
into the header or into an `evidence` note, where it travels with the document.

**A rig that cannot resolve carries no controls at all.** Those are the two
states, and the test checks both directions. Dusty Hill's rig cited a Marshall
JMP-1, which no bass amplifier on this device emulates, and it carried thirteen
controls with `BrtDrive` and `NrmDrive` among them: a jumped Marshall Super Lead,
a guitar amplifier, resolved into a bass rig before the shortlist was held to the
instrument. Taking those controls out was the fix. Choosing a stand-in nobody has
evidence for would not have been.

## When it cannot be established

Stop and say so, which [vague.md](vague.md) states plainly: do not ship a
half-known rig as though it were known. Specifically, do not commit one.

A rig of `kind: llm` assertions is worse than no rig, because it looks like
knowledge and the confidence field is the only thing saying otherwise. What to
hand back instead is what was searched, what was found, and which piece of gear
the research could not reach. That is a useful answer. A plausible rig is not.

The honest middle is a rig with a real citation for the amplifier and nothing for
the cabinet, committed with `confidence: low` and a `caveat` saying which half is
guessed. Say that out loud rather than letting the fields imply it.

## The pull request

```bash
mise exec -- just ready
```

Then open it. The description says who the rig is for, which album or era, what
the sources are, and which claims are weaker than the others. A reviewer should
not have to open the YAML to learn how well sourced it is.

[CONTRIBUTING's Sourcing a rig](../../../../CONTRIBUTING.md#sourcing-a-rig) is the
bar, and it says what a pull request has to have finished before it is opened.
Read it before starting rather than after.
