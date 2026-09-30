# How far to trust a catalog figure

A sweep needs a control's range, and the range comes from the catalog. So does
the DSP cost a chain is budgeted against. Neither is uniformly trustworthy, and
the catalog says which is which.

```bash
mise exec -- go run main.go catalog show --model HD2_AmpSVBeastNrm --json
mise exec -- go run main.go catalog list --category amp --json
```

## Every figure says how far to trust it

Each block and each DSP cost carries a `prov`:

| value      | means                                                                          |
| ---------- | ------------------------------------------------------------------------------ |
| `official` | Line 6 stated it                                                               |
| `observed` | inferred from presets, so the bounds are only what the corpus happened to hold |
| `assumed`  | neither, so a guess                                                            |

This is not decoration. **A chain is never filled with a block whose DSP cost is
`assumed`**: budgeting on a guess produces a rig that validation refuses a moment
later, blaming a block nobody asked for. An `observed` range is honest about
being the corpus's range rather than the device's.

## A catalog is only true of one release

The catalog is generated from HX Edit's own data and ships committed in the
binary, one per device. It records which release it came from, and that matters:
**a catalog is only true of the models one release knew about, and a device on
older firmware may not have all of them.** A catalog that cannot name its source
displays as `source unknown` rather than being presented as authoritative.

**Nobody needs HX Edit to use this project.** Generating a catalog does; using
one does not, which is the whole reason the files are committed rather than built
on demand. `--device` names another pedal's and `--catalog` reads one somebody
generated themselves, with `--catalog` winning when both are given.

## What the device can do is not what people do with it

Line 6 state a default Treble of 0.77 for one amplifier's bright channel; the
median across the presets using it is 0.85. Both are facts, and **the catalog
only knows the first**. A measured starting value comes from the corpus, not from
here.

Two limits before trusting a device other than the one in the room. The corpus
statistics are HX Stomp presets only, so a rig built against another catalog gets
the model table and none of the measured starting values. And **only an HX Stomp
has ever been written to over USB**, so the other three are read from Line 6's
own files and have never been checked against the hardware they describe.

A reading taken on one device is a reading on that device. Say which.
