# A rig fits the person holding it

Date: 2026-09-30

**Status: implemented.** A Setup carries `technique`, and
`pkg/sdk/internal/compile/playing.go` turns the gap between two right hands into
a word. Brightness is deliberately not compensated, for the reason at the end.

## The problem

Every figure this project ships was measured off somebody else's playing. A rig
built for Mike Dirnt is built from records he made with a pick, so somebody who
plays with fingers loads it and finds it dull. Nothing in the chain is wrong.
The right hand changed and no part of the system knew.

The contract had already named this. `Technique` has existed with
`attack: [pick, fingers, slap, thumb, hybrid]`, and its own description said the
fix is in the amp and the compressor rather than in the player, and that stating
both sides lets the difference be compensated rather than discovered.

It had one side. `technique` hung off the subject, meaning how the emulated
player played, and there was nowhere to say how the person holding the
instrument does. The description promised the other side sat "under `target`",
and no `target` existed anywhere in the schema. The feature was specified,
described in prose, and absent.

Worse, the side that did exist moved nothing. `ask.Technique.Attack` reached
`Intent.Attack` and then `claimed()`, which can pull a block into a chain for a
word that has nowhere to land. But the move table is keyed by vocabulary terms,
and `pick` and `fingers` are not terms. The value travelled the whole way and
fell out the end.

## What was decided

**The Setup carries it.** A right hand changes on the same clock as a bass and a
pedal, which is the argument the Setup exists on. Restating it per request is
how the twelfth request comes to contradict the first.

**The compensation is a word, not a move.** `compile` turns the gap between the
two techniques into a term and appends it to the intent. Everything downstream
is then what it already was: `demand` pulls a compressor into a chain that has
none because the word asked for it, the corpus decides how far the term travels,
and the run reports it beside every other word. One place decides, and nothing
else learns a new rule.

**One axis, and it says so.** `attack` is what the vocabulary has for the front
of a note, and a compressor's Attack is the control that decides how much of it
gets past. The other half of the difference is brightness. The direction is not
in doubt and the size is not measured, so it is not compensated and the report
says which half was left.

**One word each direction, with no size.** The attack terms are a scale on one
control, so reaching for `percussive` to mean "a wide gap" would have worked
arithmetically and lied: percussive means the string against the fretboard,
which is not what somebody playing a picked rig with fingers is asking for. So
`audible-pick-attack` when this person plays softer than the subject and
`soft-attack` when harder, whatever the size of the gap.

**An ask's own word wins.** Two terms on one axis cancel by design. Appending a
second would have taken the ask's own word out with it, which the first working
version did: asking for fingers on a rig whose ask already says
`audible-pick-attack` moved the compressor nowhere and reported both as
contested. So the compensation stands down and says it did.

**A thumb ranks with fingers.** The rank is about what is touching the string
rather than about the tone, and a thumb is flesh. Somebody playing thumb against
a fingered rig is asking for no compensation.

## What was rejected

**A `target` section on the ask**, which the old description implied. An ask
changes per request and a right hand does not, so it would have been the wrong
clock and a field somebody has to restate.

**Compensating brightness with a default step.** `move.go` has a `fallbackStep`
for exactly this, a tenth of a range where the corpus cannot measure. Using it
here was tempting and would have been a number nobody took on an axis nobody
measured. Two unmeasured claims are worse than one, and the second would have
been invisible behind the first.

**A research project into how other people solve it.** Forums have opinions and
the direction is not in dispute. What is missing is the size, and this
repository measures better than it researches: record the same part fingered and
picked, measure both, and the difference in transient, centroid and harmonics is
the compensation in the same figures the solver already aims at. That is the
work worth doing next, and it is a measuring job.

## What it cost

`Resolve` gained a return value, which reached the `Compiler` interface, its
generated double, and `presets.Make`. `Client` gained `WithSetup`, and
`presets make` a `--setup` flag, so the Setup reaches a build that starts from a
rig rather than from an ask.

## What it does not compensate, by decision

**Brightness.** A pick is brighter as well as sharper, and nothing here moves a
tone control for it. Not an open item: the direction is not in doubt and the
size is not measured, and this project does not write a number nobody took. The
report says which half it left, so the person reading it knows.

Closing it is a reading rather than a decision. Record the same part fingered
and picked, measure both, and the difference in transient, centroid and
harmonics is the size, in the same figures the solver already aims at.

**`position` and `muting` reach no control.** `pkg/cli/rig.go` prints them as a
sentence, so a reader sees them; no build changes because of them. They are on
the schema because a technique is three things, and `attack` being the only one
that moves anything is recorded here rather than discovered later.

An earlier version of this section said they were read by nothing, which was
wrong. The test meant to establish that matched bare identifiers, so
`lipgloss.Position` in a renderer counted as somebody reading a technique's
position. It reads selectors in packages that can name a contract type now.
