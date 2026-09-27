# Recording what a person thought of it

A correction is a round of asking. The rig holds the settings it arrived at
without saying why they are what they are, so the reasoning lives on the ask,
beside the words that started it.

```yaml
corrections:
  - ask: make it clunkier
    changed:
      - { path: chain[1].settings.drive, from: 0.47, to: 0.58 }
    reason: >-
      Line 6 document Sag as "lower values offer tighter responsiveness…
      higher values provide more touch dynamics & sustain". Read "clunky" as
      a looser power-amp feel rather than more gain.
    verdict: closer, but muddy now, so keep the feel and put the drive back
```

Nothing in this project can hear. Every other input is a measurement or an
assertion, and **the verdict is the only place a human ear is written down.** It
is also the only thing unrecoverable later: a catalog can be regenerated next
year, and nobody can go back and ask themselves what they thought of round three.

Four fields, four jobs:

- `ask` is the person's words, verbatim. "Clunky" is not a parameter, and
  normalising it away loses the question.
- `changed` is what moved.
- `reason` is how the ask was interpreted, cited. If the reading was wrong, this
  is the line that shows it, rather than only that the value was.
- `verdict` is what it sounded like. **Absent means not yet heard**, which is
  useful state rather than a gap.

## Append-only, and never replayed

The rig's `chain` always holds the current state. A correction's paths point into
the build it was made against and a rebuild makes another one, so replaying the
history would apply somebody's round three to settings that never went through
rounds one and two. `tone build` reads the entries out and says that the rebuild
does not replay them, which is true and not obvious.

Reconstructing a rig from its history would be more elegant and much worse to
read, and a person reads these files.

## What none of this can tell you

Whether it sounds right. That is a person with the preset loaded, and the answer
belongs here so the next round starts from it rather than from nothing.
