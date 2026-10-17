# sweeps

What the device actually does, measured rather than asserted. Two kinds of file:

`hx-stomp/fingerprints.json` is one reading per block at its own defaults, which
is what picking a block out of hundreds needs.

`hx-stomp/<model>.json` is every control of one block swept across its range,
which is what setting a chain needs once the chain is picked.

**Nothing here is hand-written.** A slope typed into a file is the guessing this
whole exercise removed, and a figure copied into prose goes stale the first time
anything is re-measured. So the page describing these is generated from them:
the readings in this directory. They say what the figures mean, what each one is
worth against the noise beside it, and how to add to them.

`just pack-measured` puts the fingerprints into the binary. Only those; the
curves stay here.

## Which blocks are swept, and why those

A fingerprint exists for all 661 blocks and a curve for 27 of them, because a
curve costs a few minutes of somebody's pedal and a fingerprint comes out of one
reading. So the question is which blocks earn one.

The answer is the blocks `tone tune`'s shortlist reaches for. It ranks what
would close an axis the dials could not, and for a bass chain already holding an
amplifier and a cabinet it picks drives, filters, mods, a compressor and wahs:
the amplifiers it ranks highest are refused, because a chain may not hold two.

Eight were swept on 2026-10-17 for exactly that reason. Before them,
`tone tune --extend` could name a block to add and could say nothing about where
to set it, because not one block the shortlist picked had a curve. It reported
"has no committed sweep" every time it ran.

**Every curve records the rig it came off**, which the first nineteen do not:

```
rig          ["External Headphones", "HX Stomp"]
open         true
pinned_on    External Headphones
```

So those nineteen and these eight are not comparable in absolute terms, and
nothing reads them that way: a position is scored against the other positions of
its own curve. A figure measured on one rig is a figure about that rig.
