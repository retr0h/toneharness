# What a plan holds that no rig does

A rig describes a sound. A plan carries everything else the preset it came from
held, which is what makes the pair safe to exchange: read out of a preset, the
two rebuild that preset exactly with the original file gone.

**Nobody writes a plan by hand**, which is why it has no contract. A build makes
one and the preset writer reads it. Reading a preset gives two documents, because
the questions are different: what gear is this, which the ToneSpec answers, and
what is this pedal actually doing, which the plan answers.

```yaml
rig: mike-dirnt
blocks:
  - model: HD2_AmpSVBeastBrt
    dsp: 0
    pos: 1
    enabled: true
    params:
      Drive: 0.53
      Sag: 0.5
```

## Why the model is pinned

`model` is what the gear actually resolved to, and it is not belt and braces:
**661 models share only 468 names.** "Ampeg SVT" matches both channels, so a
document carrying the name alone rebuilds into a different preset.

`params` are device parameters under their own names, as distinct from the rig's
`settings`, which is the small musical vocabulary that means something anywhere.
They are here because somebody dialling `Sag` by ear is producing the one kind of
knowledge nothing else can.

**A plan stating parameters means exactly those.** No catalog defaults are added
on top, because the block was described completely. A rig naming gear with no
plan beside it takes Line 6's defaults.

`pos` is where a block sits on the device's grid, which is not always its order
in the chain. A preset can hold `block5` whose position is 6.

## The division, field by field

| field          | on   | what it holds                                           |
| -------------- | ---- | ------------------------------------------------------- |
| `chain`        | rig  | the gear, in order, with settings and evidence          |
| `sections`     | rig  | the parts of a song, as the roles that play in each     |
| `blocks`       | plan | the model, position and parameters of each              |
| `snapshots`    | plan | what each footswitch recalls: name, tempo, block states |
| `footswitches` | plan | what the pedal prints under each switch, and its colour |
| `controllers`  | plan | what an expression pedal or footswitch moves            |
| `device`       | plan | everything else, verbatim: routing, metadata            |

`snapshots`, `footswitches` and `controllers` are modelled rather than kept
verbatim, because they are decisions somebody made and might want to change.
`device` is the remainder: entries a person would not hand-edit, kept exactly as
they arrived, so a field nobody has modelled yet is not a field this drops.

Only a plan read off a device carries `device`. A rig somebody typed has no plan
beside it at all, and compiling it uses an untouched preset the device wrote.

## How the round trip is known to hold

Three assertions over every HX Stomp preset in the corpus, all of which must
pass:

1. **A preset becomes a rig and a plan, and is written back into itself**, byte
   for byte.
2. **That pair is built into an untouched preset**, byte for byte, which is the
   path they take when somebody shares them.
3. **A pair becomes a preset and is read back as a pair**, the same two
   documents.

The third catches a field these formats read but never write. Such a field
survives the first two, because the preset underneath still holds it, and
disappears in the third because the pair is all there is.
