# Examples

One of each document, with the fields filled in and commented. These are
teaching documents rather than rigs anybody plays, so they sit beside
[../artists/](../artists/) rather than in it: the loader reads `artists/` and
nothing else, so nothing here is listed by `rigs list` or packed into the
binary.

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

| file                                                             | shows                                                                              |
| ---------------------------------------------------------------- | ---------------------------------------------------------------------------------- |
| [tonespec/chunky-punk.yaml](tonespec/chunky-punk.yaml)           | an ask in nothing but words: a genre, a technique, and what it should sound like   |
| [tonespec/like-a-player.yaml](tonespec/like-a-player.yaml)       | naming somebody, which resolves to the cited rig researched for them               |
| [tonespec/like-a-record.yaml](tonespec/like-a-record.yaml)       | pointing at a recording, which is the kind that resolves fully                     |
| [tonespec/corrected-by-ear.yaml](tonespec/corrected-by-ear.yaml) | an ask that was built, heard, and asked again                                      |
| [tonespec/mike-dirnt.yaml](tonespec/mike-dirnt.yaml)             | every part of the ask on one subject. Most asks are a tenth of this                |
| [tonespec/my-setup.yaml](tonespec/my-setup.yaml)                 | a Setup: what somebody has, which changes on a different clock from what they want |
| [rigspec/mike-dirnt.yaml](rigspec/mike-dirnt.yaml)               | the rig that answers the ask beside it, with a source per claim                    |
| [rigspec/dir-angl-meteor.yaml](rigspec/dir-angl-meteor.yaml)     | a rig read back off a device rather than researched                                |
| [plan/dir-angl-meteor.yaml](plan/dir-angl-meteor.yaml)           | the device half of that same preset, committed exactly as the HX Stomp wrote it    |

## Running one

```bash
mise exec -- go run main.go tone build \
  --ask marketplace/core/examples/tonespec/chunky-punk.yaml --out rig.yaml
mise exec -- go run main.go presets make --rig rig.yaml \
  --ask marketplace/core/examples/tonespec/chunky-punk.yaml --out punk.hlx
```

`--setup` is optional; without one the rig is for a bass on an HX Stomp.

`presets make` rather than `presets compile`, because the words are still
unspent after `tone build`: a rig carries the gear and the ask carries the
sound. Compile is for a document that has already made those decisions, such as
a rig lifted off a device.

**Words cannot start a chain.** An ask carrying only a genre and adjectives is
refused: there is nothing to pick an amplifier from. Three things start one:

|                            |                                                          |
| -------------------------- | -------------------------------------------------------- |
| `gear:`                    | name it yourself, and the catalog resolves it            |
| `like: { artist: ... }`    | the rig somebody researched for them, with its citations |
| `like: { recording: ... }` | measured, and the nearest of 224 amplifiers chosen       |

The words adjust what that gives you rather than producing it. `subject:` is not
one of the three: it says who the ask is for, and `like:` says what to aim at.

[write-a-spec](../../../.claude/skills/write-a-spec/SKILL.md) owns every field
on both contracts and says which document a fact belongs in. The contracts
themselves are
[tonespec.openapi.yaml](../../../pkg/sdk/tone/data/tonespec.openapi.yaml) and
[rigspec.openapi.yaml](../../../pkg/sdk/rig/data/rigspec.openapi.yaml), and a
document that violates either is refused at load rather than half-read.
