<p align="center">
  <picture>
    <source srcset="docs/assets/logo-dark.svg" media="(prefers-color-scheme: dark)">
    <source srcset="docs/assets/logo-light.svg" media="(prefers-color-scheme: light)">
    <img src="docs/assets/logo-dark.svg" alt="toneharness" width="610">
  </picture>
</p>

<p align="center">Describe a guitar or bass sound, get a Line 6 Helix preset.</p>

<p align="center">
  <a href="https://github.com/retr0h/toneharness/releases/latest"><img alt="release" src="https://img.shields.io/github/release/retr0h/toneharness.svg?style=for-the-badge"></a>
  <a href="https://codecov.io/gh/retr0h/toneharness"><img alt="codecov" src="https://img.shields.io/codecov/c/github/retr0h/toneharness?style=for-the-badge"></a>
  <a href="LICENSE"><img alt="license" src="https://img.shields.io/badge/license-MIT-brightgreen.svg?style=for-the-badge"></a>
  <a href="https://github.com/retr0h/toneharness/actions/workflows/go.yml"><img alt="build" src="https://img.shields.io/github/actions/workflow/status/retr0h/toneharness/go.yml?style=for-the-badge"></a>
  <a href="https://github.com/goreleaser"><img alt="powered by" src="https://img.shields.io/badge/powered%20by-goreleaser-green.svg?style=for-the-badge"></a>
  <a href="https://conventionalcommits.org"><img alt="conventional commits" src="https://img.shields.io/badge/Conventional%20Commits-1.0.0-yellow.svg?style=for-the-badge"></a>
  <a href="https://just.systems"><img alt="built with just" src="https://img.shields.io/badge/Built_with-Just-black?style=for-the-badge&logo=just&logoColor=white"></a>
  <img alt="github commit activity" src="https://img.shields.io/github/commit-activity/m/retr0h/toneharness?style=for-the-badge">
  <a href="https://pkg.go.dev/github.com/retr0h/toneharness"><img alt="go reference" src="https://img.shields.io/badge/go-reference-00ADD8?style=for-the-badge&logo=go&logoColor=white"></a>
  <a href="https://github.com/tekk/hovnokod-badge"><img alt="hovnokod" src="https://raw.githubusercontent.com/tekk/hovnokod-badge/main/assets/badges/hovnokod-for-the-badge.svg"></a>
</p>

<p align="center">
<b>Built for agents, there to empower humans.</b>
</p>

<p align="center">
Every block on the device was measured through it, every claim in a rig names
where it came from, and where nothing has been measured the tool says so.
Point an agent at a checkout and tell it what you want to sound like.
</p>

## What ships in the binary

| in the binary     | what it is                                                                                               |
| ----------------- | -------------------------------------------------------------------------------------------------------- |
| **661** blocks    | what an HX Stomp models, of which **224** are amplifiers and **133** cabinets. A Helix Floor is **670**. |
| **661** measured  | every one of those blocks, played and recorded on the device rather than read off a spec sheet           |
| **4,324** presets | what other people built, measured into the statistics that say where a control usually sits              |
| **15** rigs       | curated, with a citation behind every piece of gear                                                      |

## Quickstart

Start your agent in a checkout and say what you want:

- _"Make my bass sound like Dookie"_
- _"Like Mike Dirnt, but punchier"_
- _"What did Geddy Lee actually play on Hemispheres?"_
- _"I want a punk sound, find me an empty slot and let's work there"_

Ask it to put that on the pedal and it does. With one plugged in over USB the
chain starts making that sound at once, nothing stored and nothing overwritten,
or goes into a slot you name. HX Edit is not in the path and does not have to be
running.

You also get the `.hlx` file, which is what HX Edit and the plugin read if you
want it there. Building one needs no pedal at all.

**Then you play it**, because nothing here can hear. Say what is wrong in your
own words and it turns the dials, rebuilds, and records what worked so the next
session starts from there. With the pedal attached it can measure what comes
back, which is the only thing that tells you whether a change did what it meant
to.

### Words, and what happens to one nothing defines

Nothing is refused over a word. Ask for "chunky" and the preset still gets
built. Twenty-five words are defined, though, and only those move a control: ask
for one that is not and you are told so with the nearest ones that are, and no
knob turns for it.

> _"What words can I actually use, and what does each one do?"_
>
> _"I asked for chunky and nothing moved. What should I have said?"_

`punchy` is a bottom that stops with the note and a front you hear first.
`mid-forward` puts the part in front of the mix. The list is
[words.json](pkg/sdk/internal/compile/data/words.json), which is the vocabulary
itself rather than a page about it.

### Somebody nobody has researched yet

Fifteen rigs ship, each with a citation behind every piece of gear.

> _"Do you have a rig for Tim Commerford, or do you need to research it?"_
>
> _"Research Justin Chancellor's Lateralus rig and write it up with sources."_

Naming somebody who does not ship is research rather than a build, and slower.
If the evidence will not hold up, the answer you get is that it could not be
established rather than a plausible rig.

### Adding a genre, start to finish

Three genres are measured. One needs **eight records from three players** before
anything may aim at it, or it is a single band's sound wearing a genre's name.

> _"What genres are ready to aim at, and which are short of the threshold?"_

Five steps, and your agent can drive all of them.

**1. Name the records before fetching them**, so what gets measured is the
recording the evidence names. One file per player, at
`resources/music/<instrument>/<player-slug>/corpus.yaml`:

```yaml
artist: Justin Chancellor
tracks:
  - track: schism
    url: https://open.spotify.com/track/1dMFQX2BqPkR5zjC2DxFUM
    year: 2001
    band: Tool
    genres: [prog-metal]
    genres_by: llm
```

> _"Add Justin Chancellor to the bass corpus with four Lateralus tracks."_

**2. Fetch each record** by its link rather than a search, so what lands is the
recording the manifest names:

```bash
mise exec -- just record resources/music/bass/justin-chancellor schism \
  https://open.spotify.com/track/1dMFQX2BqPkR5zjC2DxFUM
```

**3. Cut the bass out of the mix.** A mix measures the band, so no figure
describes the player until the instrument is separated:

```bash
mise exec -- just stems resources/music/bass/justin-chancellor \
  resources/music/bass/justin-chancellor/stems bass
```

**4. Measure.** A word is earned by sitting clear of the other players, so this
compares the whole tree rather than profiling one person:

```bash
mise exec -- go run main.go measure players --corpus resources/music/bass
mise exec -- go run main.go measure genres  --corpus resources/music/bass
```

**5. Regenerate what ships**, which rolls the measurements into the file the
binary embeds:

```bash
mise exec -- just generate     # writes pkg/sdk/audio/data/genres.json
```

Re-measure only when the records or the stems change. The numbers are committed,
so nobody else re-runs any of this.

**What a pull request carries, and what it must not.** The manifest and the
regenerated `genres.json`. **Not the audio and not the stems**: those are
somebody else's records, `.gitignore` refuses them, and
[resources/README.md](resources/README.md) says neither may be redistributed.
The measurements travel, the recordings stay on your disk.

> _"I've added the records and run the measurements. Open the PR."_

Adding a player to a genre that already exists is the same five steps. A genre
crosses the threshold on its own once enough players sit behind it.

## Skills

The CLI is for an agent more than for a person, so the way to use toneharness is
to point an agent at a checkout and say what you want to sound like. Each
skill's own README says how to install and use it.

| Skill                                                         | Answers                                                                          |
| ------------------------------------------------------------- | -------------------------------------------------------------------------------- |
| [build-a-rig](.claude/skills/build-a-rig/README.md)           | _"Make my bass sound like Dookie"_, without guessing at the gear                 |
| [write-a-spec](.claude/skills/write-a-spec/README.md)         | Which document a fact goes in, and why a field was refused                       |
| [measure-music](.claude/skills/measure-music/README.md)       | What a player's records actually sound like, and which words the figures earn    |
| [work-a-device](.claude/skills/work-a-device/README.md)       | What is on the pedal, getting a preset onto it, and moving slots around          |
| [measure-a-device](.claude/skills/measure-a-device/README.md) | What a control actually does, by pushing a known signal through it and listening |

Each follows the [Agent Skills] format: a slim `SKILL.md` that routes, with the
detail in reference files an agent reads only when the question calls for them.
None of them writes down a list the tool can print. A list in a skill is right
the day it is written and wrong after the next change, with nothing marking the
moment.

## Install

```bash
curl -fsSL https://github.com/retr0h/toneharness/raw/main/install.sh | bash
```

`toneharness <command> --help` is the command reference, and no page here
duplicates it. From a checkout it is `go run main.go --help`, which compiles the
tree and answers from the source rather than from a description of it.

Installs to `~/.local/bin` or `/usr/local/bin`, verifying SHA256 checksums.
Override with `TONEHARNESS_INSTALL_DIR=/some/path`, or pin a version with
`TONEHARNESS_VERSION=1.1.1`.

<details>
<summary>Other ways</summary>

```bash
go install github.com/retr0h/toneharness@latest
```

Released binaries reach a Helix over USB on macOS. On Linux they build, validate
and write presets, and the device commands say they are not supported yet.

</details>

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for prerequisites, setup, conventions and
the pull request workflow.

## License

The MIT License, see [LICENSE](LICENSE).

[agent skills]: https://agentskills.io
