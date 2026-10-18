# Naming gear, and putting it in order

## Name it the way a person would

`gear: Ampeg SVT`, never `HD2_AmpSVBeastNrm`, and never `Ampeg SVT Brt`.

Line 6 rename every model for trademark reasons. Naming the identifier would
tie a rig to one manufacturer, break when they rename a model, and stop it being
read on other hardware, which is the whole reason the format exists.

**The model's name is as wrong as its identifier, and it is the one that gets
written.** `Cali 400 Ch1` is this manufacturer's label for a MESA/Boogie Bass
400+ and `Agua Sledge` is theirs for an Aguilar Tone Hammer. Twelve shipped
documents read that way, because two writers disagreed: the compiler names a
block it adds by the gear it emulates, so a cabinet came out `Legacy 6x10"
MESA/Boogie Power House`, while the chooser that picks an amplifier by
measurement named it off the measured library, which files a sweep under Line
6's name. A test now refuses it, by asking whether the name says something other
than the gear underneath it: the catalog's own entry for an Ampeg B-15NF is
named "Ampeg B-15NF", and that is the gear, not a label for it.

Where two models claim one gear name, name the gear anyway. `Teletronix LA-2A`
is both an LA Studio Comp with six controls and a legacy DM4 Tube Comp with one,
and the catalog sorts a legacy model last, so the name reaches the model with the
controls. Reaching for the model name to disambiguate is how a document stops
being readable anywhere else.

Check the name resolves before trusting it:

```bash
mise exec -- go run main.go catalog list --search ampeg --json
```

Nothing back means the device does not model that gear. Name something it has,
or give the entry a `substitute` saying what to use instead and why. That is the
one place a rig admits the device cannot do what it names, and saying so is
honest where inventing a model identifier is not.

On the ask, `insist: true` on a gear entry means refuse rather than substitute.

## Order is the signal path

`chain` is ordered and the order is what the signal does. Drive ahead of an
amplifier overdrives its input; drive after it is a different sound entirely.

You do not have to guess the conventional order, because it is measured:

```bash
mise exec -- go run main.go corpus presets chains --instrument bass --json
```

Ask rather than assume the figures. They come off real presets and they move as
the corpus grows. A rig omitting something near-universal gets it added during
the build, and **the build says so rather than doing it quietly.**

## On the ask, order is not what you typed

Gear on a ToneSpec is sorted into the ordinary signal path, because listing gear
is not stating an order: a compressor belongs in front of the amplifier whichever
way round somebody wrote it. Where they mean otherwise, `after: amp` on an entry
puts it behind that role.

Named by **role rather than by position**, because a request does not know how
many blocks the chain ends up with. "After the amp" survives the compiler adding
a cabinet and "position 4" does not.

A drive behind the amplifier is the case worth knowing about. It is a known way
to use one rather than a mistake.

## A player's instrument is not their amplifier's

`instrument:` on a rig is what somebody plays. It is not derived from the
amplifier, and the two genuinely disagree.

Dusty Hill played bass through a Marshall Super Lead 100, which is a guitar
head: ZZ Top ran the same Marshall rig for his bass as for the guitars, because
the grind those amps offered worked for both. Glen Matlock's rig names a Fender
Bassman. Neither is a mistake in the document and neither is a substitution.

Until 2026-10-17 the amplifier decided, and both rigs read as guitar chains to
everything downstream. What that cost:

- the corpus filled them from guitar habits, so a bass player got a guitar
  chain's conventions
- a loudness match read guitar cabinets
- the guard that compares a chain against its reference refused Dusty Hill's
  bass rig a bass recording and told him to push a guitar one

**So the document decides, and the amplifier is the fallback.** A chain with no
document has only its amplifier to go on, which is every preset in the corpus: a
`.hlx` carries a `build_sha` and a name. `plan.InstrumentFor` is that fallback
and nothing else should reach for it where a rig is in hand.

## What you may name, and what the device has

A rig names gear a person would recognise and the catalog resolves it. Where
nothing emulates the named gear, the rig states a `substitute:`, which is
somebody deciding rather than a search guessing.

For bass the device models fourteen real amplifiers, and that is the list an
automated rig has to land on:

```
Ampeg SVT (bright, normal)        Ampeg B-15NF Portaflex
MESA/Boogie Bass 400+ (ch 1, 2)   MESA/Boogie M9 Carbine
Gallien-Krueger GK 800RB          Sunn Coliseum 300
Acoustic 360                      Aguilar DB51
Aguilar Tone Hammer               Pearce BC-1 (ch 1, ch 2, jumped)
```

Ask the catalog rather than this list, which is right the day it is written:

```bash
mise exec -- go run main.go catalog list --category amp --json
```

**An artist whose amplifier is not among them needs a stated substitute before
they can be built automatically.** A name found only among the other
instrument's amplifiers is not a substitute: it is a different instrument's
amplifier that happens to share a name, and resolving to it silently is what
produced a guitar rig for a bass player.
