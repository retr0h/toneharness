# Who it is for, when it applied, what it was played on

All three are the ask's, because all three are facts about the request rather
than about the gear.

## Say when the rig applied

```yaml
subject:
  era: American Idiot
  years: { from: 2004, to: 2004 }
```

`era` is how a person says it, `years` is the same thing a machine can check.
They matter together because the audio evidence comes from records, and **a
record made outside the period the gear describes measures other gear.** Paul
McCartney's rig names an Acoustic 360, his ask names 1975 to 1979, and his corpus
once held a record cut in 1966, two years before Acoustic built one. All three
were honest and the joins between them were wrong.

```bash
mise exec -- go run main.go rigs records --corpus resources/music/bass --json
```

reads every pair against its manifest and says which records fall outside the
years. A pair stating no years reads `no era to hold them to` rather than passing
quietly, because an unstated period is not a period every record falls inside.

It **reports rather than refuses**, because which half is wrong is a judgement:
the ask may name the wrong period, or the records may be the wrong records, and
only somebody who knows the player can say which.

## Say what it is played on

```yaml
played:
  - gear: Carl Thompson 4-string
    strings: round
    evidence:
      - { kind: cited, url: "…", note: "the 4-string he has played since he was a teenager" }
  - gear: Tune 6-string, de-fretted
    records: [jerry-was-a-race-car-driver, tommy-the-cat]
```

A list, because players use more than one and the figures know it. Les Claypool
took a four and a de-fretted six to the same session.

`records` names which measured records an instrument made, by the track names the
corpus manifest uses. Leave it out where nobody knows. Getting it wrong is not
free: this repository once explained a 162 Hz spread across one session as a
four-string against a six-string, and the source it already cited put both
records on the six.

**No device models an instrument, and every measured figure carries one.** A
fretless played near the bridge is bright before an amplifier is involved.
Without this field those figures read as the amplifier's doing, and a word
derived from them moves a control that was never responsible.

`strings` takes `round`, `flat`, `tape` or `unknown`. Flatwounds against
roundwounds is a larger difference than most pedals make, and `unknown` is worth
saying out loud rather than omitting.

It resolves to no block and changes no preset. It is here so two rigs on the same
amplifier are legible as different sounds.

## One artist, several pairs

A rig is not one thing, and treating it as one is part of why generated tones
sound generic. Five layers, each stable over a different span:

| layer       | stability        | example                 |
| ----------- | ---------------- | ----------------------- |
| instrument  | career-long      | Precision Bass          |
| amp and cab | career-long      | Ampeg SVT into an 8x10  |
| technique   | career-long      | pick, near the bridge   |
| settings    | per era or album | drive amount, EQ curve  |
| effects     | per song         | the octave on one track |

"A Mike Dirnt sound" is the characteristic rig: the stable layers plus median
settings. "The *Longview* bass tone" keeps the same instrument and amplifier and
moves only the settings. **Getting the amplifier right and the drive wrong is a
fixable near miss; getting the amplifier wrong is not.**

So a player's rig changes by era and by song, and two that differ at the
amplifier are siblings rather than variations. Each is a whole document with its own `id`, and
one ask carries `default: true`, because asking for "a Mike Dirnt sound" with no
qualifier has to land somewhere.

### `extends` records lineage, and nothing merges

People expect this to work the other way, so it is worth saying plainly: **a
document that extends another still holds everything itself.** Nothing is inherited,
nothing is looked up at build time, and deleting the parent leaves the child
working. All `extends` does is record where the ask came from, which is what lets
`rigs show` list a rig's variants underneath it.

The reason is that these files are meant to be read. If one held only its
differences, the rig that compiled would not be the rig on the page, and
answering "why is this amp here" would mean opening two files and knowing the
merge rules.

So copy the parent and edit the copy:

```bash
mise exec -- go run main.go rigs new --from flea \
  --id flea-under-the-bridge --kind song --name "Under the Bridge"
```

That writes both files with `extends: flea` on the ask and the parent's `aliases`
and `default` dropped, since those name the parent to whoever asks for it.

**The corrections do not come across**, and that one would do real harm if it
did: a correction is a round of somebody listening to particular gear, so copying
it would attribute a verdict to a rig nobody has heard.

The citations coming across is the point and also the trap. A claim sourced for
one subject is not evidence for another, so **anything you change loses its
evidence with it.** Drop what you cannot stand behind rather than leave a
citation pointing at gear that is no longer there.
