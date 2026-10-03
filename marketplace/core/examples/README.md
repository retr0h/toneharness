# Examples

One of each document, with the fields filled in and commented.

A ToneSpec is one file holding both halves: `ask:` is what somebody wanted and
`rig:` is the gear that answered. The other two kinds carry a suffix, so a
filename says which it is: `mine.setup.yaml` is what somebody owns and
`dir-angl-meteor.plan.yaml` is the device half of the preset
`dir-angl-meteor.yaml` compiles to. These are teaching documents rather than
rigs anybody plays, so they sit beside [../artists/](../artists/) rather than in
it: the loader reads `artists/` and nothing else, so nothing here is listed by
`rigs list` or packed into the binary.

The tests read these. A field added to the contract without an example here
fails the build, which is what keeps them from going stale.

## Three documents, and a person writes one

| document     | what it is                                                                                                   | who writes it                |
| ------------ | ------------------------------------------------------------------------------------------------------------ | ---------------------------- |
| **ToneSpec** | `ask:` what you want, `rig:` the gear that answers it with a source per claim. The rig is required           | you, or `tone build`         |
| **Setup**    | what you own: the pedal, the instruments, the strings, what it plays into                                    | you, once                    |
| **Plan**     | the Line 6 layer: which model each piece resolved to, every parameter by its device name, routing, snapshots | nothing. It is machine state |

The ask and the rig were two files until version 2 of the contract. They were
one-to-one in every case that existed, and an ask with no rig was already
refused, so the second file was an annotation kept in step by hand.

Nobody authors a Plan. No command writes one for you to keep. It is here because
reading a real one is the clearest way to see what the device actually stores,
and because `presets compile --plan` takes one back.

## The files

| file                                                   | shows                                                                              |
| ------------------------------------------------------ | ---------------------------------------------------------------------------------- |
| [chunky-punk.yaml](chunky-punk.yaml)                   | an ask in nothing but words: a genre, a technique, and what it should sound like   |
| [like-a-player.yaml](like-a-player.yaml)               | naming somebody, which resolves to the cited rig researched for them               |
| [like-a-record.yaml](like-a-record.yaml)               | pointing at a recording, which is the kind that resolves fully                     |
| [corrected-by-ear.yaml](corrected-by-ear.yaml)         | an ask that was built, heard, and asked again                                      |
| [mike-dirnt.yaml](mike-dirnt.yaml)                     | every part of both halves on one subject. Most documents are a tenth of this       |
| [mine.setup.yaml](mine.setup.yaml)                     | a Setup: what somebody has, which changes on a different clock from what they want |
| [dir-angl-meteor.yaml](dir-angl-meteor.yaml)           | gear read back off a device rather than researched, so it carries no ask           |
| [dir-angl-meteor.plan.yaml](dir-angl-meteor.plan.yaml) | the device half of that same preset, committed exactly as the HX Stomp wrote it    |

## Running one

```bash
mise exec -- go run main.go tone build \
  --ask marketplace/core/examples/chunky-punk.yaml --out rig.yaml
mise exec -- go run main.go presets make --rig rig.yaml --out punk.hlx
```

`--setup` is optional; without one the rig is for a bass on an HX Stomp.

`presets make` rather than `presets compile`, because the words are still
unspent after `tone build`: the ask comes back out of the build beside the gear
it resolved to, and `make` is what reads it. Compile is for a document whose
decisions are already made, such as gear lifted off a device.

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
`genre: [rock]` is still refused. So is one the corpus has measured and has too
little of: eight records from three players, or the figures are one band's sound
wearing a genre's name. `corpus music genres` says which clear it, and an ask
may name several in the order to try them, which is why `chunky-punk.yaml` asks
for punk and falls to pop-punk.

`subject:` is not one of the four. It says who the ask is for, and `like:` says
what to aim at.

[write-a-spec](../../../.claude/skills/write-a-spec/SKILL.md) owns every field
on the contract and says which half of the document a fact belongs in. The
contract itself is
[tonespec.openapi.yaml](../../../pkg/sdk/tone/data/tonespec.openapi.yaml), and a
document that violates it is refused at load rather than half-read.
