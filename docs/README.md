# docs

Two things, and they are different kinds of document.

| page                               | what it covers                                                                                                                  |
| ---------------------------------- | ------------------------------------------------------------------------------------------------------------------------------- |
| [architecture.md](architecture.md) | How the whole system works and fits together, and what is not built yet. The one file to read first.                            |
| [superpowers/specs/](superpowers/) | Why it is this shape. Dated records of what was decided, superseded rather than rewritten, so one of them is history not truth. |

Nothing else is here, on purpose. How to do a job is a skill, in
[.claude/skills/](../.claude/skills/), and
[AGENTS.md](../AGENTS.md#finding-your-way-around-the-domain) is the table that
says which of the five owns what. The pages that used to sit in this directory,
on authoring a rig, the solve, measuring, the catalog, the device and the USB
protocol, are each in the skill that owns the job, or beside the code in
[`pkg/sdk/internal/wire/README.md`](../pkg/sdk/internal/wire/README.md). One
owner per fact is the point: a second copy drifts and is then confidently wrong.

For how to work *on* this rather than how it works, read
[CONTRIBUTING.md](../CONTRIBUTING.md): setup, layout, conventions and testing.

## Where the architecture is recorded

The current shape is
[ToneSpec is the ask](superpowers/specs/2026-09-19-tonespec-is-the-ask-design.md)
with
[A rig is a plan, for one device](superpowers/specs/2026-09-19-a-rig-is-a-plan-for-one-device-design.md)
beside it, together superseding
[RigSpec as the one model](superpowers/specs/2026-09-06-rigspec-as-the-one-model-design.md).

A ToneSpec is what somebody means, a RigSpec is the gear that answers it, and a
Plan is that rig on one device. A person writes the first two; the compiler
produces the third.
