# Get it onto the pedal

**HX Edit must be quit.** It holds the USB interface exclusively and nothing
here can claim it while that is running.

```bash
mise exec -- go run main.go device hardware --json
```

## Audition without writing flash

`device play` replaces what is playing and writes nothing:

```bash
mise exec -- go run main.go presets compile --rig rig.yaml --out a.hlx
mise exec -- go run main.go device play --preset a.hlx
```

That is the right way to try something, and it lasts until the next preset is
selected. Prefer it to a write every time, because a slot is flash.

## When it has to be stored

**Scratch writes go to slots 40 and up.** Banks 01 to 10 hold John's own presets
and are not to be touched. Whatever a slot held is backed up before a write
replaces it, and the answer says where that backup went.

```bash
mise exec -- go run main.go slots import --preset a.hlx --slot 42C --json
mise exec -- go run main.go device current --json
```

Anything beyond this, and **anything that writes flash in a loop**, is the
`work-a-device` skill's job: it owns the session rules, the write-completion
protocol and what to do when the pedal stops answering. Those rules cost real
hardware to learn and are stated in one place so they cannot drift. If it is not
installed, say so rather than improvising a write sequence.
