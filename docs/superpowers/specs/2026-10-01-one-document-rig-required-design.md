# One document, and the rig is required

2026-10-01

**Status: the merge is built, 2026-10-02. Three parts of it are not.** The ask
and the rig are one document and the rig is required. The Setup still lives
wherever `--setup` names rather than at a config path, the Plan is still a
document of its own, and there is no `built:` section: a test refuses a contract
field nothing writes, so it waits for `tone tune` to write one. It supersedes
the three-document split in
[ToneSpec is the ask](2026-09-19-tonespec-is-the-ask-design.md), which stands as
the record of why there were three.

## What a person has on disk

Two files, and one of them is written once.

```yaml
schema: ToneSpec
id: mike-dirnt

ask:                       # optional. Why, in the words somebody used.
  subject: { kind: artist, name: Mike Dirnt, band: Green Day }
  genre: [punk, pop-punk]
  words: [{ term: audible-pick-attack, evidence: [...] }]

rig:                       # required. The gear, cited, portable to any Helix.
  instrument: bass
  chain:
    - { role: amp, gear: Ampeg SVT, evidence: [...] }
    - { role: cab, gear: Ampeg 8x10, evidence: [...] }

built:                     # optional, per device, usually absent.
  - device: HX Stomp
    heard: true
    chain: [...]
```

The other file is a Setup at `$XDG_CONFIG_HOME/toneharness/setup.yaml`, read
without being named.

## Why the rig is required

Because the tool already behaves this way and the schema was the only thing
saying otherwise. An ask of a genre and some adjectives is refused unless
something resolves to gear, which is checked in `chainFor` and has been true
since the solver existed.

John put it as: if you cannot define the rig you cannot model it, so what is the
point of guessing. That is the same statement as the refusal, from the other
side.

It also settles what `ask` is for. The three starting points — gear you name, a
player somebody researched, a recording or genre measured — are instructions
while a request is being resolved and provenance once it has been. Keeping them
in `ask` records what was said; the rig records what answered. Nothing re-reads
them to overwrite a chain somebody edited by hand.

## Why there is one document rather than two

The pair was 1:1 in every case that existed: 15 rigs, 15 asks, never diverging.
An ask with no rig beside it is asserted impossible by a test. A rig with no ask
is ordinary, which is the asymmetry that matters — the ask was always an
annotation on a rig, and a separate file for an annotation is a file to keep in
step by hand.

`pair.md`'s stated reason does not survive being read: "folding them would leave
a rig nobody can read without the ask above it" is circular, since it only bites
while they are two documents.

What did survive is the device round trip, and it is the reason `rig` and
`built` are separate sections rather than one. Compiling a rig and compiling a
plan both produce a preset byte for byte:

```
presets compile --rig  → cf37698b…
presets compile --plan → cf37698b…   (same bytes, twice, from the plan alone)
```

A pedal has nowhere to store who a sound is for. So the layer that round-trips
through hardware cannot be the layer that carries the ask.

## Why `built` exists, and why it is usually absent

A preset built by solving is reproducible. Building the same rig twice is byte
identical, and there are `IsDeterministic` tests guarding it. Writing that down
buys nothing.

A preset **tuned against hardware** is not reproducible. `tone tune` pushes a
reference recording through a real pedal in a real room and solves against what
comes back; nobody re-derives that from the rig. So `built` is written when it
carries something a rebuild cannot recover, and `heard: true` marks those.

**This has to be a stated rule rather than a convention.** The committed
reference plan is 368 lines for one device. A 20-line cited rig with two `built`
entries is 95% model identifiers, reviewing a marketplace rig would mean reading
past a device dump, and "YAML somebody can read, write, diff and send to a
friend" stops being true. The 15 core rigs stay the size they are today because
none of them is tuned.

### What each entry has to carry

Three things the first draft of this design was missing, each found by asking
what makes the promise "anyone can build the exact `.hlx`" true:

|                               | why                                                                                                                                                                                                                                                                                           |
| ----------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `device`                      | the ceilings differ. An HX Stomp holds 8 blocks on one path; a Helix Floor holds 29 across two, and carries 670 blocks against 661. A Stomp entry describes a chain that stopped at 8 because that is all the pedal had.                                                                      |
| `tool` and `catalog` versions | the catalog and the corpus statistics ship *inside* the binary. Same rig, newer version, different preset. Determinism holds at a pinned version and nothing else.                                                                                                                            |
| `technique`                   | the mids compensation reads the right hand off the Setup, so `presets make` already produces different presets for different people. An entry would otherwise silently encode one person's hands.                                                                                             |
| `from`, a hash of the rig     | swap the cab in `rig.chain` and a stale `built` still holds the old one. Whoever compiles from it gets the old sound while the file says the new one. The embed drift test exists for exactly this class of bug, and `genres.json` and `hands.json` already use a hash to decide "unchanged". |

## Why the Setup is still its own file

It is per-person, not per-sound: a bass, strings, a pedal, what it plays into.
Put it on each document and the bass is restated twenty times and the twentieth
disagrees with the first, which is `my-setup.yaml`'s own argument for existing.

What changes is that nobody passes it. The project already resolves rigs under
`$XDG_DATA_HOME` and presets under `$XDG_STATE_HOME`; a Setup belongs at
`$XDG_CONFIG_HOME/toneharness/setup.yaml`, read automatically, with `--setup`
left as an override. One file, written once, and `--setup` disappears from every
command anybody types.

## Why the Plan stops being a format

Nobody authors one and no command hands a user one to keep. Of the three
writers, two go to temp directories and the third is `tone tune --out`, whose
consumer is `presets compile --plan`.

A `plan` section would put HX Stomp parameter numbers in a document a Helix
Floor owner reads, which is what the contract already warns against: "a rig that
carried device parameters would not survive being read on different hardware,
which is the whole point of the format."

So `tone tune` writes `rig.settings` — musical terms, 0 to 1, portable — and the
verbatim device numbers live in `built` for the device that produced them and in
the `.hlx` beside it. `presets compile --plan` goes.

**The loss, stated:** a plan holds every device parameter; `rig.settings` is
"deliberately small and deliberately lossy". Tuning keeps what travels and drops
the rest. That is the right trade — a rig that only works on one pedal is not a
rig — and it is a loss.

## Key order is explicit

`schema`, `id`, `ask`, `rig`, `built`. The question before the answer, and the
part nobody reads last.

This needs the writer swapped from `sigs.k8s.io/yaml` to `go.yaml.in/yaml/v3`,
which the project already imports. The current alphabetical order is a side
effect of marshalling through JSON rather than a choice, though the reference
plan's comment rationalises it as one: "Keys are alphabetical, as a rig's are,
so `name` and `rig` sit in the middle rather than at the top." Loading stays on
`sigs.k8s.io/yaml`, which is what feeds the JSON Schema validator.

Every committed document is reformatted by the change, so that lands as its own
commit inside the work.

## The name, and a reservation

**ToneSpec.** It is what the README publishes as the shareable standard and it
is the project's own word. `RigSpec` stops being a document and becomes the name
of a section.

Recorded against it: this names a document whose *required* half is the rig
after its optional half, and this project has a scar from exactly that. RigSpec
version 2 merged a former Recipe and RigSpec because "their names were the wrong
way round: Recipe held the abstraction and RigSpec held compiler output."
Keeping ToneSpec is a decision about what is already published rather than about
what fits best.

## What this does not decide

**`tone build`'s input.** With no ask-only document, nothing can be handed to
it. Either it goes away and resolution happens as the agent writes the file, or
it fills a thin `rig` in place — which weakens "required" from "has gear" to "is
there". Not settled.

## Scale

103 YAML documents carry one of the two schemas. Six Go call sites load them,
plus the loader's stem pairing, the embed, `rigpack`, `slots export`'s output,
`tone build`'s output, every test fixture, and the docs.

The two contracts merge into one, which also removes a duplication the split
created: `Evidence`, `EvidenceKind` and `Confidence` are each defined twice
today, generating two Go types with the same shape and different identities, and
neither package imports the other.

It lands as one pull request with nothing else in it.

## What it partly undoes

[every document says which of the four it is](../../../marketplace/core/examples/README.md)
shipped on 2026-10-01 and renamed the pair to `.rig.yaml` and `.tone.yaml`. That
was the right answer for two files. One file makes the suffix pointless, and the
rename is superseded within a day. Said here rather than quietly reversed.
