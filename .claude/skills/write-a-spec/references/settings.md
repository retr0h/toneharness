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

**The report says what the preset holds, not what each input wanted.** Where a
number beats a word the line says so and shows both:

```
says grit-on-attack — Drive is 0.88, which the document states: the word would have made it 0.72
```

It printed only the word's move until 2026-10-14, which made it wrong by
whatever the stated value was: "Drive 0.60 to 0.72" against an 0.88 in the file.
That is the one job the line has. Somebody hand-editing a document reads it to
check their edit took effect, and it told them it had not. It was caught by
dumping the `.hlx` and comparing, which is not a check anybody should have to
run.

A word beaten this way is still printed rather than dropped. A word that moved
nothing because a number outranked it is exactly the kind of decision taken on
somebody's behalf that has to be visible.

## A block the corpus adds, and whether it arrives switched on

Two questions, and one number was answering both: would a player have this
block, and would they use it. The corpus disagrees about them sharply. Of the
chains that hold one, these run it:

```
bass     cab 97.0%  gate 88.9%  comp 87.0%  eq 83.3%  reverb 67.4%
         delay 52.0%  drive 48.6%  filter 45.5%  mod 27.8%  pitch 26.1%  wah 0.0%
guitar   cab 95.6%  reverb 92.1%  comp 84.3%  eq 84.3%  gate 82.1%
         delay 61.6%  mod 48.1%  drive 47.3%  filter 38.8%  pitch 38.6%  wah 3.8%
```

**Nothing falls between 61.6% and 82.1%**, which is where the line goes. Above it
are the conventions, below it the pedals. A guitar wah is the clearest: present
in one chain in six and running in one in twenty-six.

So a block clears one gate to be added and a second to be switched on:

| gate | at | what it answers |
| ---- | -- | --------------- |
| presence | 75% of chains hold one | should this be in the chain |
| engagement | 75% of those run it | should it be doing something |

A block clearing the first and not the second **arrives bypassed**. It costs the
chain its DSP and nothing else, the device's own footswitches toggle it, and the
player decides.

```
added Simple Delay — almost every chain carries one and most leave it off, so
      this arrives bypassed (77% of chains carry one, 62% of those run it)
```

**This was wrong on the guitar side until 2026-10-17** and nobody noticed,
because the figure that would have shown it was not counted. A guitar build
arrived with a delay and a drive switched on at 61.6% and 47.3% engagement, so
the gate decided the sound for the player in exactly the way the presence gate
exists to prevent. Bass was unaffected: a compressor reads 87.0% and stays on.

Both figures are measured per instrument and not per genre, because the preset
corpus carries no genre: a preset holds a `build_sha` and a name. The genre
figures come from records, which are a different corpus and say nothing about
what blocks anybody put in a chain.

A block a **word** asks for still arrives on, whatever these figures say. A block
seated because somebody described a sound is a block they want doing something;
the corpus adding one on a statistic is the other case.

Statistics measured before engagement was counted read as nothing running, so a
block from them arrives bypassed. A block switched off changes no sound, which is
the direction to be wrong in.

**`presets compile` resolves no words at all, and says so.** It builds the `rig:`
half, which is the layering rather than a gap: resolving words needs the corpus
and a second chain builder, and `presets make` is that. What it would not do
until 2026-10-14 is admit it:

```
note this builds the rig, so the ask's 2 words moved nothing: sub-octave, grit-on-attack.
     `presets make --rig` is what resolves them.
```

Handing a whole document to the command that uses half of it is an easy mistake
to make and was an impossible one to see. Working it out from the two commands'
help cost a session.
