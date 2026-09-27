# A rig is a plan, for one device

2026-09-19

## Two changes, one edit

A RigSpec is doing two jobs it should not be doing, and both are moves of the
same fields. Doing them apart means moving each field twice.

It carries what a person wrote, and that belongs in the ask: who the rig is for,
why each claim is believed, what was corrected and when. It also carries Line
6's own vocabulary next to gear named the way a musician names it, so a document
that could describe any modeller is welded to one.

John settled both on 2026-09-19. A ToneSpec never names a knob position, and the
device-bound half comes off the RigSpec into a plan with a driver behind it.

## What a rig is after this

Three layers, which is what has been said all along and has not been true:

|              | holds                                        | written by |
| ------------ | -------------------------------------------- | ---------- |
| **ToneSpec** | what somebody means, and why it is believed  | a person   |
| **RigSpec**  | gear in signal order, named as a person does | the tool   |
| **Plan**     | that rig realised on one device              | a driver   |

The middle one is the portable layer nobody had. It is what survives moving to
another modeller: roles in order, gear by its real-world name, the instrument it
is for. A Plan is that rig fitted to hardware: which model each piece of gear
resolved to, where it sits in the DSP, what the footswitches do.

### The names, corrected 2026-09-26

An earlier draft of this table called the middle layer a Chain and gave the
device-bound one no suffix. Both were wrong, and the reason is worth keeping
because the draft read fine until somebody went to write the code.

`pkg/sdk/chain` already existed, and it is the device-bound one: it holds Helix
model identifiers and Helix parameter keys. So the draft's names had the word
Chain meaning the portable document in the contract and the device-bound one in
the tree, which is the confusion this record exists to remove rather than a
place to add one.

John settled it on 2026-09-26. **RigSpec keeps its name and becomes the portable
layer**, because a rig is what a player calls the gear they play through and
naming that document after the pedal would be the opposite of what the word
means. The Helix half becomes a **Plan**, and `pkg/sdk/chain` is renamed to
`pkg/sdk/plan` to match.

It is also the cheapest of the three options. Nothing that currently says "rig"
has to stop saying it, so #136 removes the confusion between recipe and RigSpec
instead of moving it to a new pair of words.

## Which fields move, and why

### To the ToneSpec

`subject`, `aliases`, `default`, `extends`, `character` (as `words`),
`technique`, `played`, `confidence` and `corrections` (was `mutations`).

These are what a person wrote and why. Counting what reads them was instructive
and is not the argument: `aliases` is how a rig is found by another name,
`confidence` is rendered, `extends` resolves a parent, `played` reaches the
backing listing. All of that is bookkeeping about an ask, and it is bookkeeping
the ask should be doing.

Most of them are already there. ToneSpec was written with `subject`, `aliases`,
`default`, `confidence`, `played`, `technique` and `evidence` on it, so the
contract work is done and what remains is taking them off RigSpec and moving the
readers. `extends` and `corrections` are the two ToneSpec did not carry, and
each is added when something reads it rather than before, which is the rule
`requires` broke. `corrections` is there now, with its reader.

### The top-level evidence, corrected 2026-09-26

An earlier draft of the list above ended "and the document's own top-level
`evidence`". It does not move, and the reason is the same one that kept the
per-entry evidence here.

A document-level citation is the one that covers the whole chain at once: a rig
rundown naming a player's entire setup in one article. Moving it to the ask
would put that citation on one document while the four claims it supports sat on
another, and would leave a published rig unable to say why its gear is what it
is. That is the #144 argument John settled on 2026-09-19 for the per-entry
evidence, and it does not weaken as the claim gets wider.

So the division is by scope rather than by document. Why the request was made is
the ask's; why this gear answered it is the rig's. A ToneSpec keeps an
`evidence` of its own for the first kind, which makes this the same pattern as
`instrument`: one field name, three layers, a different claim on each.

Nothing moved either way in practice. None of the nine shipped rigs carried a
document-level citation, so this was a decision about the contract rather than a
migration of anything.

`character` and `technique` are the two that also reach the compiler, in
`compile/move.go` and `compile/demand.go`. They move with the rest and the
compiler stops reading them, because resolving a word into a knob position is
the translation step's job and doing it twice is how the two answers drift. This
is the same argument the measurement code already lost once, in two languages.

They do not travel inside the rig and they are not resolved before it. They are
passed beside it, as a `compile.Intent` carrying the words, the attack and the
name. The section below says why resolution cannot move earlier than the
compiler.

### A word keeps its evidence, corrected 2026-09-26

`character` becomes `words` on the ask, and the rename was nearly a silent
regression.

A character term was an object, `{term, evidence}`. ToneSpec's `words` was a
bare list of strings, so the obvious move was to flatten each term to its name.
That flattening removes the weighting. `weightOf` sizes how far a word moves a
control from how far that word's own measurement sits from every other rig's, so
a word nobody measured moves a control by a full step and a word measured close
to the pack moves it barely at all. With the evidence gone, every word moves the
same amount whether somebody listened or a model guessed, which is the guessing
this project exists to remove.

It nearly shipped that way, as an interim with the weighting deleted and a note
to re-attach it later. It is not an interim worth having: a build that looks
right and is weighted wrong is worse than one that does not build.

So `words` carries `{term, evidence}`, which is the old shape moved across
unchanged, and the nine shipped asks carry their per-word citations. Twenty
evidence entries came off the rigs and twenty went back onto the words.

The vocabulary is still one axis per word, and that is now a known gap rather
than a rule. "Punchy" answers two axes at once and cannot be written, while a
compound word would let the contradiction check see a real contradiction it
currently cannot. What is missing is how to divide a step between two axes, and
nobody has measured that, so it waits on the sweeps.

`mutations` was read nowhere at all. It moves rather than going, because a
record of what a correction changed is a record about the ask, and it is called
`corrections` on the other side: the field is being moved anyway, and
"correction" is the word the workflow page and everybody using it already use.

Moving it gave it the reader the rule above asks for. `tone build` reads the
history out, and says of each entry that this rebuild does not replay it. That
is true and not obvious. A correction's paths point into the plan it was made
against, and a rebuild makes a new one. An entry with no verdict is the one
worth surfacing, because it has been built and not yet listened to.

### Where the words are resolved, corrected again

An earlier draft said the resolution moves into translate. It cannot, and the
reason is worth writing down because it looks like it should.

Resolving "mid-forward" into a knob position needs the resolved chain: which
blocks are actually in it, what parameters those models have, and the corpus
statistics for them. That chain does not exist until the compiler has built it
from the plan, so a translate step that tried to resolve words would have to
build the chain first, which is the compiler's job and would mean two of them.

So the words are not embedded in the plan and not resolved before it. They are
passed alongside it:

```
Resolve(plan, words, catalog, stats)
```

which keeps resolution where the data is, takes prose out of the plan, and
leaves exactly one implementation. A RigSpec read off disk by
`presets compile --rig` carries settings that were already applied, which is
what makes it a plan rather than a request, and needs no words at all.

### Staying in the RigSpec

`chain` with each entry's `role`, `gear`, `capture`, `substitute` **and its
`evidence`**, plus `instrument`, `id`, `schema`, `version`.

That is the portable set: what the gear is, what order it is in, and why each
piece of it is believed to be there. All of it reads the same on a device nobody
has written a driver for.

The evidence on a chain entry stays here, and John settled that on 2026-09-19
after an earlier draft of this record had it moving to the ask. It follows from
#144: a RigSpec is what gets published, and a published rig that cannot say why
this amp is a rig nobody can check. So the split is by scope rather than by
kind. Why the request was made is the ask's; why this piece of gear answered it
is the rig's.

Mike Dirnt's amplifier is the case that settles it. Its citation is a scanned
1994 magazine page in which the producer says which amp they chose and why, and
that sentence is the reason an Ampeg SVT is in the chain. Publishing the chain
without it publishes an assertion.

### Moving to the Plan

The resolved `models`, the `settings`, `position`, `device`, `snapshots`,
`footswitches`, `controllers`, `sections` and `target`.

Settings go with the models because a knob position is only meaningful against
the block it is turning, and that block is a Helix model. That is what #139
decided from the other side: "more drive" is a request and `drive: 0.7` is a
plan. `models` was already keyed by device, so the contract has been
half-expecting this since it was written.

### Two of those stayed, corrected 2026-09-26

`settings` and `sections` are still on the RigSpec, and the paragraph above is
wrong about both.

`settings` is not a knob position. It is seven words that mean roughly the same
on any amplifier, and the compiler puts each on whichever control the model has
for it. `drive: 0.7` under `settings` reaches a Drive or a Gain or neither, and
a model with no control for it refuses the word. What #139 decided is that a rig
may not name a device parameter, and `settings` names none. The device
parameters, `Sag` and `Bias X` and the rest, are the ones that went, as
`plan.Block.Params`.

`sections` is what somebody plays: Verse, Chorus, a solo. `snapshots` are what a
device stored. The two answer different questions and the contract already said
so, so a rig keeps its sections and the compiler turns them into the plan's
snapshots.

### A Plan is not a contract, corrected 2026-09-26

An earlier draft called this layer a PlanSpec and had it arriving with an
OpenAPI contract like the two above. It does not, and the pattern in the tree
says why.

Two packages here are contract-backed, `rig` and `tone`, and both are formats a
person authors. Everything else that gets serialised is hand-written Go with
json tags: `measured.Curves` is written into `resources/sweeps`, and so are
`corpus`, `catalog` and `preset`, which is the `.hlx` document itself. Becoming
a file is not what earns a contract. Being typed by somebody is.

A Plan is written by a driver and read back by the compiler. The nearest thing
to it is `pkg/sdk/preset`, and that is the shape it takes: hand-written types
with tags, and strict decoding so a misspelt field is refused rather than
dropped, which is the one thing the contract would have bought.

Generating it would also have cost three things. `Params` is keyed to
`catalog.ParamValue`, which has unexported fields and marshalling of its own,
and `Attrs` holds `json.RawMessage`; `pkg/mcp/internal/tools/register.go`
already hand-patches the schema for both because reflection cannot express them.
Generated optionals are pointers, which suits a document being unmarshalled and
not an intermediate the compiler builds a block at a time. And the package holds
`BlockLookup` and `Limits`, which are not document types at all.

### Splitting

`instrument` is in all three. A ToneSpec says which instrument the request is
about, a RigSpec which one the gear is for, and a Plan which one it was built
for. They are the same value and three different claims, and folding them would
mean a document that cannot be read without the one above it.

## The driver

A driver owns everything a modeller knows about itself: its catalog, its limits,
its routing, and the file it writes.

```
RigSpec + Setup ──▶ driver.Fit ──▶ Plan ──▶ driver.Write ──▶ a file the device loads
```

What the Helix driver supplies today, and therefore what the interface asks for:
the catalog of what the device can do, the DSP budget a chain is fitted to,
routing built from `HelixControls.json` enums, snapshots and footswitches, and a
`.hlx` writer.

The interface is derived from what that driver actually needs rather than from
what a second device might. The risk in building this before a second device
exists is designing for a requirement nobody has, and the mitigation is that
shape: no method exists here that the Helix driver does not already call.

What a second driver would have to supply, written down so the next one is a
checklist rather than a redesign:

- a catalog: models, parameters, ranges, costs
- a way to say whether a chain fits
- a mapping from a role and a position to wherever that device puts blocks
- a writer for its own preset format
- measurements taken on that device, because a ranking built from one device's
  readings ranks that device's blocks and `measured.Library` already refuses a
  setup naming another

## What this costs

The measurements are safe. They are keyed by device already and the guard
against using one device's readings for another is in place.

`pkg/sdk/internal/recipes` reads seven of the moving fields across four call
sites and is the package that changes most. It is also the package #136 renames,
which is why that task says to follow the semantic change rather than lead it:
renaming files that are still RigSpecs underneath moves the confusion instead of
removing it.

The nine shipped rigs are RigSpecs holding authorial fields. Each becomes a
ToneSpec beside a RigSpec, and the conversion is mechanical but is nine files of
real research that must not lose a citation on the way.

`docs/recipes.md` is already wrong on its central claim, "a rig is written as a
RigSpec, the project's only hand-authored format", and stops being salvageable
here. It folds into the ToneSpec documentation.

## Order

1. Rename `pkg/sdk/chain` to `pkg/sdk/plan`, which frees the vocabulary the rest
   of this uses.
2. Move the authorial fields to ToneSpec, converting the nine shipped rigs in
   the same change.
3. Give `pkg/sdk/plan` the device-bound fields, and take them off RigSpec.
4. Put the Helix realisation behind a driver, derived from what it calls.
5. Retire the word recipe (#136).

Each step leaves the suite green and the gate passing.

Step 2 cannot be half done: a field read from two documents is the drift this is
meant to remove. It also cannot be done before the nine rigs are converted,
because they are the files carrying the fields it takes away, and an earlier
draft of this list had those as separate steps. Either order leaves the suite
red in the middle, so they are one step.

## What this does not do

It does not let a request name where in the chain a block goes (#132). That
stays a RigSpec-layer edit until somebody wants it.

It does not build a second driver. It makes building one a day's work instead of
a rewrite.

### `target` says which device, not how somebody plays, corrected 2026-09-27

`Target` carried `technique` as well as `device` and `catalog`. It no longer
does.

Two unlike things were in one field. Which hardware and which catalog release is
the plan's business. How the person plays is authorial, and ToneSpec already had
a top-level `technique` with the same three fields and its own evidence, so the
claim had two homes.

It had already drifted before anybody noticed the duplication. mike-dirnt's rig
said `target.technique.attack: fingers`; its ask said `attack: pick`, backed by
a 1994 Bass Player interview in which he answers "Do you always play with a
pick?" with "Yeah". The rig's copy was deleted as stale on 2026-09-26, a day
before anybody saw it was structural rather than a typo.

John settled it on 2026-09-27: `technique` comes off `Target`, and ToneSpec is
the only place how-somebody-plays is stated. Nothing read the plan's copy, so
nothing moved with it; the `Technique` schema in rigspec.openapi.yaml was
unreferenced after the change and went too, along with four dead type aliases
and thirteen dead constants in `pkg/sdk/rig/types.go`. Nothing in the corpus
stated it, so there was no migration.
