# Get it onto the pedal

**HX Edit must be quit.** It holds the USB interface exclusively and nothing
here can claim it while that is running.

```bash
mise exec -- go run main.go device hardware --json
```

## Audition without writing flash

`device play` replaces what is playing and writes nothing:

```bash
mise exec -- go run main.go presets make --rig rig.yaml --out a.hlx
mise exec -- go run main.go device play --preset a.hlx
```

`presets make` rather than `presets compile`, and the difference is the ask.
Compile lowers the gear and nothing else, so the words in a document's ask and the
words its genre earned reach no control. Make reads the ask out of the same file
and resolves them. Use `compile` for a document with no ask in it, such as one
lifted off a device.

That is the right way to try something, and it lasts until the next preset is
selected. Prefer it to a write every time, because a slot is flash.

## When it has to be stored

**Write only to a slot that is empty, or one its owner named.** `slots list` says
which are free and `TONEHARNESS_SCRATCH_SLOT` is how somebody names the one they
are willing to lose; a high bank is the convention only because low banks are
where most people keep what they play.

Whatever a slot held is backed up before a write replaces it, and the answer says
where that backup went.

```bash
mise exec -- go run main.go slots import --preset a.hlx --slot 42C --json
mise exec -- go run main.go device current --json
```

Anything beyond this, and **anything that writes flash in a loop**, is the
`work-a-device` skill's job: it owns the session rules, the write-completion
protocol and what to do when the pedal stops answering. Those rules cost real
hardware to learn and are stated in one place so they cannot drift. If it is not
installed, say so rather than improvising a write sequence.
