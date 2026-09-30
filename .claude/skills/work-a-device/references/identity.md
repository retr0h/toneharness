# Which device, and which catalog answers

## Two identifier systems, deliberately separate

A device answers to a **USB `vid:pid`** on the bus and is named by a different
integer inside a preset's own device field. They are unrelated numbers and both
are needed: one finds the hardware, the other says which hardware a preset is
for. A preset for one device will not load on another.

Ask for both rather than remembering either:

```bash
mise exec -- go run main.go device hardware --json
mise exec -- go run main.go presets show --preset a.hlx --json
```

A device missing from the model table is still reachable over USB and shows up on
the bus. It will not be named, and presets cannot be written for it. That is the
failure to expect from a pedal nobody has added yet. It is seen and not
understood.

## Which catalog answers

Every command that has to resolve a model takes two flags that overlap:

| Flag        | Means                                                |
| ----------- | ---------------------------------------------------- |
| `--device`  | use the built-in catalog for that pedal               |
| `--catalog` | use a catalog generated from an HX Edit installation  |

**`--catalog` wins over `--device`.** A catalog somebody generated themselves is
a stronger statement than the name of a device this binary happens to carry, so
passing both is not an error and the generated one is what answers.

Use `--catalog` whenever the pedal is on a firmware newer than the release this
binary was cut from. Model numbers are positions in a table that firmware can
extend, and a catalog generated from the installed HX Edit is the only thing in
step with the pedal in front of you.

## Only one device has ever been written to

Writing over USB has been verified on an **HX Stomp** and nowhere else. The other
three devices in the table are read from Line 6's own files and have never been
checked against the hardware they describe.

That is a real limit on what you may claim. Reading and rearranging a backup for
a Helix Floor is the same code path and is fine to report as done; telling
somebody a preset was written to their Helix LT over USB is a claim nothing here
supports. Say which device answered.

## Setlists

A slot label is only unique inside a setlist. An HX Stomp answers with 126 slots
and a Helix or HX Stomp XL with 128, so the count is something to read rather than
assume. `--setlist` picks one out of a backup holding several, `--from-setlist`
and `--to-setlist` let a copy or a swap cross between them, and `device select`
takes `--setlist` for the same reason.
