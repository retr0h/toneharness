# Keeping the pedal alive while measuring

A slot is flash, and **flash corrupts rather than merely wearing**. A burst of
writes took a setlist past what a power cycle could clear, and a device stops
accepting them after about a dozen racing commits.

Measuring is the workload most likely to do it. Loading a different chain over
and over is the method: once per block to say what each of 661 sounds like, and
once per control to put a chain back between sweeps. Through a slot write that is
a flash write every time, **for readings nobody wanted to keep**.

## Play what is being tried, import what is being kept

```bash
mise exec -- go run main.go presets compile --rig rig.yaml --out a.hlx
mise exec -- go run main.go device play --preset a.hlx --json
```

`device play` replaces the edit buffer, names no slot, and every slot keeps what
it held. What it replaces lasts until the next preset is selected or the device
is power cycled, at which point the slot's own version comes back.

`slots import` is for a preset somebody wants kept, such as the measuring preset
that carries the input and output the loop needs. It keeps a copy of whatever the
slot held and says where.

## Never overwrite a slot nobody offered

Read `slots list` and write only where something empty is, or where the person
whose pedal it is has said to. Every slot holds somebody's work until shown
otherwise, and there is no undo on the device.

A high bank is the convention here, because low banks are where most people keep
what they play: `TONEHARNESS_SCRATCH_SLOT` names the one the device tests use, and
setting it is how somebody says which slot they are willing to lose. Ask rather
than assume, on a pedal that is not yours.

A misplaced write stalls the endpoint and the pedal needs a power cycle, which is
a bad thing to discover while its owner is out.

## Three more facts that cost real hardware

**HX Edit must be quit.** It holds the USB interface exclusively and nothing here
can claim it while that is running.

**A write is not finished when it is accepted.** The reply says the device took
it; completion arrives later as a separate notification. Treating the first as
the end races the next write against a commit still running.

**A stalled endpoint needs a power cycle.** If writes start timing out while
reads still work, the interface will not be claimed again until the pedal is
switched off and on. Nothing in software clears it.

## Before a long run

`measure blocks` writes everything as it goes, so a run interrupted partway keeps
what it had:

```bash
mise exec -- go run main.go measure blocks --out readings.json --resume
```

`--retry` tries the ones that refused again. Check what is attached first, and
name the pedal explicitly rather than letting a default pick:

```bash
mise exec -- go run main.go device hardware --json
```
