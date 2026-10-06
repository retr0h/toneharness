# Get a preset onto the device

Two ways in, chosen by whether it is being tried or kept.

## Trying something: write no flash at all

```bash
mise exec -- go run main.go presets compile --rig rig.yaml --out a.hlx
mise exec -- go run main.go device play --preset a.hlx
```

`device play` puts the chain in front of the device and stores it nowhere. No
slot is read and none is written, so there is nothing to put back. What it
replaces lasts until the next preset is selected or the pedal is power cycled.

Use it for anything being auditioned. A slot is flash, flash wears out, and six
hundred chains through slots is six hundred writes for readings nobody keeps.

## Keeping it: into a slot

```bash
mise exec -- go run main.go slots import --preset a.hlx --slot 42C --json
mise exec -- go run main.go presets show --slot 42C --json
```

**Write only to a slot that is empty or that its owner named.** `slots list`
says which, `TONEHARNESS_SCRATCH_SLOT` is how one is named, and a high bank is
the convention only because low banks are where most people keep what they play.

A device has no undo, so whatever the slot held is read and saved first
and the answer says where. Put it back with `slots import --preset` and that
file. `--backup-dir` moves where they go; a backup never replaces a file already
there, and one that cannot be written in full leaves no file behind and nothing
is written to the pedal.

A slot holding no blocks is kept as a `.bin` instead, the bytes the device sent,
and `slots import --preset <file>.bin` puts one back as it came off. The slot
keeps the name it has, because a `.bin` carries none.

With `--file` it edits an HX Edit backup instead of the device. `--preset` is the
`.hlx` going in, `--file` is the `.hls` or `.hlb` being edited, and the wrong way
round fails in a way that reads like a missing file.

## What a write costs if it is done wrong

Three rules, and the device punishes each. They decide what a failure means.

**The document must be byte-exact.** One whose length differs from what its own
offset table claims is accepted and then read by the pedal as an empty chain.

**A message goes out in pieces, paced on that channel's own acknowledgement.**
Sending a whole preset at once fills the device's receive window and stalls the
endpoint, and **the interface will not be claimed again until the pedal is power
cycled**. A misplaced or interrupted write is how that happens.

**A write is not finished when it is accepted.** A slot write is answered when
the device has the document; the erase and program that follow never appear on
the wire, so waiting for a completion notification waits ten seconds for one that
is not coming while the preset already sits in the slot. Both status 0 and
status 1 have been seen for a write that landed, so neither is read. A second
write landing on the first stacks its commit, so what is left is the settle.

Status `1` means accepted and completing later, not failed. It is not validation
either, because selecting preset 999 on a device holding 126 answers `1` and does
nothing. Anything treating a non-zero status as failure decides every deferred
operation failed. Ctrl-C will not cut a write off halfway either, and
[session.md](session.md) says why.

## Every float is a float32

Writing 0.45 and reading it back gives 0.44999998807907104. A rig lifted off
hardware has been through this and survives unchanged; one somebody typed gets
rounded to what the device can store. Not a bug to chase.
