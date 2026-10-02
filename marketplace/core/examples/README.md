# Examples

One of each document, with the fields filled in and commented.

The suffix says which kind each one is, so a pair sits together:
`mike-dirnt.tone.yaml` is the ask and `mike-dirnt.rig.yaml` is the gear that
answered it, and `dir-angl-meteor` has both a rig and the plan it compiled to.
There are no `tonespec/` and `rigspec/` directories any more, because a filename
that says what it holds does the same job in less. These are teaching documents
rather than rigs anybody plays, so they sit beside [../artists/](../artists/)
rather than in it: the loader reads `artists/` and nothing else, so nothing here
is listed by `rigs list` or packed into the binary.

The tests read these. A field added to either contract without an example here
fails the build, which is what keeps them from going stale.

## Four documents, and a person writes two

| document     | what it is                                                                                                   | who writes it                |
| ------------ | ------------------------------------------------------------------------------------------------------------ | ---------------------------- |
| **ToneSpec** | what you want: a genre, how it is played, how it should sound, gear you insist on                            | you                          |
| **Setup**    | what you own: the pedal, the instruments, the strings, what it plays into                                    | you, once                    |
| **RigSpec**  | real-world gear in signal order with a source for every claim. Portable to any Helix                         | research, or `tone build`    |
| **Plan**     | the Line 6 layer: which model each piece resolved to, every parameter by its device name, routing, snapshots | nothing. It is machine state |

Nobody authors a Plan. No command writes one for you to keep. It is here because
reading a real one is the clearest way to see what the device actually stores,
and because `presets compile --plan` takes one back.

## The files

| file                                                     | shows                                                                              |
| -------------------------------------------------------- | ---------------------------------------------------------------------------------- |
| [chunky-punk.tone.yaml](chunky-punk.tone.yaml)           | an ask in nothing but words: a genre, a technique, and what it should sound like   |
| [like-a-player.tone.yaml](like-a-player.tone.yaml)       | naming somebody, which resolves to the cited rig researched for them               |
| [like-a-record.tone.yaml](like-a-record.tone.yaml)       | pointing at a recording, which is the kind that resolves fully                     |
| [corrected-by-ear.tone.yaml](corrected-by-ear.tone.yaml) | an ask that was built, heard, and asked again                                      |
| [mike-dirnt.tone.yaml](mike-dirnt.tone.yaml)             | every part of the ask on one subject. Most asks are a tenth of this                |
| [mike-dirnt.rig.yaml](mike-dirnt.rig.yaml)               | the rig that answers the ask beside it, with a source per claim                    |
| [mine.setup.yaml](mine.setup.yaml)                       | a Setup: what somebody has, which changes on a different clock from what they want |
| [dir-angl-meteor.rig.yaml](dir-angl-meteor.rig.yaml)     | a rig read back off a device rather than researched                                |
| [dir-angl-meteor.plan.yaml](dir-angl-meteor.plan.yaml)   | the device half of that same preset, committed exactly as the HX Stomp wrote it    |

## Running one

```bash
mise exec -- go run main.go tone build \
  --ask marketplace/core/examples/chunky-punk.tone.yaml --out rig.yaml
mise exec -- go run main.go presets make --rig rig.yaml \
  --ask marketplace/core/examples/chunky-punk.tone.yaml --out punk.hlx
```

`--setup` is optional; without one the rig is for a bass on an HX Stomp.

`presets make` rather than `presets compile`, because the words are still
unspent after `tone build`: a rig carries the gear and the ask carries the
sound. Compile is for a document that has already made those decisions, such as
a rig lifted off a device.

**Words still do not choose gear.** Four things do, and a word adjusts what they
give you rather than producing it:

|                            |                                                           |
| -------------------------- | --------------------------------------------------------- |
| `gear:`                    | name it yourself, and the catalog resolves it             |
| `like: { artist: ... }`    | the rig somebody researched for them, with its citations  |
| `like: { recording: ... }` | measured, and the nearest of the instrument's amps chosen |
| `genre:`                   | measured too, from the records the corpus tags with it    |

The last one is why an ask of a genre and three adjectives builds. It used to be
refused, and the reason it can work now is that a genre is read as a
*displacement* from the records of players who hold none of it, rather than as a
position: those two scales do not subtract, because a genre is measured off
finished records and a block off a dry signal pushed through it.
[build-a-rig's measuring.md](../../../.claude/skills/build-a-rig/references/measuring.md)
owns the arithmetic and the two cases it refuses.

A genre the corpus has not measured names nothing, so an ask of adjectives and
`genre: [rock]` is still refused: `corpus music genres` says which have enough
records to mean anything.

`subject:` is not one of the four. It says who the ask is for, and `like:` says
what to aim at.

[write-a-spec](../../../.claude/skills/write-a-spec/SKILL.md) owns every field
on both contracts and says which document a fact belongs in. The contracts
themselves are
[tonespec.openapi.yaml](../../../pkg/sdk/tone/data/tonespec.openapi.yaml) and
[rigspec.openapi.yaml](../../../pkg/sdk/rig/data/rigspec.openapi.yaml), and a
document that violates either is refused at load rather than half-read.
