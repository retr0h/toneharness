# ToneSpec is the ask

2026-09-19

**Status: implemented.** `pkg/sdk/tone` carries the contract, and `tone build`
is what resolves an ask through it.

## The name

**ToneSpec.**

It sits beside RigSpec and says what each is for without a gloss: a ToneSpec
describes the tone somebody wants, a RigSpec describes the rig that makes it.
One is the question and one is the answer.

"Recipe" is wrong twice over. It is already taken here for the curated gear
knowledge, so it names two things. And a recipe is instructions for producing
something, which is exactly what this is not: it is a description of the result
somebody is after, and working out the instructions is the tool's job.

Considered and rejected: `SoundSpec`, which is vaguer for no gain; `IntentSpec`,
which is a programmer's word for a musician's document; and `Brief`, which reads
well in prose and badly in a filename next to `rigspec`.

## Three documents, not two

Writing it down made one thing obvious that the conversation had been eliding.
The ask and the person asking are different, and they change on different
clocks.

|              | holds                    | changes                         |
| ------------ | ------------------------ | ------------------------------- |
| **Setup**    | what somebody has        | rarely, when they buy something |
| **ToneSpec** | what they want this time | every request                   |
| **RigSpec**  | what to build            | generated from the two          |

Folding a person's instrument into the ask would mean restating their bass,
their strings and their pedal in every request, and the twelfth one would
contradict the first. It is a fact about them, not about the sound they are
chasing today.

### What a Setup holds

What the person has and the tool must work with rather than around:

- the device: an HX Stomp, a Helix Floor, what firmware
- the instrument: a Jazz bass, rounds, passive
- gear they own, which is what `requires` was reaching for: purchased models,
  impulse responses, anything a preset can name but a device may not have

**This is the answer to "how do we account for the instrument".** Every figure
in the corpus was measured off a record made with somebody else's bass. A preset
aims the amplifier at that record while the person holds a different instrument,
and the difference between the two is a knob position. Nothing could hold that
difference before, because nothing described the near end.

It is also the answer to "use what they have". If a Setup names an amplifier the
device models, that model is used and nothing guesses. A ToneSpec asking for a
sound and a Setup naming the amplifier that made it are not in conflict: the
Setup wins on gear, because it is a fact.

### What a ToneSpec holds

The ask, in whatever terms somebody has:

- **a player** — "like Mike Dirnt"
- **a genre** — "punk", which is
  [a corpus with a name](2026-09-19-a-genre-is-a-corpus-with-a-name-design.md)
- **words** — "chunky", "chippy", whatever they say, which have to be measured
  regions rather than declarations
- **gear** — "through an SVT", when they want a particular thing
- **a record** — audio handed over, measured, aimed at
- **nudges** — "darker than that", which move from where the last answer landed

Every one of those is optional and any combination is legal. "Punk, but on my
Jazz" is a genre and a Setup. "Like Dirnt but chunkier" is a player and a nudge.
The document is a request, not a form.

## The translation layer

```
   ToneSpec  ──┐
               ├──▶  compile  ──▶  RigSpec  ──▶  lower  ──▶  .hlx
   Setup     ──┘         ▲
                         │
              corpus, catalog, measurements
```

The two names are already in this repository. `compile` turns a description into
a chain; `lower` turns a chain into the device's own file. What changes is what
goes in at the left.

**RigSpec is the deliverable**, and stays what it is: exact, resolved, every
value with a reason. What it stops being is hand-written. A person writes a
ToneSpec and a Setup; the tool writes the RigSpec, and
[a lockfile is the analogy](2026-09-18-a-recipe-compiles-to-a-rigspec-design.md).

Both new documents get the same treatment RigSpec already has: a hand-authored
OpenAPI contract, types generated from it, and a generated reference page. That
is how this project keeps a format from drifting from the code that reads it,
and there is no reason for these two to be the exception.

## What the corpus is for, in this shape

The measurements make the middle of that diagram possible, and there are two
kinds, measured the same way and meaning different things.

**Records** say where a target sits. Pino at 96Hz against 172Hz for the other
eight is a target: it is what a record sounds like, and no setting was involved
in producing it.

**Sweeps** say what a control does. A knob moved through its range against a
fixed reference signal says how far each figure travels and in which direction.

A request is then answered by putting them together: the records say where to
land, the sweeps say which way to turn, and the Setup says what the signal is
before any of it. None of that is somebody's opinion, which is the entire point
of the exercise.

## Two things this does not solve

**Microphones and cabinets.** A cabinet block carries a mic, a position, a
distance and an angle, and each is a parameter like any other, so each is
sweepable once a chain block can be addressed at all. Nothing special is needed.
Nothing can be measured yet either, and [measuring.md](../../measuring.md) says
why.

**Making a new model.** Building an amplifier that sounds like somebody, rather
than choosing among the ones Line 6 ship, is a different exercise: it needs the
sweep rig pointed at capturing a response rather than at describing an existing
model, and it produces a file for the device's marketplace rather than a preset.
Worth noting that the measuring loop built here is most of what that would need,
and worth not confusing with this.

## Order

1. **Setup**, because it is small, it is a fact, and the instrument difference
   is the most common thing that makes a preset wrong for the person holding it.
2. **ToneSpec**, contract first, then the compile path that reads it.
3. **The measurements**, once a chain block can be addressed.

Nothing here should be built before a chain block can be addressed, except the
two contracts, which are hand-authored and depend on no hardware.
