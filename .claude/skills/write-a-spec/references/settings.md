# What a control is set to, and what decides it

```yaml
settings: { drive: 0.47, bass: 0.52, mid: 0.71, treble: 0.85 }
```

A small vocabulary from 0 to 1, each word meaning roughly the same thing on any
amplifier. The contract holds the list and **nothing outside it is allowed**: a
word not on it is refused when the rig is read.

Each word lands on whichever control the model has for it. `drive` reaches a
Drive or a Gain, `level` reaches a Level, a Ch Vol, a Master, a Volume or an
Output, `bass` reaches a Bass or a Low. The 0 to 1 is scaled onto the range the
control is counted in, so 0.5 on a control from −12 to 12 is 0.

**A word the model has no control for is refused**, and the error lists the words
it does take. The Ampeg SVT has no Presence, so a rig asking for one on that amp
does not build.

## Device knobs are not settings

`Sag`, `Bias X`, `Ripple` and `Hum` are one manufacturer's controls. A rig
carrying them would not survive being read on other hardware, which is the whole
point of the format. They belong to the plan, and the compiler sets them from
catalog defaults and corpus medians.

## Four inputs decide a value, in this order

Each may raise on the one before it and none may ignore it.

1. **Line 6's stated default** is the floor and is never invalid.
2. **The corpus median** may raise on it where players agree closely.
3. **A word on the ask** may move it from there.
4. **A number somebody wrote in the rig** beats all three.

That order is not arbitrary: a number is somebody's decision about this gear,
where a word is a description of a result something else has to turn into a
number.

**Where players disagree, the default stands.** An average of disagreement
presented as a measurement is worse than the factory figure. The spread is the
useful column, because it says how much of an opinion is worth having:

```bash
mise exec -- go run main.go corpus presets show --model HD2_AmpSVBeastBrt --json
```

Leaving `settings` out is fine and often better. The four inputs still run, and
the build reports what each one did.
