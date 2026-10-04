# Iterate on a rig with the pedal in front of you

The loop this project is for. A solver gets close; a person turning a knob
decides. What follows keeps that decision instead of losing it, and does it
without writing to flash until there is something worth keeping.

## Live first, flash last

The device has two ways to take a chain, and the difference is hardware rather
than taste.

| what | opcode | reaches flash | CLI |
| --------------------------------- | ------ | ------------- | ---------------- |
| replace what is playing | 21 | no | `device play` |
| move one control on what is playing | 30 | no | `device turn` |
| read back what is playing | 22 | no | `device current` |
| write a slot | 9 | **yes** | `slots import` |

**Iterate with 21 and 30. They store nothing.** A preset played this way is
gone on the next power cycle, which is exactly what you want while deciding.

**A slot write is a flash write and they are not free.** Each one commits for
about 750ms, and a burst of them is the one thing in this project known to hurt a
device: it took a setlist past what a power cycle could clear, and a device stops
accepting writes after about a dozen racing commits. Saying which of 661 blocks
belongs in a chain through slots would be 661 flash writes for readings nobody
wanted. So write a slot when you have an answer, not while looking for one.

Never write to a slot holding something you did not put there. `slots list`
first, pick an empty one, and prefer a high bank well away from whatever the
owner of the pedal plays. `slots import` overwrites without asking and the device
has no undo, so what was there is read and kept in the backup directory first —
which is a recovery, not a reason to be careless.

## The loop

| step | command |
| ------------------------------ | --------------------------------------------------- |
| write every control down | `rigs resolve --id <id>` |
| build it | `presets make --id <id> --out /tmp/<id>.hlx` |
| play it without storing it | `device play --preset /tmp/<id>.hlx` |
| change one thing and listen | `device turn --block <n> --param <name> --value <v>` |
| read back what it is playing | `device current` |
| keep it | `slots import` into an empty slot, then `slots export --as tonespec` |

Everything up to "keep it" leaves the device as it was.

## Resolve first, always

`rigs resolve` is what makes the rest work. It builds the rig and writes the
chain back with every control at the value it was given, so the document says
what the preset is.

Without it a rig names gear and seven musical words, the compiler answers the
other six hundred controls from corpus medians, and nothing records what they
were. Move a microphone and there is no field for it to come back to.

It also writes down the blocks the compiler added, each labelled `kind: corpus`.
A rig that named an amplifier and quietly became five blocks could not be read as
a description of its own preset.

## Turning a control

Two refusals worth knowing, because both are clear and immediate:

A control the model does not have is refused and the message lists the ones it
does take. A value past the end of a control is refused with the range, as in
`parameter "LowCut": 99999 out of range [19.9, 500]`.

A microphone is a choice rather than a sweep. `mic` on a cabinet is an index, so
there is no "more" to ask for: step through them and listen. Nothing here can
tell you that microphone 5 is a 421 rather than a 57, because the catalog carries
the numbers and no names for them.

## Keeping it

Saving what you tuned live costs exactly one flash write, and the reason is a
gap: opcode 71, which saves the edit buffer to a slot, is not implemented, and
`device current` cannot write a document. So the route is `slots import` to an
empty slot, then `slots export --slot <n> --as tonespec --out <id>.yaml`.

Every control comes back into the field it went out of, so `git diff` against the
rig you started from says exactly what changed and nothing else. That diff is the
thing worth having: it is how somebody else takes the change, and how you will
know in a year why that cabinet has its low cut at 120Hz.

## What a changed value claims

A value turned by ear is a stronger claim than one a solver chose, and the
document should say which. Mark the entry's evidence `kind: heard` with a note
saying what you were listening for.

Nothing re-derives it afterwards. The ask's words and `like` are read once, when a
rig is first resolved, and the contract is explicit that nothing re-reads them to
overwrite a chain somebody edited by hand. So "punk, punchy, like Dirnt" will not
come back and undo a microphone.

If you want the solver's answer again, resolve into a new file and compare.
Asking for a rebuild over the top of hand work is how the work goes missing.

## A variant rather than an edit

Wanting the same rig a bit different is a second rig, not a mutation of the
first. `ask.extends` names the one it departs from, and the contract says asks
differing at the amplifier are siblings rather than deltas.

Use it when the change is a judgement somebody else might not share: a song
played differently from the rest of a set, or a cabinet one player prefers. One
person's tweaks stay in their file and the original stays intact.
