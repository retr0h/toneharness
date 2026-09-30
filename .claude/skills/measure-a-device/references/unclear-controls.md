# The controls whose name does not say which way they move

Mostly the name says. The catalog holds 641 parameter names across 5,602
controls, and the common ones need no explaining: `Level`, `Treble`, `Drive`,
`Mix`, `Feedback`, `Decay`.

About 650 do not, and they are nearly all the same five: `Sag`, `Hum`, `Ripple`,
`Bias` and `Bias X`. **Line 6 publish no table per model.** The HX Edit manual
documents them once, under *Common Amp Settings*:

| control         | lower                                             | higher                                  |
| --------------- | ------------------------------------------------- | --------------------------------------- |
| `Master`        | less power amp distortion, less from the rest     | more power amp distortion               |
| `Sag`           | *"tighter" responsiveness for metal and djent*    | *more touch dynamics & sustain*         |
| `Hum`, `Ripple` | less heater hum and AC ripple                     | more; *"things get freaky"*             |
| `Bias`          | *a "colder" Class AB biasing*                     | at maximum, Class A                     |
| `Bias X`        | *a tighter feel*                                  | *more tube compression*                 |

Most of what remains is not a tone knob at all but a switch or a placement:
`TempoSync`, `Mic`, `Position`, `Angle`, `Pan`. Those have no direction to find,
which is not the same as nothing to measure: a sweep reads every setting of one
and the loop ranks them, so what is missing is a slope rather than an answer.

## Prose is not a direction, and a sweep is

A control can be swept rather than read about, so the question has an answer
wherever somebody spends the hour. Measured alone on one amplifier's normal
channel, per full turn:

| control | centroid   | level     | what that is                                      |
| ------- | ---------- | --------- | ------------------------------------------------- |
| `Sag`   | −4,316 Hz  | −4.2 dB   | darkens and quietens, straight, and not small     |
| `Hum`   | −69 Hz     | −0.048 dB | under the rig's own wander, so it reaches nothing |

Line 6's prose for `Sag` says nothing about brightness or level. **The
measurement says tighter means brighter and louder**, and it moves the centre of
gravity about a third as far as the Treble knob does. That is the kind of answer
no amount of reading produces.

`Hum` is the other useful kind of answer. Nothing can aim at it, so nothing
should try, and a word that tried would move a control for no reason.

## It is per model, not general

`Sag` on a different amplifier is a different number and needs its own sweep.
Carrying one amplifier's figure to another is the mistake this table exists to
stop: the prose is shared across every amp and the behaviour is not.

```bash
mise exec -- go run main.go measure controls --model <model> --param Sag --json
```

`resources/sweeps/` holds what has been measured. Ask it rather than assuming a
control has been done, and say which device and which model a figure came from.
