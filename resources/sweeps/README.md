# sweeps

What the device actually does, measured rather than asserted. Two kinds of file:

`hx-stomp/fingerprints.json` is one reading per block at its own defaults, which
is what picking a block out of hundreds needs.

`hx-stomp/<model>.json` is every control of one block swept across its range,
which is what setting a chain needs once the chain is picked.

**Nothing here is hand-written.** A slope typed into a file is the guessing this
whole exercise removed, and a figure copied into prose goes stale the first time
anything is re-measured. So the page describing these is generated from them:
[docs/measurements.md](../../docs/measurements.md). It says what the figures
mean, what each one is worth against the noise beside it, and how to add to
them.

`just pack-measured` puts the fingerprints into the binary. Only those; the
curves stay here.
