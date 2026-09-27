# A rig is a plan, for one device

2026-09-19

## Two changes, one edit

A RigSpec is doing two jobs it should not be doing, and both are moves of the
same fields. Doing them apart means moving each field twice.

It carries what a person wrote, and that belongs in the ask: who the rig is for,
why each claim is believed, what was corrected and when. It also carries Line
6's own vocabulary next to gear named the way a musician names it, so a document
that could describe any modeller is welded to one.

John settled both on 2026-09-19. A ToneSpec never names a knob position, and a
RigSpec becomes a device plan with a driver behind it.

## What a rig is after this

Three layers, which is what has been said all along and has not been true:

|              | holds                                        | written by |
| ------------ | -------------------------------------------- | ---------- |
| **ToneSpec** | what somebody means, and why it is believed  | a person   |
| **Chain**    | gear in signal order, named as a person does | the tool   |
| **Plan**     | that chain realised on one device            | a driver   |

A Chain is the portable middle nobody had. It is what survives moving to another
modeller: roles in order, gear by its real-world name, the instrument it is for.
A Plan is that chain fitted to hardware: which model each piece of gear resolved
to, where it sits in the DSP, what the footswitches do.

`models` was already keyed by device. The contract has been half-expecting this
since it was written.

## Which fields move, and why

### To the ToneSpec

`subject`, `aliases`, `default`, `extends`, `character`, `technique`, `played`,
`confidence`, `corrections` (was `mutations`), and the document's own top-level
`evidence`.

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

`character` and `technique` are the two that also reach the compiler, in
`compile/move.go` and `compile/demand.go`. They move with the rest and the
compiler stops reading them, because resolving a word into a knob position is
the translation step's job and doing it twice is how the two answers drift. This
is the same argument the measurement code already lost once, in two languages.

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

### Staying in the Plan

`chain` with its resolved `models`, `settings`, `substitute` **and its per-entry
`evidence`**, plus `device`, `snapshots`, `footswitches`, `controllers`,
`sections`, `target`, `id`, `schema`, `version`.

Settings stay because a knob position is the plan. That is what #139 decided:
"more drive" is a request and `drive: 0.7` is a plan, and the number is only
meaningful against a chain.

The evidence on a chain entry stays, and John settled that on 2026-09-19 after
an earlier draft of this record had it moving. It follows from #144: a RigSpec
is what gets published, and a published plan that cannot say why this amp is a
plan nobody can check. So the split is by scope rather than by kind. Why the
request was made is the ask's; why this model answered it is the plan's.

Mike Dirnt's amplifier is the case that settles it. Its citation is a scanned
1994 magazine page in which the producer says which amp they chose and why, and
that sentence is the reason an Ampeg SVT is in the chain. Publishing the chain
without it publishes an assertion.

### Splitting

`instrument` is in both. A ToneSpec says which instrument the request is about;
a Plan says which one it was built for. They are the same value and different
claims, and folding them would mean a plan that cannot be read without its ask.

## The driver

A driver owns everything a modeller knows about itself: its catalog, its limits,
its routing, and the file it writes.

```
Chain + Setup ──▶ driver.Fit ──▶ Plan ──▶ driver.Write ──▶ a file the device loads
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

The nine shipped rigs are RigSpecs holding authorial fields. They become
ToneSpecs plus Plans, and the conversion is mechanical but is nine files of real
research that must not lose a citation on the way.

`docs/recipes.md` is already wrong on its central claim, "a rig is written as a
RigSpec, the project's only hand-authored format", and stops being salvageable
here. It folds into the ToneSpec documentation.

## Order

1. Add the Chain and Plan contracts, leaving RigSpec in place.
2. Move the authorial fields to ToneSpec, and move word-to-setting resolution
   out of the compiler and into translate.
3. Put the Helix realisation behind a driver, derived from what it calls.
4. Convert the nine shipped rigs.
5. Retire RigSpec and the word recipe (#136).

Each step leaves the suite green and the gate passing. Step 2 is the one that
cannot be half done: a field read from two documents is the drift this is meant
to remove.

## What this does not do

It does not let a request name where in the chain a block goes (#132). That
stays a RigSpec-layer edit until somebody wants it.

It does not build a second driver. It makes building one a day's work instead of
a rewrite.
