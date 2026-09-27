# Switch and rearrange

The pedal plugged in and HX Edit quit, or `--file` to work on a backup instead.

```bash
mise exec -- go run main.go device select --slot 27B          # load it, like a footswitch
mise exec -- go run main.go slots copy --from 01A --to 02A    # 02A becomes a copy of 01A
mise exec -- go run main.go slots swap --from 01A --to 02A    # exchange the two
```

`select` writes nothing: the slot it came from is untouched, so it is the one
command that changes what you hear without changing what the device holds.
`copy` and `swap` write, and keep what the destination held first the way
`import` does. `--from-setlist` and `--to-setlist` reach across setlists.

## Selecting has to be waited for

A select is deferred. The device says it has taken the request and finishes
afterwards, **and it answers other questions while the switch is still in
flight**, so "it answered again" is not "it finished".

Returning early and closing the session leaves the device holding a half-finished
switch, and it settles that by wiping its edit buffer: the preset comes up with
no blocks and no footswitch colours. Asking the device what it is playing, until
it names the preset that was asked for, is the only honest signal. Confirmed on
hardware both ways. Without the wait the colours went; with it they stayed.

So when a select looks like it did nothing, check what `device current` says
before touching anything else.

## Moving a preset is a swap

There is no move command, because a swap already is one. When one of the two
slots holds no preset the exchange is a move: the preset lands in the empty slot
and **then** the slot it came from is emptied, in that order, so a device that
fails between the two leaves the preset in both slots rather than in neither.

A swap of two slots that both hold no preset is refused, and nothing is kept or
written, because there is nothing to move. `copy` fills an empty slot and leaves
the source as it is.

## Rearranging a backup instead

```bash
mise exec -- go run main.go slots swap --file device.hlb \
  --from 01A --to 02A --out edited.hlb
# HX Edit → Restore
```

Whatever the destination held is gone, so the result is written to a new file
rather than over the one it came from. A device backup is often the only copy of
what the hardware holds, which is why `--out` is not optional.

## Moving one control rather than a whole preset

```bash
mise exec -- go run main.go device turn --block 1 --param 3 --value 0.4
mise exec -- go run main.go device current --json
```

A live parameter edit, heard at once, writing no flash. It is the only way to
change one control: a slot given a new document reads back as the new one and
goes on sounding like what it held before.

A block is named by its position in the chain from zero and a parameter by its
position in that model's own list; position is the only thing that identifies
either on the wire, so neither takes a name. The kind of value has to match,
`--value` for a dial, `--choice` for a list, `--switch` for on or off, because a
device does not coerce. The wrong one is refused with the same error a block that
is not there gives.
