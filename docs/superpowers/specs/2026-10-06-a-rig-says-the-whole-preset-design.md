# A rig says the whole preset

**2026-10-06**

A dated record of what was decided, not documentation. How `rig.preset` works
today is
[write-a-spec's preset.md](../../../.claude/skills/write-a-spec/references/preset.md).

## The problem

A rigspec said the signal path and every control on it. A preset holds eighteen
other kinds of member beside the chain, and the document had nowhere to put any
of them. Those members were read off the pedal and dropped, so importing
somebody else's preset kept their gear and lost the snapshots, the footswitch
assignments, each processor's routing, the noise gate on the input, the
microphone distance on a paired cabinet, and the output's destination.

Measured against the preset corpus: 18 member kinds under `data.tone`, 335
distinct field names, nesting five deep at `snapshot0.controllers.dsp0.block1`.

The facts were not entirely lost. `plan.Plan.Device` has carried them since
before this, as `map[string]json.RawMessage` on `DeviceState.tone` and
`.routing`. Two things were wrong with that as an answer. They are opaque blobs,
so nobody can open a document and change a snapshot's name. And they hang off
the plan, which the compiler writes and no person edits, rather than off the
rig, which is the redistributable document the whole project is pointed at.

## What was decided

**One generic member type rather than eighteen.** `PresetMember` has a model,
attributes, controls and members under it, and recurses. A member is the same
shape at every level because the device uses one shape for all of them.

**Not hand-written types per kind.** Every type inferred from reading a sample
of the corpus has been wrong at least once, and each wrong one refused hundreds
of real presets: `@tempo` is fractional in 3,155 snapshots, `@topology1` is a
string in 506 presets and a number elsewhere, `@dt_reverb` is a boolean in some
and a number in others, and a cabinet spells one control `HighCut` in 1,994
presets and `High Cut` in 113.

**Not generated from the corpus either.** That describes what has been seen
rather than what is allowed. It would refuse the first field Line 6 add, and it
would publish a packed-RGB footswitch colour of 462,860 as the bottom of a
range.

**`attrs` apart from `controls`.** The device spells its own attributes with a
leading `@`. An attribute says where a member sits, whether it is on, or which
of several things it is; no word reaches one and the catalog carries no range
for one. A control is a knob and the catalog describes all 5,602 of them.

**A value type that allows the empty string.** `catalog.Setting` refuses one on
purpose, because a control with no value is not a reading. That argument is
about a control somebody is setting, and this is a transcription of what a
device already holds: an unassigned impulse response slot is the empty string,
and 128 slots hold one in every preset carrying a table. `catalog.Held` is
Setting with that state added.

**A nil pointer for the device's own null, rather than a third state on
`Held`.** `@cursor_path` is null in three presets. One spelling per state is
what stops a writer choosing between two that mean the same.

**Merged on the way back, not replacing.** A lifted plan names every member the
preset had, so replacing is right for it. A rig's `preset:` names only what
somebody wrote down. Replacing on that deleted the snapshots a rig did not
mention, which showed up as a rig stating one output leaving the device with
none and two sections that no longer fit.

**The chain's blocks stay out of it.** A processor's `blockN` is a chain entry.
A `blockN` under a footswitch or a snapshot is kept, because that is what the
switch or snapshot does about a block rather than the block itself.

## What was rejected

**A `built:` section beside the rig.** It makes the rig a summary with the
answer in an appendix, and two places claiming to set one control.

**Filling `plan.Device` from `preset:` as well.** Written first, then deleted.
It compiled the same facts into a second representation, which is the
duplication this was meant to remove. `plan.Device` is now filled by a lift and
nothing else, which is what it did before.

**Replacing `DeviceState.tone` and `.routing` outright.** The right end state
and not this change. `pkg/cli/headroom.go` rewrites the output block's routing
through `DeviceState.Routing` keyed `dsp0.outputA`, and that is the signal path:
the thing this project has got wrong most often and the thing I cannot verify
without the pedal and the lead. Left alone deliberately.

## What it is checked against

4,324 of the 4,426 presets in the corpus are lifted into this form and written
back, and every field that is not a chain block comes back unchanged. The 102
that do not lift fail `preset.Read` or `Lift` for reasons that predate this.

Resolving a rig and building from the resolved document produces a preset
identical to the one built from the rig it came from. Changing three values by
hand in the document, a noise gate threshold, an output's destination and a
split's balance, changes exactly those three in the built preset and nothing
else.
