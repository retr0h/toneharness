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

Writing the whole layout at once rather than one slot is `slots fill`, and
which slots that touches is [setup.md](setup.md).

### Build it with `presets compile`, not `presets make`

`presets make` builds a preset from a rig's identifier to look at. **A device
refuses its output**, and says so in a way that reads like a broken slot:

    [err] writing slot 14B: device refused the request: opcode 8, error -3

`presets compile` is the one whose help says "a preset a device will load", and
it is what the example above uses. It also takes the flag that matters:

```bash
mise exec -- go run main.go slots export --slot 42C --as hlx --out was.hlx
mise exec -- go run main.go presets compile --rig rig.yaml --template was.hlx --out a.hlx
```

`--template` is "a preset to write the chain into, instead of an untouched
one", which is this page's own rule about everything a chain does not describe.
A preset the device wrote for the destination slot is the template to use.

The two commands also disagree about the chain: `make` adds a compressor the
corpus says 85% of chains have, and `compile` does not. Four blocks against
three for one rig, so they are not two ways of saying the same thing.

### Pick a slot something has been written to before

`~/.local/state/toneharness/presets/` holds a backup of every slot this tool
has ever replaced. The filenames carry the slot, so that directory is the
record of which slots have taken a write:

```bash
ls -t ~/.local/state/toneharness/presets/ | head
# 42C-s0-punk-20261019-045656...hlx
# 42C-s0-mike-dirnt-20261010-043838...hlx
# 42B-s0-slot-42c-20261010-043836...hlx
```

Use one of those. A session that invented a slot instead spent four power
cycles finding out the hard way, and every write that had ever worked on that
pedal had gone to 42B or 42C.

### Read the slot back, every time, before saying it is written

**`[ok] written` is not evidence.** A write has been seen to report that, store
the slot's name, and commit no chain at all: the pedal lists the preset and
plays nothing. Same file, same session, one slot took it and another did not,
and this page cannot yet say why. So the only thing that settles it is asking:

```bash
mise exec -- go run main.go presets show --slot 42C
```

A slot holding a chain prints its gear. One that took the name only prints
`# <name> is empty`, which is the whole fault visible in one line.

This is [claims.md](../../measure-a-device/references/claims.md)'s third rung
and the reason it exists.

### Never retry a slot call that timed out

A slot the device has never held a preset in does not answer a read of it, and
the read times out after six seconds:

    [err] reading slot 40A before replacing it: no reply to opcode 4 within 6s

**The retry is what costs the session.** A second attempt stalls the endpoint,
and from then on every call including `slots list` returns
`kIOUSBTransactionTimeout` until the pedal is powered off and on. One timeout is
recoverable and two is not, so read `slots list` instead of trying again.

The tool no longer walks into it on the common path. A write asks the listing
what the slot is called and does not read a slot the device still calls
`New Preset`, which is 107 of the 126 on an HX Stomp: there is nothing there to
back up, and the preset being written comes from an embedded blank rather than
from the destination. The slot that still times out is one the listing does not
mention, and the timeout now says what silence means and what the next call
costs instead of leaving the opcode to speak for itself.

Those two are not the same claim and the name cannot tell them apart. A slot
somebody wrote in HX Edit and never renamed is also called `New Preset`, and it
holds a real preset. **So a write into a slot by that name keeps no backup**,
which is the trade: losing an unnamed preset against a dead interface. An export
or a show of one is still attempted, because there is no write to protect and
refusing would make an exportable preset unreachable.

### A write says when there was nothing to keep

`kept nothing, the slot held no preset`, rather than leaving the absent `kept`
line to speak. The two read alike and mean opposite things, and a device has no
undo: somebody looking for the way back to what a slot held needs to know
whether there was anything or whether the backup did not happen.

### Clearing a slot is not something this tool can do

Three commands and none of them clears one:

- `slots import` with an empty slot's `.bin` strips the chain and **keeps the
  name**, because a `.bin` carries none
- a name is only ever set by importing an `.hlx`, and that name is the
  ToneSpec's `id`, which the contract holds to `^[a-z0-9]+(-[a-z0-9]+)*$`. So
  `New Preset`, which is what the device calls a slot nobody has written, cannot
  be produced
- `slots copy` and `slots swap` both refuse an empty source, and `slots export`
  refuses an empty slot, so a blank name cannot be lifted off the device either

What that leaves is a slot reading as a named preset with nothing in it, which
is worse than either state on its own. Renaming or deleting it is a job for the
pedal's own menu or HX Edit. Say so rather than leaving somebody to find it.

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
