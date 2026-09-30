# Which document a fact belongs in

The line between the two is **scope, not kind**. Why the request was made
belongs to the ask. Why this particular piece of gear answered it belongs to the
rig, because a rig is what gets published and a published chain that cannot say
why this amplifier is in it is a chain nobody can check.

## The smallest pair that builds

The ask says who it is for:

```yaml
schema: ToneSpec
subject: { kind: artist, name: Mike Dirnt, band: Green Day }
instrument: bass
```

and the rig beside it is the gear and an identifier:

```yaml
schema: RigSpec
version: 2
id: mike-dirnt
instrument: bass
chain:
  - { role: amp, gear: Ampeg SVT }
  - { role: cab, gear: Ampeg 8x10 }
```

Everything else on either file is optional. `instrument` sits on both on
purpose: the ask says which instrument the request is about, the rig says which
one the gear is for. Usually the same value, always two different claims, and
folding them would leave a rig nobody can read without the ask above it.

## How a pair is stored

Paired by filename stem in one directory. `mike-dirnt.yaml` is the gear,
`mike-dirnt.tone.yaml` is the ask. **Neither file points at the other**, so
neither can end up pointing at the wrong one, and somebody editing the words has
the gear in the next tab rather than in a parallel tree kept in step by hand.

`rigs new` writes a pair of yours under `$XDG_DATA_HOME/toneharness/rigs/`, and
the rigs that ship are read beside it. A pair of yours takes the place of a
shipped one when the names match, where a rig's names are its own `id` and the
`aliases` on the ask. Which names answer to a subject is a fact about the
subject rather than about the gear, which is why the aliases are on the ask.

A file that is neither a valid rig nor a valid ask makes `rigs list` fail and
name it, but it does not stop a shipped rig building. The exception is a file
whose name matches the one you asked for: that gets reported rather than
silently passed over.

## Where evidence goes

On the claim, never on the document, because the amplifier may come from an
interview and a drive figure from measuring a corpus:

```yaml
- role: amp
  gear: Ampeg SVT
  evidence:
    - { kind: cited, url: "…", note: "Bass Player interview" }
  confidence: high
```

A rig also takes a top-level `evidence` for a rundown covering the whole chain
at once. A rig rundown names everything in it, and requiring the citation on
each entry would only encourage repeating it.

`note` carries **the sentence the claim rests on, quoted**, not a summary of it.
That is what lets the next reader see whether the source says what the field
claims without opening anything. The kinds a claim may carry are in the
contract's `description:`, strongest first.

## Reading the pair back

```bash
mise exec -- go run main.go rigs show --id mike-dirnt --json
mise exec -- go run main.go presets make --id mike-dirnt --out mike.hlx
```

The build reports every block it chose, what real gear each emulates, what it
costs, and anything it added the rig did not ask for. A wrong amplifier should
be visible before anybody plugs in rather than after.

Worked pairs: `examples/tonespec/mike-dirnt.yaml` with
`examples/rigspec/mike-dirnt.yaml` beside it.
