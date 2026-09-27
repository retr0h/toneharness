# The loop, and which one a question belongs to

A record is the far end of a signal chain: an instrument, an amplifier, a
microphone, a desk and a master, all in one number. It says what the answer
should sound like. It cannot say what any one control did, because every one of
them is already in it and none can be moved.

So a known signal goes in, and what comes out gets measured. The difference is
the device and nothing else.

```
                 act                observe
    settings ──────────> pedal ──────────> the figures
        ▲                                       │
        └────────── compare to the target ──────┘
```

## It is closed on numbers, not on hearing

The loop settles whether a setting moved a figure toward a target. It cannot
settle whether that sounds right, because nothing here can hear. There are two
loops and only the inner one is automatic:

- **Inner, automatic, fast.** Move a control, measure, compare, repeat. Runs
  overnight with nobody in the room.
- **Outer, human, rare.** Listen to where the inner loop landed and say whether
  the target was worth aiming at. If a preset measures as `dark` and sounds
  wrong, the definition of `dark` is wrong and gets moved.

The inner loop is what makes a word testable. The outer loop is what makes it
mean anything. **Confusing the two is how a project ends up confident and
wrong**, so when a reading disagrees with a pair of ears, say which loop the
disagreement belongs to before changing anything.

## The cheap half and the expensive half

Two subjects, two commands, and the split is about cost:

```bash
mise exec -- go run main.go measure blocks --json
mise exec -- go run main.go measure controls --model HD2_AmpUSDripmanNorm --json
```

`blocks` takes one reading per block at its own defaults. Sweeping every control
of all 665 is about a hundred and sixty hours, and it is also the wrong thing to
want, because a slope belongs to its chain and a stored one would be rebuilt
before anything used it. What cannot be worked out at solve time is which blocks
belong in the chain to begin with, and that needs one number per block.

`controls` is the expensive half: a control is about two minutes and a
twelve-control amplifier most of an hour. Aim it at the blocks a chain reaches
for.

**The empty loop is measured first and kept as the baseline.** Without it a
figure says nothing: 95 Hz is not what an equaliser does to a bass, it is what
the bass already was. Everything is written as it goes, so a run interrupted
partway keeps what it had and `--resume` picks it up.
