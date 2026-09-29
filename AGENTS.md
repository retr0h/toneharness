# AGENTS.md

Test: `just test` | Before committing: `just ready`

Read [CONTRIBUTING.md](CONTRIBUTING.md) first. It covers prerequisites, setup,
package structure, code standards and testing. All of it applies to agents
exactly as it applies to people. This file carries only what is specific to
agents.

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
not have a command added on the branch you are working on, and reporting that a
command "does not exist yet" when it was added an hour ago is what happens
otherwise.

## CONTRIBUTING is not optional reading

[CONTRIBUTING.md](CONTRIBUTING.md) is the source of truth for layout,
conventions, testing and the licence header every file carries. It applies to
agents exactly as it applies to people, and none of it is repeated here.

Four of its rules are easy to skip and worth naming. Write Go, and reach for
Python only where there is nothing in Go to call: see
[The language is Go](CONTRIBUTING.md#the-language-is-go), which lists the four
jobs that qualify and what it cost the last time something was written twice.
Run `just ready` before committing. Put every markdown change through the unslop
skill first, see [Prose](CONTRIBUTING.md#prose). And when the change touches a
rig, read [Sourcing a rig](CONTRIBUTING.md#sourcing-a-rig) before starting: it
is the difference between research and typing, and it says what a pull request
has to have finished before it is opened.

## Finding your way around the domain

**How to do anything here is in [.claude/skills/](.claude/skills/)**, and each
skill is authoritative for its own domain. Five of them, self-contained and
separately installable, so nothing is stated in two of them:

| Skill              | Owns                                                                                                     |
| ------------------ | -------------------------------------------------------------------------------------------------------- |
| `build-a-rig`      | research, citing gear, resolving an ask, tuning after hearing it                                         |
| `write-a-spec`     | every field on the two contracts, and which document a fact belongs in                                   |
| `measure-a-device` | **playing audio through the pedal and hearing it back**, sweeps, what a control does, trusting a catalog |
| `measure-music`    | growing a corpus, measuring records, players and genres, deriving words                                  |
| `work-a-device`    | reading and writing what a pedal holds, and the rules that keep one alive                                |

Read the skill that matches the task. Do not read all five, and do not restate
one skill's knowledge in another: that is the duplication the split exists to
prevent.

**Nothing you send reaches an amplifier by default.** If audio is going out and
coming back wrong, silent, or unchanging, that is the signal path and it is
`measure-a-device`'s `references/signal-path.md`, before the wire framing and
before anything about the block being measured. The two numbers that decide it
live on the preset rather than in any setting, and one of them is a destination
whose label says it carries USB when it does not.

## Nothing here is one pedal on one laptop

Every number that describes hardware is somebody else's different number, and
each of these has already been a bug:

| Varies with     | Do not hardcode                                                                                                    |
| --------------- | ------------------------------------------------------------------------------------------------------------------ |
| the Helix model | how many audio channels it presents. Ask the device; a Stomp is 8 in and 8 out                                     |
| the Helix model | the routing enum indices. The same file carries separate lists for a Stomp, an LT and the plugin                   |
| the Helix model | how many blocks, paths and snapshots it holds. `plan.LimitsFor` answers from the catalog                           |
| the computer    | the audio device's name and its backend. `--hardware` names one                                                    |
| the computer    | the sample rate. 48kHz is what every committed figure was taken at, and the loop checks it rather than assuming it |
| the room        | whether a cable loops the output back to the input, or the return comes over USB                                   |

A figure measured on one device is a figure about that device. Say which.

Three things are not in a skill, on purpose:

| Task                             | Read                                                                                                                                 |
| -------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------ |
| What is built and what is not    | [docs/knowledge.md](docs/knowledge.md), the status board and why the evaluator is a person                                           |
| Changing the shape of the system | [docs/superpowers/specs/](docs/superpowers/specs/), dated design records, superseded rather than rewritten                           |
| Talking to a device over USB     | [`pkg/sdk/internal/wire/README.md`](pkg/sdk/internal/wire/README.md), the reverse-engineered framing, beside the code that speaks it |

And four questions answer themselves from the tool rather than from any page:

| Question                                  | Ask                                                                                      |
| ----------------------------------------- | ---------------------------------------------------------------------------------------- |
| Which commands and flags exist            | `go run main.go --help`, which compiles the tree. Never a list written down              |
| Which tools an agent gets over MCP        | the running server advertises them. Start it and read what it offers                     |
| What a field may say, and what is refused | the two contracts' `description:` fields                                                 |
| Which words an ask may use                | [`words.json`](pkg/sdk/internal/compile/data/words.json), which is the vocabulary itself |
| What a control actually does              | [resources/sweeps/](resources/sweeps/), the readings themselves                          |

There are two contracts, each embedded in the package that reads it.
[`pkg/sdk/tone/data/tonespec.openapi.yaml`](pkg/sdk/tone/data/tonespec.openapi.yaml)
is what somebody may ask for, and
[`pkg/sdk/rig/data/rigspec.openapi.yaml`](pkg/sdk/rig/data/rigspec.openapi.yaml)
is what that resolves to. They are the only hand-authored formats; the Go types,
both grammar pages and everything downstream are compiled from them. The
generated catalog, the gear map and the corpus are in
[resources/schemas/](resources/schemas/), and
[resources/README.md](resources/README.md) says what else is in that tree and
which of it may be redistributed.

## Say which claim you have

"The rig validates against the catalog", "HX Edit imported the file" and "the
hardware loaded it" are three different claims. The first needs no device. The
third needs one attached and `device current` read back afterwards, because a
chain that is stored is not a chain that rendered: for a fortnight every one
this tool wrote read back byte for byte and drew nothing on the pedal.

Do not report one as another, and do not describe work as verified on evidence
you did not gather. If you did not run it, say you did not run it.

## Task tracking

Work is tracked with Claude Code's task tools, which the superpowers plugin is
built on. `.claude/settings.json` turns them on and names the list `tonestack`,
so it is one list across sessions rather than one per session. See
[CONTRIBUTING.md](CONTRIBUTING.md#claude-code) for setup.

- Check `TaskList` at the start of a session, before picking up work.
- When somebody asks for something to be done later, create a task for it and
  carry on with the current work. Do not start it.
- Mark a task `in_progress` when starting it and `completed` only once the work
  is merged or the decision is made. An open pull request is not done.
- Anything left over at the end of a piece of work, a follow-up, a decision
  nobody has made, a bug found on the way, becomes a task rather than a sentence
  in a reply. A sentence in a reply is gone after the next session.
- When a pull request finishes something [docs/knowledge.md](docs/knowledge.md)
  marks not built or partly built, update that line in the same pull request,
  and say so in its description. That table is how the next session learns what
  exists, and it fell three features behind when nobody did.
- When a pull request changes behaviour a skill describes, update that skill in
  the same pull request. One skill owns each fact, so there is exactly one file
  to change, and a skill that has drifted is worse than no skill: it is
  confident and wrong.

If `TaskCreate` is not available, the tools are off. Say so instead of carrying
on without them.

## Commit trailer

When committing via Claude Code, end the message with:

```
🤖 Generated with [Claude Code](https://claude.ai/code)

Co-Authored-By: Claude <noreply@anthropic.com>
```

## Re-running the scaffold

`retemplate-go` brings this project up to a newer template. It reads
`.swamp-template.json`, which records what the last run wrote and the hash of
each file, and decides per file:

- A file **unchanged since generated** gets the current template's version.
- A file **edited here** is left alone, and named in the output.
- A file that is **absent** gets written.

Keep `.swamp-template.json` in the repository. Without it every file looks
edited, and a retemplate can only skip.

### When a file is reported as an orphan

An orphan is a file an earlier template generated, this one no longer generates,
and nobody has edited. An `internal/cli/` left behind when the entry point moved
to `cmd/` is one. It gets reported, never deleted:

```
orphan internal/cli/cli.go: generated by an earlier template, no longer
       part of this one, and unchanged since. Safe to delete.
```

Check nothing imports it, then delete it. Left in place it compiles and passes
the gate while being unreachable, which is how it goes unnoticed.

### When a file is held

Some files only make sense together: `main.go` imports `cmd`, `cmd` imports
`internal/<pkg>`; the library stub's test calls a function its source defines.
If one of them has been edited, the others are held rather than written, and the
output says so:

```
HOLD cmd/root.go: the entrypoint files this project already has came from an
     earlier template.
```

The run cannot merge your edit into the new shape. Either port the edit by hand,
or move those files aside and re-run to get the current set.
