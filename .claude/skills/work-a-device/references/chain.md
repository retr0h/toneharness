# What a chain carries besides blocks

A device lays a chain out on a fixed grid of 20 positions. The first holds the
input, two in the middle hold the split and the join, the last holds the output,
and the device decides where those sit. The other sixteen hold blocks or nothing.

So the position in that array is the block's number and **it is not its place in
the chain**: a preset holding four blocks can have them at 5, 6, 8 and 13. Nor is
it the number the wire answers to, which is one higher again, because the grid
keeps the input at 0. `device turn --block` takes the position and does that
arithmetic itself.

Footswitch assignments address blocks by that number, so renumbering them to 0
through 3 would break the only link between a switch and the block it works on.

Two things travel with the chain that nobody chooses, and leaving either out
produces a preset that looks right and behaves wrongly.

## The switches are part of the chain, and they are the trap

A document counts its blocks from zero along the chain and the wire counts them
across the grid, so the same block is `block1` in a file and 2 on the wire. **A
switch written with the chain index names the block to its left.** On a chain of
drive, amp and cabinet the drive asks for position 0, which no block occupies,
and the switch lights and does nothing; put a compressor in front and the switch
turns the compressor off instead. The symptom is a switch that shows the right
name and refuses to toggle, which reads as a dead switch rather than a wrong
number.

Three claims to keep apart, because the first two have passed while the third
failed:

1. the document carries the assignment
2. the slot reads back with the switch on it
3. the pedal toggles the block when somebody steps on it

`presets show` establishes the first and reading the slot establishes the
second. Only a foot establishes the third, and nothing in this tool can.

**A template's assignments outlive the chain unless they are cleared.** A preset
compiled with `--template` inherits that template's whole footswitch section, so
a chain naming one switch used to arrive with whatever the template had on the
others: a second switch with an empty name, pointing at a block the new chain
never placed. Two switches on one block fight, and stepping on one moves the
other, which looks like a locking switch rather than a leak. Every group the
chain does not name is now emptied, the same way every grid position the chain
leaves out is emptied.

**An export carries them now**, with every field a device writes, so a preset
exported as `.hlx` and re-imported keeps its assignments. It did not until
2026-10-22: the reader found them and the writer rendered nothing, so a round
trip lost every one. HX Edit's own export of a slot is what the section is held
to, field for field.

## A switch carries two names and two colours

Four fields and they are pairs, which this tool collapsed into two.

| field              | what it is                                  |
| ------------------ | ------------------------------------------- |
| `@fs_label`        | the block, up to thirty characters          |
| `@fs_customlabel`  | what the pedal prints, room for about six   |
| `@fs_customcolor`  | the colour somebody chose, 0 to 11          |
| `@fs_ledcolor`     | the colour the block lights with            |

The two labels are task 263 and the default is the kind of block, upper case.

**The chosen colour is an index into the device's own list and the automatic one
is not.** `@fs_customcolor` is a place in `catalog.LEDColours`, which the
catalog reads off the pedal: 0 auto, 2 red, 11 off. `@fs_ledcolor` is a packed
24-bit value, and the bypassed state is the same hue at about a sixteenth of the
brightness.

It was written as zero, which is not a colour. The table is mined from the 4,426
presets in `resources/schemas/corpus/`: across 5,819 assignments in the seven
categories a switch is given, 99.2% agree on one pair per category, and all 313
models that appear resolve to one of them.

    drive   0xd0af00 on   0x080600 off
    eq      0x84ff00      0x080c00
    delay   0x06ff00      0x010900
    mod     0x0032ff      0x00040d
    reverb  0xff2d00      0x100400
    filter  0x3200ff      0x03000b
    pitch   0x3200ff      0x03000b

Three pairs repeat, which is the device colouring by what a block does rather
than by which block it is. A utility block never gets a switch and is the one
category whose readings do not settle, at 58.5% of 573.

Nothing visible changes from it while a chosen colour is always set, and a
chosen one wins. What it buys is a preset that says what the pedal would say.

## The snapshots are part of the chain

Each of the three snapshots carries an array of 20 in step with the grid, holding
whether each position is switched on. **A chain written without them recalls the
wrong blocks the moment anybody presses a snapshot.**

Writing the chain's own state across all three is right for a chain nobody has
snapshots for and wrong for a preset that arrived with three different sounds,
which would come back holding one under the names the blank shipped with. What
gets written instead is what the source recalls: the name, the tempo, the colour,
and which positions each snapshot switches on.

The four routing positions are left alone, because a snapshot records those too
and switching one off recalls the chain with its split bypassed.

### A snapshot the device does not trust breaks the footswitches

Each snapshot also carries `@valid`, which says whether the device believes its
record. **An untouched preset has all three invalid and no block record at
all**, and a preset built here is written into an untouched one, so it inherits
them unless something writes over them.

A pedal loading a preset whose snapshots are invalid has nothing to recall and
initialises them from whatever the chain happens to be. The footswitches then
take three presses before they work: the first goes into that initialisation
and moves nothing, the second leaves the pedal and the block disagreeing, and
the third settles them. After that the switch works for good.

**That symptom points nowhere near its cause**, which is why it is written down:

- the switch is bound to the right block, and reading the preset back proves it
- `@fs_enabled` is `false`, and so is a working preset's, so it is not the
  difference
- the preset comes up edited partway through, which is the snapshot being
  written
- it never recurs for that slot, so it reads as a one-off

So if a built preset's switches need pressing three times, read `@valid` before
anything else:

```bash
mise exec -- go run main.go slots export --slot 37C --as hlx --out /tmp/was.hlx
python3 -c "import json;t=json.load(open('/tmp/was.hlx'))['data']['tone'];print([t[s]['@valid'] for s in ('snapshot0','snapshot1','snapshot2')])"
```

Three `True` is right. Any `False` is this. Compare against a preset the pedal
itself wrote, which carries `True` on all three and a block record per
snapshot.

The protocol side, and what was read off the device to establish it, is
[the wire README](../../../../pkg/sdk/internal/wire/README.md).

## The routing is part of the chain too

A chain is written into the other sixteen positions, so a preset built from a
file keeps the routing of the slot it was built into: an output gain somebody set
arrives at zero, and a split somebody balanced arrives as whatever the blank
carried. What the source says is written instead, each entry found by the kind it
declares rather than by where one blank happens to keep it.

**The values are capped at the length the device itself wrote**, because a device
sends fewer parameters than a model names. An input names seven and sends three,
an output names three and sends two. Writing the model's full list would hand the
device a longer array than it produced.

## One thing that looks wrong and is not

A snapshot's tempo is stored and read back correctly while HX Edit shows the
preset's tempo for all three. That is not a lost value: the device has a global
Tempo Select setting deciding whether tempo follows a snapshot at all. Check that
before going looking for a bug.

## Reading the chain back before trusting an index

```bash
mise exec -- go run main.go presets show --slot 42C --json
mise exec -- go run main.go catalog show --model HD2_AmpSVBeastNrm --json
```

`presets show` lists a chain in order and `catalog show` lists a model's
parameters in theirs, which is what `device turn` addresses. A parameter has no
name on the wire, only a position in the model's own list, so counting down a
table printed for a reader mislabels every control, and the numbers stay
plausible while it does.

An amplifier and its cabinet are one block to a device and two entries in a
preset, which is why `device turn --model 1` reaches a cabinet fused into an
amplifier's slot.
