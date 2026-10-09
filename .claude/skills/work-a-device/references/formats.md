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

A preset read into a rig and built again is the preset it came from, give or take
a handful of fields named below. That is a measurement over the preset corpus
rather than a claim: on a real preset from it, 12 of 494 values differ, and nine of
those are because the preset was written by a different Helix.

Three things make it work, and all three matter if the rig is hand-edited.

**The controls decide which model a name means.** 661 models answer to only 468
names: three are called `1x12 US Deluxe` and only one carries a `Pan` and a
`Delay`. A name alone picks the shortest match, so a rig stating controls gets the
model that has all of them, and one stating none still resolves on the name. Delete
the controls from a lifted rig and it rebuilds into a different model.

A control no candidate carries does not throw away the ones that do. Where no
model has every control stated, the model carrying the most of them wins, and the
name decides only when none of them match. This matters to a hand-edited rig
rather than a lifted one: a lifted rig's control names came off the device and
are all real. Write one by hand with a control name slightly wrong and the model
is still chosen by the rest. `Teletronix LA-2A` with the LA Studio Comp's six
controls and one more besides matched six of seven there and one of seven on a
legacy Tube Comp carrying only a level, and the Tube Comp used to win.

**Each block keeps the key it was filed under.** `position` on a chain entry is the
`blockN` key the device used, which is not the slot the block sits in: 4,437
processors in the corpus have a set of keys that is not the set of slots. The slot
is `@position` in the entry's `attrs`. Renumbering either moves blocks around the
file.

**The rest of the preset is in `rig.preset`.** Everything beside the chain: the
snapshots, each processor's routing, the footswitch and expression assignments, the
DT and Powercab members, the impulse response table. Written by a lift and merged
over an untouched preset the device wrote, so the result carries the inputs,
outputs, split and join a device expects. `--template` uses a particular preset as
that base instead.

What still does not come back: the file's own metadata outside the preset payload,
and the few fields the untouched preset carries that the original lacked, which are
where the editing cursor sat and whether a snapshot had been renamed. None of it is
audible. The device identifier and the schema version come from the catalog in use,
so building somebody's Helix Floor preset for an HX Stomp changes those on purpose.

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
