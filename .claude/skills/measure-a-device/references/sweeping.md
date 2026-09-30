# Running a sweep

Moving a control and measuring the result is easy. Getting a number that means
anything is where the work is, and four things have to be true. Each was got
wrong first, and each produced numbers that looked fine.

## A slope is not a property of a control

The same Treble into a 4x12 and into a 1x15 are two different numbers. Put a
drive in front and they change again. So there are two artifacts and they are
not interchangeable:

|                  | measured on                                     | reusable                 | cost                                   |
| ---------------- | ----------------------------------------------- | ------------------------ | -------------------------------------- |
| a block curve    | one block, nothing else in the chain            | yes, this is the library | one sweep per control, once            |
| a chain Jacobian | the preset being tuned, at its current settings | no                       | one measurement per control, per solve |

The size of the difference: measured alone, one amplifier's Treble moves the
centroid 10,872 Hz. Measured through its cabinet, in a chain the previous sweeps
had left maxed, **717 Hz**. Both are correct about different things.

```bash
mise exec -- go run main.go measure controls --model HD2_AmpUSDripmanNorm --json
```

That builds the first: a rig holding one block, compiled and played. **Every
sweep records the chain it ran through**, read back off the device with
`device current`, and marks whether that chain held anything else.

## A sweep leaves its control where it finished

A live edit writes nothing back, so a control stays at the top of its range.
Sweep a second control after that and it is measured on a chain the first one
skewed; sweep eleven and the eleventh runs on an amplifier with four controls
pinned at maximum.

Playing the same file again replaces the edit buffer and undoes every move, so
the chain goes back in front of the device before every control. Through
`device play` rather than by reselecting a slot: twelve sweeps would otherwise
spend twelve flash writes putting a chain back. See
[device-care.md](device-care.md).

## Silence is repeatable, so the noise floor cannot catch it

The floor catches a figure that wandered. Two takes of nothing agree to the last
digit, so silence clears it more convincingly than music does.

What comes out is not a null result, it is a confident one. The centroid of hiss
is broadband and reads high, so a control that mutes the chain at one end
reports an enormous move. **2,959 Hz was filed as an amplifier's Drive moving
the centroid; 2,959 Hz is the difference between hiss and sound.** A reading more
than 30 dB under the settled level is now marked `silent` and left out of the
totals, and a sweep with fewer than two positions left refuses to report a curve.

## A reading that moved less than the rig's wander moved nothing

`--takes` is how many takes the floor is measured from, and the floor travels
with the curve. A sweep carries the rig's own repeatability, so anything under it
is reported as no movement. Without that, the last digit of a float looks like a
finding.

**A control's range comes from the catalog**, which is not a nicety: 1,452 of the
device's 4,835 float controls do not run zero to one, and a Simple EQ's Mid Freq
swept 0..1 never leaves its bottom stop and reports as inert.
