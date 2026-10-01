# A recipe compiles to a RigSpec

2026-09-18

**Status: implemented.** `pkg/sdk/tone` is the ask, `pkg/sdk/rig` is what it
resolves to, and `pkg/sdk/translate` is the compile between them.

## The problem

One format is doing two jobs and neither well.

A rig file today carries what a person asserts, with citations and caveats, and
also carries knob positions. The first is knowledge. The second is a conclusion.
Nothing separates them, so a conclusion nobody could defend sits in the same
file as a quote from the person who was in the room, wearing the same clothes.

That is how `dark` came to move Treble down by a quarter of its range. A quarter
is not a fact anybody established. It was typed, and because it was typed in the
same file as the evidence, it reads as though it were evidence.

The format is also too large. Ten axes, twenty-four terms, a map of which
control each reaches and how far, a vocabulary check, an axis check. All of that
machinery exists to get from a human word to a knob by hand. It is scaffolding
around the unsolved problem, and it is most of what makes RigSpec hard to read.

## The shape

Three layers, each with one job.

| Layer       | Written by | For                                  |
| ----------- | ---------- | ------------------------------------ |
| **recipe**  | a person   | what we mean, and why we believe it  |
| **rigspec** | the tool   | exact resolved values, deterministic |
| **.hlx**    | the tool   | what the pedal loads                 |

The middle one is a lockfile, not bytecode. `package.json` says "react, about
version 18", which is intent, readable, arguable. `package-lock.json` says
"react 18.2.0, this hash", which is generated, committed, technically readable
and never hand-edited. That is exactly the middle layer, and it is a pattern
people already trust.

### What a recipe holds

Only things a person can assert:

- who, which band, which era, which years
- which records back it
- gear by the name a person uses: "Ampeg SVT", never a model identifier
- the character words, and how it is played
- evidence, caveats and confidence for all of the above

### What a RigSpec holds

Everything nobody should be typing:

- resolved model identifiers for this device
- block order and position relative to the amplifier
- the DSP budget it fits in
- controllers, footswitches, snapshots
- **the numbers**

Plus, for every value, what produced it: the recipe line, the corpus median, the
character term, or a figure measured off a record. A lockfile that does not say
why is a lockfile nobody can audit.

### Sharing

Three levels, and each is the right answer to a different request.

- The **recipe** is what you share to be argued with. It is the claim.
- The **rigspec** is what you share to be reproduced exactly, on any Helix, by
  somebody who does not have the corpus this was compiled against.
- The **.hlx** is what you share with somebody who just wants the sound.

## Why this order

The measuring loop is the bigger prize and this should still come first, because
of what it does to the cost of everything after.

Right now, improving a mapping means editing nine rig files by hand and
re-reading nine sets of evidence. Once values are generated, improving the
generator improves every rig at once and costs one command. The split is what
makes the measurement work cheap to land rather than expensive.

It is also reversible in a way the current arrangement is not. A generated file
can be regenerated when the generator gets better. A hand-typed one has to be
found and corrected by somebody who remembers why it said what it said.

## The rule that keeps it honest

A person may pin a number, and must show why.

Sometimes a human genuinely knows a value: a rundown photograph shows a knob at
three o'clock, or Line 6's own manual states a setting. Refusing that would lose
real sourced knowledge and would turn the split into "the computer knows best".

So a recipe may carry a value, with evidence, exactly as it carries a claim
about gear. The compiler honours it, and the RigSpec records that it was told
rather than that it worked it out. What a person may not do is write a number
with no source, which is the thing that happened and the reason for all of this.

## What this deletes

Most of the vocabulary machinery, once a word names a measured region rather
than a direction on a knob. The axes stay, because saying `mid-forward` should
still mean "not `scooped`", and two words answering one question is still a rig
claiming nothing. What goes is the table of which control each word reaches and
how far, along with the fallback chain that searches for a control when the
first one is missing.

That is the half of the repository that feels unnecessary, and it is unnecessary
for a good reason: it is a hand-built bridge across a gap that measurement
crosses properly.

## Genre

"Punk" is not a word somebody defines. It is where punk records sit.

Measure a pile of them, and the region they cluster in is what the word means,
in the same nine figures everything else is measured in. Generating a punk rig
is then "find settings that land in that region", which is the same solver that
matches one player, pointed at a cluster instead of a record.

No new machinery, and no new vocabulary to argue about. A genre is a corpus with
a name.

## What is not decided here

**Where the numbers come from.** Until the loop in
[measuring.md](../../measuring.md) runs, the compiler would emit the same
guesses it emits today, just in a file that admits they are derived. That is
still an improvement and it is not the finished thing.

**Whether RigSpec is checked in.** A lockfile usually is. It makes diffs
reviewable and lets somebody reproduce without the corpus. It also churns
whenever the generator changes, which is noise in a pull request. Worth deciding
before the first one is written rather than after.

**What happens to the nine existing rigs.** They are recipes with numbers in
them. Splitting them is mechanical for the gear and the evidence, and a
judgement for every value that is currently typed: each one is either evidence
somebody can point at, or it is a guess that should be generated.

## Related

- [measuring.md](../../measuring.md), the loop that would supply the numbers
- [nothing here has ever heard anything](2026-09-18-nothing-here-has-ever-heard-anything-design.md),
  which argues why the numbers are guesses today
