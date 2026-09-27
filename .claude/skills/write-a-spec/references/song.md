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

## A rig says what a foot reaches, in `moves`

```yaml
moves:
  - by: expression
    role: amp
    setting: drive
    min: 0.3
    max: 0.85
    no_snapshot: true
```

The counterpart of `sections`, and portable for the same reason: a role and a
setting word mean the same thing on any hardware, where a block number and one
manufacturer's parameter name do not.

`by` is `expression` or `footswitch`, **named rather than numbered**, because the
number is one family's: an expression pedal reads 2 on an HX Stomp and a rig
saying 2 would be describing that pedal instead of the music. `role` reaches the
first block in the built chain with it, the way a word on the ask reaches the
first amplifier. `setting` is the same vocabulary `settings` uses, so it lands on
whichever control the model has for the word, and a move and a setting on one
word reach the same knob.

`min` and `max` are over the control's own range, and leaving them out gives the
full sweep. Three things are refused: a role the chain has no block for, a
setting the model has no control for, and two moves claiming the same mover,
because one pedal cannot be on two knobs.

**One caveat that is real today.** A move survives into a preset file and reads
back out of one, but it does not reach a pedal over USB yet: the wire writer does
not write that section. So a built file is right and the hardware will not show
the assignment. Say which of those you have.

## What a plan carries instead

A plan's `controllers` is what a move becomes once a device has been chosen, and
what a rig read off a device carries. A rig may say one or the other, never both:
a move is what somebody wants and a controller is what a device stored, and a
build given both would have to pick one without saying so.

An assignment names a block number and a device parameter, and neither exists
until the compiler has chosen the models.

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
