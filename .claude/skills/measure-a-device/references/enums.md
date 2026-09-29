# The input and output enums

**The exception to asking the tool.** No command prints these. The catalog holds
the 661 blocks a chain is made of, and the Input and Output blocks are the
device's own fixed structure rather than catalog entries, so `catalog show` on any
model answers with that model's parameters and never with these. They live in HX
Edit's own resources, at
`/Applications/Line6/HX Edit.app/Contents/Resources/HelixControls.json`, under
`input_type` and `output_type`.

Reproduced here because a machine somewhere will not have HX Edit installed, and
a preset cannot be built for measuring without them.

| `@input` | Input block source          |
| -------- | --------------------------- |
| 0        | None                        |
| 1        | Multi (Guitar, Aux, Variax) |
| 2        | Guitar                      |
| 3        | Aux                         |
| 4        | Variax                      |
| 5        | Variax Magnetics            |
| 6        | Mic                         |
| 7-10     | Return 1 to Return 4        |
| 11       | Return 1/2                  |
| 12       | Return 3/4                  |
| 13       | S/PDIF                      |
| 14       | USB 3/4                     |
| **15**   | **USB 5/6**                 |
| 16       | USB 7/8                     |

| `@output` | Output block destination              |
| --------- | ------------------------------------- |
| 0         | None                                  |
| 1         | Multi (1/4", XLR, Digital, USB 1/2)   |
| 2-4       | Path 2A, Path 2B, Path 2A+B           |
| 5         | 1/4"                                  |
| 6         | XLR                                   |
| 7, 8      | Send 1/2, Send 3/4                    |
| 9         | Digital (S/PDIF, AES/EBU, or L6 LINK) |
| 10        | USB 1/2                               |
| 11        | USB 3/4                               |
| 12        | USB 5/6                               |

## The indices are the family's, not the device's

The same file carries `input_type_lt` and `output_type_lt` for a Helix LT and
`_native` for the plugin. An HX Stomp uses the unsuffixed pair, which lists four
Returns it has no sockets for and an XLR it does not have.

So an entry existing does not mean the socket does. Entry 1's label is the one
that has already cost an evening: see
[the output block does not mean what its label says](signal-path.md#the-output-block-does-not-mean-what-its-label-says).

## Setting them

Take a preset off the device, change the one number, put it back, and read it
back rather than trusting the write:

```bash
mise exec -- go run main.go slots export --slot 42C --as hlx --out 42C-before.hlx
# set data.tone.dsp0.inputA.@input to 2, and outputA.@output to 10
mise exec -- go run main.go slots import --preset 42C-reamp.hlx --slot 42C --json
mise exec -- go run main.go slots export --slot 42C --as hlx --out 42C-after.hlx
```

`import` keeps its own copy of whatever the slot held and says where. It is a
flash write, so use it for the measuring preset and nothing else. Everything
being tried goes through `device play`, which
[device-care.md](device-care.md) explains.
