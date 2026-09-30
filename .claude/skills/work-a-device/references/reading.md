# Read what a device holds

Read-only in the strongest sense. The device answers and goes on playing
whatever it was: nothing is selected, loaded or written.

```bash
mise exec -- go run main.go slots list --json
mise exec -- go run main.go presets show --slot 31A --json
mise exec -- go run main.go slots export --slot 31A --out lead.yaml
mise exec -- go run main.go device current --json
```

A slot is addressed the way the pedal labels it, `01A` through `42C`. A bare
number from zero works too, for scripts.

`device current` is the edit buffer rather than a slot, and that is the whole
difference. A live parameter move changes the buffer and writes nothing back, so
reading the slot it came from answers with the stored document and makes it look
as though nothing happened.

## A named slot can still be empty

A device names every slot. An untouched one is called `New Preset`, and a slot
somebody named and then emptied keeps its name. **A name says nothing about
whether anything is in it**, and the two states are identical to anything reading
names alone.

```text
01A   Chunky Monkey    amp → cab
27B   BAS:SVT Nrm      empty
42C   New Preset       empty
```

`42C` is untouched. `27B` has a name and no blocks, and that is the state worth
knowing: a listing that counted names called it "in use" while every read of it
correctly answered "nothing", and the gap between the two read as a bug in
reading. It cost a day.

There are two kinds of empty on the wire, no document at all or a whole document
holding no blocks, and both report as empty here.

## What listing costs

`slots list` reads each named slot, about a second and a half for a device
holding thirty. `--all` includes the untouched ones.

## Working from a backup instead

Every reading command takes `--file`, for a backup HX Edit wrote when no device
is attached:

```bash
mise exec -- go run main.go slots list --file device.hlb
mise exec -- go run main.go presets show --file device.hlb --slot 31A
```

A `.hlb` holds every setlist, which makes it the only file stating everything
the hardware currently holds: one backup is the whole instrument and one restore
puts it back. `--setlist` picks one out of a backup holding several.

## What a read does not carry

A rig read off the device carries its routing, so compiling one puts the device's
own inputs, outputs, split and join back. Controller assignments are the
exception: nothing has decoded them over USB yet, so a rig read from hardware
carries none. Reading the same slot out of a backup carries them.
