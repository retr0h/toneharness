# How toneharness is put together

**If you read one file to understand this system, read this one.** It is written
to be that: how the whole thing works and how the parts fit, front to back, from
a request arriving to a knob moving on a pedal and somebody saying it is still
wrong.

What it covers, in order: why the system exists rather than copying presets, the
three layers a request passes through, the pipeline that turns one into a
preset, the three ways in, the loop a person sits inside, what runs on real
hardware, what the system deliberately cannot do, and what is not built yet.

Three places, three jobs, and this page is the way in to the other two:

| For                              | Read                                                                      |
| -------------------------------- | ------------------------------------------------------------------------- |
| how to *do* something            | [the skills](../.claude/skills/), which are the authority on their domain |
| why something is shaped this way | [the design records](superpowers/specs/), dated and never updated         |
| what it is and what works        | this page                                                                 |

A skill is evergreen and gets updated when the code changes. A design record is
history: it says what was decided on a date and is left alone afterwards, so it
is the wrong place to learn how anything works today.

## Constructing is not copying

The corpus holds a preset called "Basket Case". Copying it would inherit one
person's opinion, including their mistakes, and would answer nothing for a
player nobody has made a preset for.

The goal is a system that knows *how a chain is built*. That decomposes into six
problems with six different sources, and conflating them is why generated tones
come out generic.

| Problem                   | Source                                       | State                                       |
| ------------------------- | -------------------------------------------- | ------------------------------------------- |
| Who plays what            | `pkg/sdk/shipped/`, a hand-written pair each | thin, grows by correction                   |
| Gear to model ID          | `resources/schemas/gear-map.json`            | 575 models                                  |
| What order blocks go in   | statistics over `resources/schemas/corpus/`  | added blocks placed; a rig's own order kept |
| Which way a knob moves    | swept on the device, in `resources/sweeps/`  | eleven blocks measured and shipped          |
| What values to set        | catalog defaults, corpus medians, intent     | six axes of ten                             |
| What a genre sounds like  | displacement over `resources/music/`         | 3 tagged, 2 earning a word                  |
| What a player sounds like | measured over `resources/music/bass/`        | 16 players, 52 records, 6 earning a word    |

**Keep this table honest.** A pull request that finishes something marked not
built or partly built updates the line in the same pull request and says so in
its description. This table fell three features behind when nobody did, and it
is how the next session learns what exists.

## Three layers

|              | holds                                        | written by                   |
| ------------ | -------------------------------------------- | ---------------------------- |
| **ToneSpec** | what somebody means, and why it is believed  | a person                     |
| **RigSpec**  | gear in signal order, named as a person does | a person, or `tone build`    |
| **Plan**     | that rig realised on one device              | the compiler, never a person |

All three exist. A rig is portable because there is no longer anywhere in it to
put a Helix answer: the resolved model per block, the position, the snapshots
and footswitches, and the device state a lifted preset arrived with all live on
the Plan.

Only the first two have contracts, and the rule is worth stating: **becoming a
file is not what earns a contract. Being typed by somebody is.**

[A rig is a plan, for one device](superpowers/specs/2026-09-19-a-rig-is-a-plan-for-one-device-design.md)
is the record for that split, and
[ToneSpec is the ask](superpowers/specs/2026-09-19-tonespec-is-the-ask-design.md)
says how the first two divide, both superseding
[the RigSpec design record](superpowers/specs/2026-09-06-rigspec-as-the-one-model-design.md).

## The pipeline

```text
request      "a Mike Dirnt sound"
   │
   ▼
the ask      pkg/sdk/shipped/artists/mike-dirnt.tone.yaml     who it is for
   │         words: scooped, clean — or genre: grunge
   ▼
the rig      pkg/sdk/shipped/artists/mike-dirnt.yaml          who plays what
   │         amp: Ampeg SVT
   ▼
gear map     resources/schemas/gear-map.json                  gear to model
   │         HD2_AmpSVBeastNrm
   ▼
catalog      pkg/sdk/catalog/data/hx-stomp.json.gz            what the device accepts
   │         Drive 0.0–1.0, default 0.53, DSP 26.67
   ▼
grammar      pkg/sdk/corpus/data/hx-stomp.stats.json.gz       what a chain almost always holds
   │
   ▼
genres       pkg/sdk/audio/data/genres.json                   what a genre is displaced on
   │         grunge: scooped, clean
   ▼
values       corpus medians + the ask's words + the genre's   what to set
   │
   ▼
RigSpec      validated against the catalog
   │
   ▼
.hlx         written, and put on a device over USB
   │
   ▼
a person     listens, and corrects the pair                   nothing above can hear
```

## Ways in

One library, three surfaces over it, and nothing behind any of them that the
others cannot reach.

| Surface | What it is                                                        |
| ------- | ----------------------------------------------------------------- |
| **SDK** | `pkg/sdk`. Every operation lives here. The other two only call it |
| **CLI** | `toneharness`, one namespace per noun, `--json` on every command  |
| **MCP** | the same operations as tools, named after the commands            |

`device select` is `device_select` and `corpus music genres` is
`corpus_music_genres`. They are the same code rather than two implementations,
and a test walks the command tree against the registered tools both ways, so
neither surface can quietly gain a capability the other lacks.

Three commands have no tool on purpose, and the same test holds the list of
reasons: `measure blocks`, `measure controls` and `measure names` are sweeps.
One reading is about eight seconds, so a twelve-control amplifier is most of an
hour, and a tool that blocks that long is not one anybody can use.

## The loop a person is in

The pipeline above builds a first answer. Everything after it is correction, and
this is the part that needs a person in it.

```text
build      tone build, or presets make from a curated rig
   │
   ▼
play       device play — opcode 21, replaces what is playing, writes no flash
   │
   ▼
listen     a person. Nothing above this line can hear
   │
   ▼
say        "darker", "needs more bite" — a nudge, which is a direction and a size
   │
   ▼
ask        tone reach — one pass, nothing applied: how near can this get?
   │
   ▼
solve      tone tune — compares the lists, solves the dials, measures again
   │
   ▼
keep       a plan, which is the only layer with room for a knob position
```

A nudge is not a target and the difference matters:
[correcting.md](../.claude/skills/build-a-rig/references/correcting.md) is the
whole of it, including how the solve works, what a tolerance is, what the noise
floor is for, and what the loop does when the chain cannot get there.

The `reach` step is there because the expensive step can fail. A tuning run is
five minutes of real-time audio and it applies every move it solves for, and it
can spend all of that to report that the chain will not get there. `reach` reads
the same chain once, applies nothing, and says how near the model thinks it can
get.

It reads the chain rather than `resources/sweeps/`, because a slope measured
with a block alone is not the slope that block has in a chain: four of eleven
controls on one amplifier move the centroid the other way.

Read what it says with care. On an HX Stomp the empty loop is healthy and a
chain holding an amplifier is not, and
[correcting.md](../.claude/skills/build-a-rig/references/correcting.md#neither-number-is-trustworthy-while-the-loop-adds-a-signal-of-its-own)
has the measurements. Until that is settled every hardware figure for a chain
with an amplifier in it is suspect.

Two hardware facts shape the loop. A slot is flash and a burst of writes has
corrupted a setlist, so tuning happens in the edit buffer and only the final
answer is written. And a live edit stores nothing, so what the device holds
drifts from every slot until it is exported.

## A chain this tool builds renders on the hardware

True since 27 September 2026 and not before. A block body carries five keys, the
device seeks to where it keeps the model rather than walking the map, and this
tool wrote that key last instead of first. Every chain it built was stored, read
back byte for byte, and rendered as no blocks at all.

What holds it now is a byte comparison rather than a read-back:
`TestAChainIsWrittenTheWayTheDeviceWroteIt` writes a capture's own chain back
into the document it came from and requires the same bytes. The cause and the
near-miss that follows it are in
[the wire README](../pkg/sdk/internal/wire/README.md#a-block-bodys-keys-go-in-the-devices-order).

The loop that hears it back broke the same day, separately, and the pair is
worth reading together because neither failed. A device is opened at its own
channel count, an HX Stomp presents eight, and the loop still strode a frame
buffer by two: a quarter of the signal went out smeared across channels instead
of forward in time. That measures 11,990Hz at -46dB where the empty loop
measures 95Hz at -21dB, so every block in a campaign came back with the same
wrong figure looking like data.

**Neither bug failed. Both measured.** That is the shape of defect this project
has to expect, and it is why the checks now live in
[signal-path.md](../.claude/skills/measure-a-device/references/signal-path.md):
read the empty loop before believing any sweep, and if it is wrong, use a block
that generates rather than processes to say whether the send or the return is
the dead leg.

So the correction loop below is now actually cheap, and the measuring that
[solving for knob positions](superpowers/specs/2026-09-27-solving-for-knob-positions-design.md)
needs is unblocked.

## Nothing here can hear

No part of this system can judge whether a preset sounds right, and no quantity
of corpus data changes that. **The evaluator is a person**, and the architecture
assumes it in three places.

**The correction loop must be cheap.** Generate, push to the device, listen, fix
one line, never hear that mistake again. That is why the device library matters
more than more corpus: it removes a manual HX Edit import from every iteration.

**Decisions must be inspectable.** A generated rig records why each block was
chosen and how confident that choice was, so a wrong amplifier is visible before
anybody plugs in rather than after.

**Corrections must be permanent.** A fix belongs in the files, where it outranks
generated knowledge for good.

## A word is earned against a population, so it is not permanent

The corpus went from nine players to fifteen and six words stopped being earned.
`scooped` and `clean` came off Mike Dirnt's ask, `scooped` off Pino Palladino's,
and three more that survived on a citation had their measurement corrected to
say the records no longer support them. Pino's `scooped` margin had been the
narrowest in the corpus at 0.1%, with a note saying one more player would take
it. Six arrived and it did.

That is the population doing its job rather than a wobble in it. **A word whose
only evidence was a comparison against nine people is a word about those nine**,
so the rule is that such a word goes when the comparison stops supporting it,
while a word with a citation behind it stays and the measurement beside it is
corrected to say what it now says.

## What is still missing

The step from a word to a value is half built: a term says which way to move a
control, and where it carries the figures that earned it the distance follows
the gap. A word nobody measured still moves a fixed step.

Nothing above the person listens, so whether half a step of drive is the right
amount of drive is a question no measurement here answers. The method for
closing that is designed and not built:
[solving for knob positions](superpowers/specs/2026-09-27-solving-for-knob-positions-design.md).
