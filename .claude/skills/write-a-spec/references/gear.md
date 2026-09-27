# Naming gear, and putting it in order

## Name it the way a person would

`gear: Ampeg SVT`, never `HD2_AmpSVBeastNrm`.

Line 6 rename every model for trademark reasons. Naming the identifier would
tie a rig to one manufacturer, break when they rename a model, and stop it being
read on other hardware, which is the whole reason the format exists.

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
