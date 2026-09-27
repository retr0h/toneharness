# A song as sections, and what moves while you play

## Sections are the rig's

```yaml
sections:
  - name: Verse
    bypass: [drive]
  - name: Chorus
    play: [drive]
```

Each section becomes a snapshot, in the order listed, named what you called it.
`play` turns on every block with that role, `bypass` turns them off, and a block
neither list mentions keeps the state the chain gives it.

Roles rather than block numbers, because a hand-written rig has no block numbers
to name. The catch is that two drives in one chain switch together.

The build refuses three things:

- a role the chain has no block for, checked **on the chain as built**, after the
  compiler has added and dropped blocks
- the same role in `play` and `bypass`
- more sections than the device has snapshots, which is three on an HX Stomp

`sections` stay on the rig because a section is what somebody plays. `snapshots`
are what a device stored, and they are the plan's. A build turns the first into
the second, so reading a built preset back gives snapshots rather than the
sections that produced them.

**The fields are not called `on` and `off`** because the YAML reader takes those
words as true and false, so the lists would vanish without an error.

## Controllers are the plan's

An expression pedal assignment names a block number and a device parameter, and
neither exists until the compiler has chosen the models. There is nowhere in a
portable rig to put one.

```yaml
controllers:
  - controller: 2
    block: 1
    parameter: Drive
    min: 0.3
    max: 0.85
    no_snapshot: true
```

The expression pedal reads 2 on an HX Stomp. `block` counts along the signal path
the way the chain counts its entries, so 1 is the second piece of gear. `path`
says which processor that block is on, and leaving it out means the first, which
is the only one an HX Stomp has. It matters on a Helix Floor, where both paths
count from zero and a number alone could mean either.

`parameter` is the control by the name the catalog gives it, checked against the
model that ended up there: a name it does not have is refused with the names it
does. `min` and `max` are what the parameter reads at either end of the travel,
and leaving them out gives the control's full sweep. `no_snapshot` keeps a
snapshot change from moving the assignment out from under your foot.

On a device with two processors the build can lay a chain across both, which
renumbers the blocks. **An assignment moves with the block it names** rather than
being left pointing at whatever took that number.

## A footswitch's colour is not a free choice

`footswitches` is the plan's too, and its block moves with the fit the same way.
Two fields that look alike and are not: `led` is the colour somebody chose, which
a build writes as the number the device files that colour under, and `colour` is
the light the switch shows when nobody chose.

The second is worked out from the block, which is why an untouched preset lights
an amp red and a delay green without anybody setting anything. Measured over
17,665 assignments in the corpus: red for an amp or cabinet, amber for drive,
green for delay, blue for modulation, purple for a filter, pitch or wah, orange
for reverb, lime for a compressor or EQ. Each is one hue at two brightnesses,
bright while engaged and dim while bypassed, which is why a palette covering
twelve categories holds twenty-four values.
