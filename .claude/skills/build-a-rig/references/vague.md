# When the ask is too vague to build

"Make me a punk bass tone" names no gear, no player and no recording. Nothing
in it can be resolved, and the tool says so:

```
nothing in the request names gear, and nothing in it can be measured against,
so there is no chain to build
```

**Do not guess.** Ask. The order below is the order that resolves the most for
the fewest questions.

## What to ask for, in order

**1. A record.** Worth more than anything else, because a recording is measured
against every block the device has and the answer is arithmetic. One file
settles the amplifier.

> "Which record should this sound like? A track name is enough, or point me at
> a file."

**2. A player, and when.** A player resolves to researched gear with citations.
The era matters as much as the name: Mike Dirnt's rig for *American Idiot* is
not his rig for *Dookie*, and measured from the wrong records he earns the
opposite word on two axes.

> "Whose sound, and from which album or era?"

**3. Gear they already know they want.** If they can name the amp, that is
exact and needs no measuring.

**4. Adjectives, last.** A word is the weakest input, because it has to be
earned against a population of players before it means anything. 91% of the
energy below 250Hz is not "scooped": every isolated bass stem is mostly low.
Words are useful for correcting a rig you have heard, not for building one from
nothing.

## One more, and it is not about the sound

**Where they are plugging in.** Not a way of narrowing the ask, so it does not
belong in the four above. It belongs in the Setup, as `plays_into`, and it
changes on a different clock from any request: somebody who plays through a PA
plays through a PA next week too.

> "What does the pedal go into? A PA or interface, headphones, the front of an
> amp, or an amp's effects return?"

Optional, and the tool does nothing with it on its own. Ask once, put it in the
Setup, and stop.

**It does not settle whether to use a cabinet block, and do not tell somebody it
does.** A cabinet block is how a chain is made to sound like a recorded rig, so
somebody chasing a record may want one into a real amplifier as well, and
somebody who wants their own amplifier to be the sound may not. Both are
reasonable. The two amplifier answers are also different paths: the front of an
amplifier puts its own preamp ahead of its speaker, and the effects return
bypasses that preamp.

What it is worth is honesty about the figures. Everything in
[resources/sweeps/](../../../../resources/sweeps/) was measured through a
cabinet block into a computer, which is one of these four, and a reading is
worth less to somebody on another. Nothing here has measured an amplifier in
anybody's room, so there is no correction to apply and inventing one is the
guessing this project removed.

## When it is already answerable, build it

Do not interrogate somebody who said "like this file". If the request names a
recording, a player this repository has, or any gear, run it:

```bash
mise exec -- go run main.go tone build --ask request.yaml --json
```

Check `rigs list` first: if they named a player who already ships, there is
nothing to research.

## How many records are enough

For a genre, eight from three artists is the threshold, because a word is
earned by sitting outside the middle half of the other players and three is the
fewest that makes "other players" mean anything.

For one player, a single record answers the amplifier. More records narrow the
words, and the report says how much the choice of record moved them, which is
the honest measure of whether one was enough.

## When the rig cannot be established

Say so. Do not ship a half-known rig as though it were known.

The bar is [Sourcing a rig](../../../../CONTRIBUTING.md#sourcing-a-rig) and it
is the same bar for an agent as for a person: every claim has a source somebody
opened. If the research does not reach it, the honest answer is that the gear
could not be established, with what was searched. A rig full of `kind: llm`
assertions is worse than no rig, because it looks like knowledge.
