# A rig is the specification of a preset

2026-10-03

**Status: partly implemented.** A chain entry carries every control and the
device attributes beside them. The preset-wide sections are not in the contract
yet.

## What this reverses

[rigspec-as-the-one-model](2026-09-06-rigspec-as-the-one-model-design.md)
weighed three ways to carry knob values and chose the smallest:

> 1. No settings in RigSpec. Portable, but discards corpus medians.
> 2. Raw device values. Accurate, not portable, defeats the premise.
> 3. **A small musical vocabulary.**
>    `drive, bass, mid, treble, presence, level, feel`, accepted as approximate.

Option 3 shipped, with device particulars named as explicitly out:

> Device particulars (`Sag`, `Bias X`, `Ripple`, `Hum`) are *not* in RigSpec;
> the compiler sets them from catalog defaults, corpus medians, and the manual's
> directional guidance.

And it closed with "Round-tripping a preset through RigSpec is lossy by design
and must not claim otherwise."
[one-document-rig-required](2026-10-01-one-document-rig-required-design.md)
restated the trade a month later: "a rig that only works on one pedal is not a
rig — and it is a loss."

**Reversed on 2026-10-03.** Option 2's accuracy is the requirement, and its
portability objection is answered by keeping the two halves separate rather than
by throwing the values away.

## Why the original reasoning does not survive

The argument against raw values was that they are device-bound and a rig must
travel. True, and it does not follow, because a document holds both halves.

`gear` names a real-world amplifier and resolves on any Helix. `controls` names
that model's controls and answers on the devices that carry it. A reader on
other hardware uses the first and learns from `resolved` whose numbers the
second is. Nothing is refused for owning a different pedal; a control that model
does not have is refused by name, which is a better failure than a silent
substitution.

What the original decision actually cost was not portability. It was this:

**A change made on hardware had nowhere to go.** `slots export --as tonespec`
read a preset back into a rig and dropped every value, because `settings` has
seven words and a preset has 618 controls. Move a microphone, export, and the
change was gone with nothing saying so. That is the opposite of a specification
somebody can send to a friend.

**Most of the surface was unreachable.** The seven words reach seven controls.
The catalog describes 5,602. A microphone is worse than unreachable: it is an
attribute rather than a parameter, so it has no range for a word to act on, and
it was whatever the blank template happened to hold.

## What is in the contract now

On a chain entry: `controls`, a map of the model's own control names to values,
and the attributes `mic`, `distance`, `trails`, `bypass_volume`, `stereo`,
`enabled`, `dsp`, `path`, `position`, `keep_on_snapshot`, `cab`, `ir`.

On the rig: `resolved`, which records the device and catalog that produced the
numbers. Advisory. It exists so a document is not silently geared to one
machine.

Values are written as strings. A document is read through a route that turns a
float of 6.0 into the integer 6, and a device reads those as different settings,
so the literal has to survive: `"6.0"` is a float and `"6"` an integer. Three
approaches were tried first — typed maps in the generated code, yaml.v3 with
generated tags, and a second parse pass — and the first two fail on the
`json.RawMessage` passthrough fields, which yaml.v3 writes as base64.

## The ordering rule, which a test found

`controls` is applied **after** `settings`, not before. The compiler's
`setKnobs` writes over whatever is already there, so applying the words last let
`drive: 0.5` overwrite a `Drive` somebody had set by ear — the one failure the
section exists to prevent. The first implementation had it backwards and a table
row caught it.

## What was considered and rejected

**Generating the contract from the corpus.** Deriving each field's type and
range from 4,426 real presets. Rejected: a derived schema is descriptive, so the
first preset with an unseen field is refused, and it encodes accidents as rules
— LED colours are packed RGB, and `minimum: 462860` reads as a constraint while
meaning nothing. It would also put 335 generated fields in a hand-authored
contract whose `description:` fields are its grammar.

**A `built:` section beside the rig.** Built and proved first: a full preset
carried per device, with a hash of the rig to notice staleness. Rejected because
it made the rig a summary with the real answer in an appendix, and left two
places claiming to set the same control. The values belong on the chain entry.

**Enumerating every control in the schema.** 5,602 controls across 661 blocks,
changing every firmware. The catalog already enumerates them and the contract
validates against it through `x-lookup`, which is the mechanism it already uses
for footswitch colours.

## What measurement is for, since this clarifies it

Nothing in this change needed a sweep. The catalog describes every control and
its range, from HX Edit, so carrying and editing values needs no measurement at
all.

Sweeps and the hands measurements serve the *generative* half: turning "punchy"
or "more like a pick" into numbers. That is `tone tune` and the words, and it is
a separate feature from the specification. Worth stating because the two were
conflated while this was being designed.

## Deliberately not done

The preset-wide sections: global settings, the stored snapshots, footswitch
labels, controller assignments, DT and Powercab, MIDI commands, the IR table. 18
member kinds and 335 fields against the corpus.

Scoped out so the chain-level work could be tested on real hardware first. Until
they land, importing somebody else's preset keeps its chain and loses the rest,
which is the same silent loss this change fixed one level down.
