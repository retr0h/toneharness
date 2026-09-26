# Get it onto the pedal

**HX Edit must be quit.** It holds the USB interface exclusively and nothing
here can claim it while that is running.

```bash
mise exec -- go run main.go devices list --json
```

## The rules that keep a device alive

Read [docs/protocol.md](../../../../docs/protocol.md) before anything that
writes. Three facts that cost real hardware to learn:

**A burst of flash writes corrupts a setlist past what a power cycle clears.**
A device stops accepting writes after about a dozen racing commits.

**A write is not finished when it is accepted.** The reply says the device took
it; completion arrives later as a separate notification. Treating the first as
the end races the next write against a commit still running.

**A stalled endpoint needs a power cycle.** If writes start timing out while
reads still work, the interface will not be claimed again until the pedal is
switched off and on. Nothing in software clears it.

## Audition without writing flash

`presets play` replaces what is playing and writes nothing:

```bash
mise exec -- go run main.go presets compile --rig rig.yaml --out a.hlx
mise exec -- go run main.go presets play --preset a.hlx
```

That is the right way to try something. It lasts until the next preset is
selected.

## When it has to be stored

**Scratch writes go to slots 40 and up.** Banks 01 to 10 hold John's own
presets and are not to be touched. Whatever a slot held is backed up before a
write replaces it, and the answer says where that backup went.

```bash
mise exec -- go run main.go presets import --file a.hlx --slot 42C --json
mise exec -- go run main.go presets current --json
```
