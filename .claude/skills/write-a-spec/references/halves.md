# Which half a fact belongs in

A ToneSpec is one file with three sections, and two different lines run between
them.

**Between `ask:` and `rig:` the line is scope, not kind.** Why the request was
made belongs to `ask:`. Why this particular piece of gear answered it belongs to
`rig:`, because the rig is what gets published and a published chain that cannot
say why this amplifier is in it is a chain nobody can check.

**Between those two and `built:` the line is who writes it.** Both of the above
are authored and `built:` is generated, replaced whole by every `rigs resolve`.
So the question "which half" has a second half: a fact somebody established goes
up, and a number the compiler worked out goes down. See the skill's own section
on it, which carries the rule and why the sections were split in 4.0.0.

A number you want to pin against the compiler is an authored fact about the
preset, so it goes up, in a chain entry's sparse `controls:`.

## The smallest document that builds

```yaml
schema: ToneSpec
version: "4.0.0"
id: mike-dirnt

ask:
  genre: [pop-punk, punk]
  subject: { kind: artist, name: Mike Dirnt, band: Green Day }
  instrument: bass

rig:
  instrument: bass
  chain:
    - { role: amp, gear: Ampeg SVT }
    - { role: cab, gear: Ampeg 8x10 }
```

`schema`, `version`, `id` and `rig` are required. `ask` is not: gear read off a
device answered nobody's written request, and a document with no ask is the
ordinary shape for one. Neither is `built:` — this is exactly what `rigs new`
scaffolds, and nothing has resolved it yet.

Everything else is optional. `instrument` sits in both halves on purpose: the ask
says which instrument the request is about, the rig says which one the gear is
for. Usually the same value, always two different claims, and folding them would
leave a rig nobody can read without the ask above it.

The ask and the rig were two files until version 2 of the contract,
`<slug>.tone.yaml` and `<slug>.rig.yaml`, paired by filename stem. The pair was
one-to-one in every case that existed and an ask with no rig was already refused,
so the second file was an annotation kept in step by hand.

## How a document is stored

One file per subject under `artists/`, named for the `id` inside it. The
identifier is said once, at the top, and the filename stem is expected to match
it. Two files claiming one identifier are reported rather than one of them quietly
winning.

`rigs new` writes one of yours under `$XDG_DATA_HOME/toneharness/rigs/`, and the
documents that ship are read beside it. One of yours takes the place of a shipped
one when the names match, where a document's names are its own `id` and the
`aliases` in its ask. Which names answer to a subject is a fact about the subject
rather than about the gear, which is why the aliases are in the ask.

A file that is not a valid ToneSpec makes `rigs list` fail and name it, but it
does not stop a shipped rig building. The exception is a file whose name matches
the one you asked for: that gets reported rather than silently passed over.

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

A rig also takes its own `evidence` for a rundown covering the whole chain at
once. A rig rundown names everything in it, and requiring the citation on each
entry would only encourage repeating it.

`note` carries **the sentence the claim rests on, quoted**, not a summary of it.
That is what lets the next reader see whether the source says what the field
claims without opening anything. The kinds a claim may carry are in the
contract's `description:`, strongest first.

## Reading it back

```bash
mise exec -- go run main.go rigs show --id mike-dirnt --json
mise exec -- go run main.go presets make --id mike-dirnt --out mike.hlx
```

The build reports every block it chose, what real gear each emulates, what it
costs, and anything it added the rig did not ask for. A wrong amplifier should be
visible before anybody plugs in rather than after.

A worked document with every field filled in:
`docs/examples/mike-dirnt.yaml`.
