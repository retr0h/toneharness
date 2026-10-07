# Take somebody else's preset, and give them yours

A document, not a file, is what travels. The `.hlx` is what a device loads; the
document is what somebody can read, argue with and change one line of.

## Theirs, into something you can edit

```bash
mise exec -- go run main.go slots export --file theirs.hlb --slot 3 \
  --as tonespec --out theirs.yaml
mise exec -- go run main.go presets make --rig theirs.yaml --out theirs.hlx
```

Works with no pedal attached. `--file` takes the `.hls` setlist or `.hlb` backup
HX Edit writes; `--slot` takes it off an attached device instead.

What arrives is the chain named the way a person names gear, every control at the
value it was set to, and everything the preset holds beside the chain under
`preset:`. What survives the trip back is in
[write-a-spec's preset.md](../../write-a-spec/references/preset.md); what does not
is in
[work-a-device's formats.md](../../work-a-device/references/formats.md#out-and-back).

**Read the chain before trusting the gear names.** A lift names a block by what the
catalog says it emulates, so a model Line 6 describe as nothing in particular comes
back as its identifier. That is honest rather than broken: `HD2_AppDSPFlowJoin` is
the join and there is no friendlier name for it.

## Yours, for somebody else

```bash
mise exec -- go run main.go rigs resolve --id <id>
```

That is the whole of it, and it is the same command the iterating loop opens with.
A resolved document builds the same preset on any Helix: catalog-independent,
because every block and control is stated, and corpus-independent, because the
blocks the corpus added are written in and labelled `kind: corpus`.

Checked rather than asserted: building a resolved rig against a Helix Floor instead
of an HX Stomp changes one value, the device identifier.

**Send the document, not the `.hlx`.** The `.hlx` is one pedal's answer and reads
as nothing to a person. The document carries why each piece of gear is believed to
be there, which is the half somebody else cannot reconstruct.

## What a handoff cannot carry

**An impulse response is a file, not a setting.** A block naming one resolves and
the table of names comes across, and the audio does not: it lives on the device,
loaded by its owner. A preset using one is a preset that needs that file.

**A block the other pedal has no model for.** An HX Stomp cannot build a stereo
wah or FX loops three and four, and says which it could not realise rather than
substituting. 14 of a 403-preset sample are that, and every one was made on other
hardware.

**Whether it sounded right.** A document is somebody's decisions, not their room.
Every measured figure in it was taken through one chain on one computer at one
output level, and `resolved:` records which. Re-tune rather than trust.

## Say which rung you reached

The document validates, the preset builds, and neither says a device took it. The
third claim needs a pedal attached and `device current` read back afterwards,
because a chain that is stored is not a chain that rendered: for a fortnight every
preset this tool wrote read back byte for byte and drew nothing.
