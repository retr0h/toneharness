# Write the request

Three documents. Two are written by a person.

| Document     | Holds                                | Changes                          |
| ------------ | ------------------------------------ | -------------------------------- |
| **Setup**    | what somebody owns                   | when they buy something          |
| **ToneSpec** | what they want this time             | every request                    |
| **RigSpec**  | what those two resolve to            | generated, never hand-written    |

Folding the instrument into the ask would mean restating their bass in every
request, and the twelfth one would contradict the first.

## What a ToneSpec may say

Every field is optional and any combination is legal. The grammar is generated
from the contract and is the only authority:
[docs/tonespec.md](../../../../docs/tonespec.md).

**It never carries a knob position.** Decided 2026-09-19. "More drive" is a
request; `drive: 0.7` is a plan, and a number is only meaningful against a
chain. Words go in `words` or `nudges` and are resolved against measured
corpus data later. Inventing a number here is the guessing this project exists
to remove.

`gear` entries name real-world gear the way a person says it. `insist: true`
means refuse rather than substitute.

## Resolve it

```bash
mise exec -- go run main.go tone build \
  --ask request.yaml --setup mine.yaml --json
```

**Read the notes.** They are in the answer, not printed above it. They say
what it substituted, what it assumed when no setup was given, and what it
could not honour. A run reported without its notes is a run that hid half of
what happened.

Worked examples: [examples/tonespec/](../../../../examples/tonespec/).
