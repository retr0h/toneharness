# The words an ask carries

What a sound should be like is `words` on the ask, **never a knob position**.
Resolving a word into a position is the build's job, and a document that already
holds the position has nothing left to resolve. Decided 2026-09-19.

Each term is one claim, so an ask saying two things says two of them:

```yaml
words:
  - term: mid-forward
    evidence:
      - kind: llm
  - term: short-decay
```

A word is an object rather than a bare string, and the evidence is the reason:
how far a word moves a control is sized from what earned it, so a term with
nothing behind it and a term measured off three records cannot be the same shape.
Leaving `evidence` off says nobody has checked.

`pkg/sdk/internal/compile/data/words.json` is every word, its axis and what it
moves. Read it rather than guessing, and note two things while writing an ask.

**An axis is what makes a term mean something.** Saying `mid-forward` has already
said "not scooped", and an ask claiming both has claimed nothing.

**Some axes move nothing.** They describe the player and the instrument, which no
amplifier, reverb or compressor has a control for. Use them anyway: they are part
of what somebody listening compares the preset against.

## A measured term carries the figures that earned it

```yaml
words:
  - term: clean
    evidence:
      - kind: audio
        measured: { harmonics: 0.12 }
        against: { harmonics: 0.24 }
```

`measured` is this player, `against` is everybody else. Reading half of what the
others read is worth half a step; a gap as wide as their own figure is worth the
whole one, and **nothing is worth more than that**. A term with no figures beside
it moves the whole step, which is what every ask did before any of this could be
measured. `measure players --corpus <dir> --evidence` writes these blocks.

## Which control a word reaches depends on the rig

Which is why the two documents have to be read together. `mids`, `highs`, `drive`
and `low-end` move the first amplifier, `space` the first reverb, `attack` the
first compressor. Where the amplifier has no such control, `mids` and `highs`
look for an equaliser instead, which calls those bands `MidGain` and `HighGain`.
Some amps have a bass knob and a treble knob and nothing between them.

**The amplifier answers first where it can.** It is the voice, and the equaliser
is a correction to it.

## Five ways a word can fail, and they are different sentences

- `nothing acts on this yet` — the project's gap, not the ask's
- `this chain holds no reverb` — the rig's gap, worth knowing because adding one
  would answer it
- `this chain has no reverb, so it is already dry`
- two terms on one axis cancelling, so neither moves and the build names the axis
- `no such word "tight low end" — did you mean tight-low-end?`

An unknown word is reported **and the preset is still written**, because refusing
a word would be refusing somebody the right to describe a sound. Add the missing
word to `words.json` with a sentence saying what it means.
