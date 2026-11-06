# AGENTS.md

Test: `just test` | Before committing: `just ready`

Read [CONTRIBUTING.md](CONTRIBUTING.md) first. It covers prerequisites, setup,
project structure, how the system and its skills fit together, sourcing a rig,
code standards, testing, and the prose rules every markdown change goes through.
All of it applies to agents exactly as it applies to people. This file carries
only what is specific to agents.

## Running tools

Invoke tools through `mise`, not from your path:

```bash
mise exec -- just test
```

`mise` is active in a person's shell and supplies the versions `.mise.toml`
declares. An agent's shell has no activation, so a bare `just` resolves to
whatever is installed globally, usually an older version.

The symptom is a check that fails here and passes in continuous integration, on
a file nobody edited. When that happens, establish which version ran before
treating the failure as real.

### Running the CLI itself

Every page here writes commands as `toneharness ...`, which is how somebody with
it installed runs them. From a checkout, run the source:

```bash
go run main.go rigs records --corpus resources/music/bass
```

Use an installed `toneharness` only if you have one. It is a release, so it does
not have a command added on the branch you are working on. Reporting that a
command "does not exist yet" when it was added an hour ago is what happens
otherwise.

## What to report, every time

The commands here say what they did and where it went. An agent that runs one
and summarises it as "done" throws that away, and the person reading has no way
to get it back. So a skill's answer carries the same four things the CLI's does.

**Where anything landed, by path.** Every file written, and every file read or
replaced on the way. This matters most where something was overwritten: a slot
write keeps what the slot held and prints where, and a reply that does not
repeat that path has quietly turned a recoverable change into a lost one.

**What was decided on somebody's behalf.** Every block the corpus added, every
substitution made for gear the device has no model of, every word that moved
nothing, every note the tool returned about what it could not honour. These are
printed because a person may disagree with them, and a summary that keeps only
the result hides them.

**Which claim you have.** Each skill carries the ladder for its own domain. Say
the rung you reached, never the one above it. "The rig validates against the
catalog", "HX Edit imported the file" and "the hardware loaded it" are three
different claims. The third needs a device attached and `device current` read
back afterwards, because a chain that is stored is not a chain that rendered. If
you did not run it, say so. A fix that passes tests is not a fix that worked on
hardware.

**What it cost the device, where it touched one.** A slot write is a flash
write. Say which slots were written, because the person deciding whether to do
it again is the one holding the pedal.

Lead with what happened, then the paths, then what was chosen for them, then the
claim. Keep the tool's own wording for a refusal or a figure. The numbers and
field names are what somebody greps for, and a reworded error cannot be
searched.

## Commit trailer

When committing via Claude Code, end the message with:

```
🤖 Generated with [Claude Code](https://claude.ai/code)

Co-Authored-By: Claude <noreply@anthropic.com>
```
