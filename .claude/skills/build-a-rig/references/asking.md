# Write the request

Three documents. Two are written by a person.

| Document     | Holds                                | Changes                          |
| ------------ | ------------------------------------ | -------------------------------- |
| **Setup**    | what somebody owns                   | when they buy something          |
| **ToneSpec** | what they want this time             | every request                    |
| **RigSpec**  | the gear those two resolve to        | every request, or hand-authored  |

Folding the instrument into the ask would mean restating their bass in every
request, and the twelfth one would contradict the first.

A researcher writes a RigSpec by hand too, when the gear is the thing being
established. What every field on either document may say, and which of the two it
belongs on, is the `write-a-spec` skill's; this page is only how a request gets
resolved.

## What a ToneSpec may say

Every field is optional and any combination is legal. The contract is the only
authority: the `description:` fields in
`pkg/sdk/tone/data/tonespec.openapi.yaml`.

**It never carries a knob position.** Decided 2026-09-19. "More drive" is a
request; `drive: 0.7` is a plan, and a number is only meaningful against a
chain. Words go in `words` or `nudges` and are resolved against measured
corpus data later. Inventing a number here is the guessing this project exists
to remove.

`gear` entries name real-world gear the way a person says it. `insist: true`
means refuse rather than substitute.

**Order is not what you typed.** Gear is sorted into the ordinary signal path,
because listing gear is not stating one: a compressor belongs in front of the
amplifier whichever way round it was written. Where somebody means otherwise,
`after: amp` on an entry puts it behind that role. A drive behind the amplifier
is the case worth knowing, and it is a known way to use one rather than a
mistake.

Named by role rather than by position, because a request does not know how many
blocks the chain ends up with: "after the amp" survives the compiler adding a
cabinet and "position 4" does not.

## Resolve it

```bash
mise exec -- go run main.go tone build \
  --ask request.yaml --setup mine.yaml --json
```

**Read the notes.** They are in the answer, not printed above it. They say
what it substituted, what it assumed when no setup was given, and what it
could not honour. A run reported without its notes is a run that hid half of
what happened.

Worked examples:
[marketplace/core/examples/](../../../../marketplace/core/examples/), the
`*.tone.yaml` ones.
