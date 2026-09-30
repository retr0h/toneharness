---
name: measure-a-device
description: Push a known signal through a Line 6 Helix and measure what comes back, so a claim about what a control does can be checked rather than asserted. Covers the reference signal and why it may not change, getting audio into the Input block and out over USB, addressing a block and a parameter by number, sweeping one control through its range, what a refusal means, and how far to trust a catalog figure. Use when asked what a knob actually does, to sweep or measure a block, to measure the noise floor, to check the catalog's parameter order against the hardware, or when a measurement came back implausible.
compatibility: Requires a toneharness checkout with mise available, and an HX Stomp attached over USB with HX Edit quit. Every command runs through `mise exec -- go run main.go`, never a bare `toneharness`.
license: MIT
metadata:
  author: retr0h
  source: https://github.com/retr0h/toneharness
---

# Measure a device

Measuring records is [measure-music](../measure-music/SKILL.md). Reading and
writing slots is [work-a-device](../work-a-device/SKILL.md). This is the pedal
itself: a known signal in, numbers out, and the difference is the device.

## 1. Establish what is true

Every run, before taking a reading. What the commands take, and which figures
come back, are the tool's answers and not this skill's:

```bash
mise exec -- go run main.go measure --help          # which subjects exist
mise exec -- go run main.go measure controls --help # what a sweep takes
mise exec -- go run main.go device --help           # what touches hardware
mise exec -- go run main.go catalog show --model HD2_AmpUSDripmanNorm --json
```

`--help` on a checkout compiles the checkout, so it is the source's own answer
rather than a description of one. Read `cmd/` only to change a command.

Never write a list into this skill. The figures a reading is reported in have
changed, a control's range comes from the catalog rather than from zero to one,
and the device carries 661 blocks that Line 6 rename between releases. Ask the
tool. The one exception is
[the input and output enums](references/enums.md), which are the device's own
firmware constants and which no command can print.

`--json` works on every command. Use it.

## Two surfaces, one SDK

Every command here has a tool beside it over MCP, named after the command:
`device select` is `device_select`, `corpus music genres` is
`corpus_music_genres`. Use whichever the session offers. **The tools are the
same operations, not a reimplementation**: both surfaces call `pkg/sdk` and
answer with the same types, and a test walks the command tree against the
registered tools both ways, so neither can quietly gain a capability the other
lacks.

`.mcp.json` starts the server with `go run`, so it compiles the working tree
every launch and cannot serve a stale binary.

Three commands have no tool, and the reason is in that test's exempt list:
`measure blocks`, `measure controls` and `measure names` are sweeps. One reading
is about eight seconds, so a twelve-control amplifier is most of an hour, and a
tool that blocks that long is not one anybody can use. Run those from a terminal
where the progress shows and Ctrl-C reaches the session holding the pedal.

## 2. Route

| The question                                                 | Read                                                               |
| ------------------------------------------------------------ | ------------------------------------------------------------------ |
| why measure at all, and which loop a question belongs to      | [references/loop.md](references/loop.md)                           |
| what goes into the pedal, and whether it may ever change      | [references/reference-signal.md](references/reference-signal.md)   |
| getting audio in and out, or nothing arriving at all          | [references/signal-path.md](references/signal-path.md)             |
| setting a preset's input or output by number                  | [references/enums.md](references/enums.md)                         |
| which block, which parameter, or what a refusal means         | [references/addressing.md](references/addressing.md)               |
| running a sweep, and what it has to control for               | [references/sweeping.md](references/sweeping.md)                    |
| anything that writes, or a pedal that stopped answering       | [references/device-care.md](references/device-care.md)             |
| a control whose name does not say which way it moves           | [references/unclear-controls.md](references/unclear-controls.md)   |
| how far to trust a catalog figure                             | [references/catalog-trust.md](references/catalog-trust.md)         |
| reporting what was measured                                   | [references/claims.md](references/claims.md)                       |
| all of it, in order                                           | All nine, in that order                                            |

## 3. Say which claim you have

Four claims, and the fourth is the one this exists for:

1. the rig validates against the catalog
2. the device accepted it
3. the hardware loaded it
4. the sound measures where it was aimed

The first two have passed while the fourth failed, on presets that rendered as
an empty chain. Reporting success because a file was written is the failure this
whole project exists to avoid, so a reading is worth nothing until the path it
came down is established. [references/claims.md](references/claims.md) says how.

If you did not run it, say you did not run it. Report in this order:

1. what was asked, in the user's words
2. what the signal path was, and how that was established
3. what the tool answered, including every position it marked silent or refused
4. which of the four claims above you have
