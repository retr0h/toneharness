# What a preset holds beside its chain

A chain is the signal path. A preset holds eighteen other kinds of member beside
it, and `rig.preset` is where they go.

Ask the contract for the fields, as always:
`pkg/sdk/tone/data/tonespec.openapi.yaml`, schema `PresetMember`. This page says
what the section is for and the four rules that are not obvious from the field
descriptions.

## What goes in it

Anything a preset holds that is not a block in the chain. The global settings,
the snapshots, the footswitch and expression pedal assignments, each processor's
input, output, split and join, the cabinets a dual block points at, the DT and
Powercab members, the MIDI commands, the Variax settings, the impulse response
table.

None of these is reachable by a word, and none was in the document before. They
were read off the pedal and dropped, so importing somebody's preset kept their
gear and lost what they had built with it.

Keyed by the device's own member name, because the name is what the member is.
`dsp0` and `dsp1` are two processors. `snapshot0` through `snapshot7` are eight
snapshots in order. A friendlier name would not survive the trip back.

## One shape for all eighteen

A member names a model, carries the device's own attributes under the names it
spells them with, carries controls by name, and holds members under it. Most use
two of the four.

```yaml
rig:
  preset:
    dsp0:
      members:
        outputA:
          model: HelixStomp_AppDSPFlowOutputMain
          attrs:
            "@output": "1"
          controls:
            gain: "0"
            pan: "0.5"
```

An attribute is what the device spells with a leading `@`. It says where a member
sits, whether it is on, or which of several things it is. A control is a knob,
and the catalog describes every one with its range.

Every value is a quoted literal, for the reason every control is: a document read
through `interface{}` turns 6.0 into 6, and a device reads those as different
settings. `"120.0"` is a float and `"120"` an integer.

## Four rules

**A processor's own blocks are not repeated here.** The chain states them, and
two parts of one document setting one control is what this section avoids. A
`blockN` under a footswitch or a snapshot is kept, because that is what the
switch or the snapshot does about a block rather than the block itself.

**Say `sections:` or say the snapshots, never one snapshot.** A preset's
snapshots are replaced rather than merged, so a rig naming `snapshot0` alone
leaves the device with one and more sections than that will not build. A
hand-written rig says `sections:` and lets the build make them. `rigs resolve`
writes all of them, which is consistent.

**Empty and absent are different.** `""` is the device's own empty string, which
is what an unassigned impulse response slot holds. A null value is the device
having no value at all. Both are real in the corpus and neither means the other.

**What a rig does not name is left alone.** A member the document does not
mention keeps whatever the preset underneath had, so a rig may state four members
and ignore the rest.

## Where it comes from

`rigs resolve` fills it in, and so does a lift off a device. A rig somebody typed
carries none of it, and a build then falls back to an untouched preset the device
itself wrote, which is the right answer for one that was never lifted from
anything.

Changing a value here is what reaches the pedal. That is the whole point of the
section, and it is checked: all 4,426 presets in the corpus are lifted into this
form and written back with nothing changed.

## A preset with no chain

`chain:` may be empty. A preset that makes no sound is one somebody meant rather
than one they left unfinished: the MIDI remotes that drive Spotify, Cubase and
Pro Tools from the footswitches hold no block at all, and so do the blank
templates people build from. 102 presets in the corpus are that, and every one
was refused until this section existed, because there was nothing in a rig for
them to be.

An empty chain is a statement, so the corpus adds nothing to it. A chain naming
one block is a different thing, and the convention around one is what the corpus
is for.

The key is still required. A rig with no `chain:` at all has said nothing about
its signal path; one with an empty list has said there is none.

## Before you resolve a shipped rig

`rigs resolve` with no `--out` writes over the file it read, and every comment in
that file is gone. The 32 rigs in the marketplace each carry a header saying what
is cited and what is a judgement. Resolve to `--out` and merge by hand, or
restore the header afterwards.
