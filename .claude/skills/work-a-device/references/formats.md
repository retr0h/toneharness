# Which file is which, and what converting loses

| What      | Is                                                          |
| --------- | ----------------------------------------------------------- |
| `.hlx`    | one preset, plain JSON, what HX Edit imports and exports     |
| `.hls`    | a setlist: 128 slots, zlib inside base64                     |
| `.hlb`    | a backup: eight setlists, so the whole instrument            |
| `.bin`    | the bytes a device sent for a slot nothing could read        |
| rig YAML  | this project's own format, gear a person recognises          |

`slots import --preset` takes the first or the last of those. `--file` takes a
`.hls` or `.hlb`. Getting them the wrong way round fails in a way that reads like
a missing file.

## Out and back

```bash
mise exec -- go run main.go slots export --file device.hlb --slot 3 --out slot3.yaml
mise exec -- go run main.go presets compile --rig slot3.yaml --out slot3.hlx
```

A preset read into a rig and compiled again is the preset it came from, asserted
over every HX Stomp preset in the corpus, so it is a measurement rather than a
claim. Two things make that work, and both matter if the rig is hand-edited.

A lifted rig records `models:`, the exact model each piece of gear resolved to.
**665 models share only 469 names**, and one name can match two channels of the
same amplifier, so a rig carrying the name alone would rebuild into a different
preset. Deleting that line makes compiling fall back to resolving the name, which
is right for a rig somebody wrote and wrong for one lifted off hardware.

Compiling writes the chain into an untouched preset the device itself wrote, so
the result carries the inputs, outputs, split and join a device expects.
`--template` uses a particular preset as that base instead, which is what makes a
rig read off a device rebuild exactly.

`slots export --as hlx` writes the device's own file instead of a rig: a faithful
copy rather than a reading, carrying the routing and snapshots a rig models but
nobody chooses.

## What the device gives back is not a `.hlx`

It is an internal document beginning with a 48-byte table of byte offsets into
itself. **The device seeks with that table rather than walking the document**, so
a re-encode that changes any field's byte width shifts every offset after it.
The device accepts such a write and then reads the preset as empty. Two
reference projects lost hardware sessions to exactly this.

Which produces the one trap worth memorising. **A written preset can render empty
and read back fine**, and reading it back can never tell you. Reading walks the
MessagePack and finds a key wherever it sits; the device seeks to where it put
one. The device test passes for the same reason: it writes, reads back and
compares, and both halves go through the reader that walks.

Two separate faults have produced that empty chain, and both are fixed. A
document whose length disagrees with its own offset table is one. The other was
the order of the five keys inside a block body, where the device puts the model
reference first and this tool put it last, so it found a class where a model
should be. Measured through the loop at the time: 4,034 Hz for a preset HX Edit
wrote against 147.67 Hz for one this tool built, the second being a bass going
down a cable through nothing.

Neither is a reason to hand-audit a chain now. The bytes a write produces are
asserted against real captures, and what to do when a reading still looks wrong
belongs to the `measure-a-device` skill.

So writing a preset synthesised from nothing is the least-solved thing in the
space and no reference project does it. **The reliable shape is read a preset,
change it, write it back.** The corpus agrees from another direction: 98.6% of
real presets carry `inputA`, `outputA`, `split` and `join` routing that a
generated one has none of.

## Editing a setlist container

`compression.crc32` and `decompressed_size` describe the inflated payload and
**must be recomputed on write**. A stale checksum is rejected by whatever loads
the file next, and the message it gives blames the wrong thing.

An empty slot is an entry whose `tone` has no `dsp*` key, and a device-written
setlist always holds all 128 of them, most untouched.
