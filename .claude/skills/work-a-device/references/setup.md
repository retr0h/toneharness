# The setup file: which slots are ours, and which three to reach for

A Setup is the document about the person rather than the sound. A rig says what
gear makes a noise; a setup says which pedal they have, which bass they are
holding, which slot each rig lives in and which slots they reach for.

Two of its fields exist only for this skill, and they are the reason this page
is here:

| Field        | Answers                                                       |
| ------------ | ------------------------------------------------------------- |
| `slots`      | which slots on the pedal this tool manages, and with what      |
| `favourites` | which of them are worth getting to in one step, and in what order |

Everything else on a setup belongs to `write-a-spec`, which owns every field on
both contracts. This page owns what the two above mean for a pedal.

## Start with init, every time

```bash
mise exec -- go run main.go setup init
mise exec -- go run main.go setup show --json
```

`setup init` writes a starter file to `$XDG_CONFIG_HOME/toneharness/setup.yaml`,
which is where `--setup` falls back to on every command that reads one. So
after one `init` nothing has to name it again.

Every field on a setup is optional, which makes the empty document valid and
useless, so what lands is the schema line, the version this build reads and a
commented line beside each field. Uncommenting is the work.

**It refuses a setup already there and never replaces one.** A setup is the only
record of decisions nothing else keeps, so moving the old one aside is the
person's call. Run `setup show` first: if it answers, there is one.

Run `init` before anything that reads a setup, and run `show` before telling
somebody what a rebuild will do. The second is not a formality: `show` is the
only answer to which slots are about to be overwritten.

## When somebody asks for genres or artists to be rebuilt and uploaded

**That request names the slots in `slots`, and nothing else.** Those are the
slots this tool manages; every other slot on the pedal is somebody's own work
and is not part of a rebuild however well the rig matches its name.

```bash
mise exec -- go run main.go setup show --json      # which slots, and what in them
mise exec -- go run main.go slots fill
```

`slots fill` builds every rig the setup places and writes it to the slot it
names. That is the whole of "put my pedal back", and it is what makes the
request askable twice: without it the layout is remembered by whoever ran it
last, and a second run either lands somewhere else or has to be told again.

What it does and does not do:

- **It writes only where the setup says to.** A slot with no entry is untouched.
- **One rig failing does not stop the rest.** A rig that will not build is a
  document to fix, and leaving the other forty written beats leaving a pedal
  half updated with no record of which half.
- **Every slot keeps what it held.** Each entry carries the path, and that path
  is the only way back from a flash write.

Each failure travels on its own entry, as the error and as `why` in the
structured answer. Repeat the ones that happened: a run of forty that failed on
two is two documents to fix, and a reply saying "mostly fine" names neither.

**Say which slots were written and that each was a flash write.** The person
deciding whether to run it again is the one holding the pedal.

### Adding a rig to what gets managed

A new genre or a new artist is a line in `slots`, and the layout is the
person's. Ask where it goes rather than choosing: a slot that looks free on
`slots list` may be free because somebody emptied it on purpose.

## Selecting the active preset, and cycling favourites

Two commands, and the difference is whether a slot has to be named:

```bash
mise exec -- go run main.go device select --slot 19B    # any slot, by number
mise exec -- go run main.go device fav                 # the first favourite
mise exec -- go run main.go device fav --step next
mise exec -- go run main.go device fav --step prev
```

`device select` reaches any of the hundred and twenty-six and wants the number
every time. `device fav` asks the setup instead, which is the right shape for
the three or four somebody actually plays: it does not ask them to remember a
number while they are holding a bass.

**The first favourite listed is where `device fav` lands.** Reordering the list
moves the starting point. There is no separate field for it, because a list
already has a first element.

`--step next` and `--step prev` move one along and wrap at either end. A person
who has reached the last one and asks for next means the first.

**Where the pedal is now is read off the pedal, not remembered.** The rig the
edit buffer names is matched against the rig `slots` puts in each favourite.
That is deliberate: the footswitch is right there, so a remembered position
would be wrong most of the time, and this survives somebody changing preset by
hand.

Two consequences worth saying out loud when they happen:

- **A favourite with no entry in `slots` can be jumped to and never recognised
  as the current one.** The pairing is what makes the match possible. `setup
  show` calls these out.
- **A pedal playing something that is no favourite goes to the first.** Not an
  error: somebody asking to cycle wanted to be among the favourites. The answer
  says what was playing, and that line is the explanation for why `next` landed
  on the first. Repeat it rather than reporting a plain success.

Neither of these writes anything. The preset goes into the edit buffer and the
slot it came from is untouched, so a cycle costs no flash however many times it
is run. That is the difference from `slots fill`, and it is worth stating in an
answer: one of these two costs the pedal nothing and the other is flash.

## Which claim you have

The same ladder as the rest of this skill. `device fav` and `device select`
reach rung two on their own; the pedal is playing it only once `device current`
says so, and "the command returned" is not that.

```bash
mise exec -- go run main.go device current --json
```

For `slots fill`, a write that read back is rung three and not rung four: a
preset can be stored perfectly and rendered as an empty chain. See
[formats.md](formats.md).
