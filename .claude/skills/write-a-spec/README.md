# write-a-spec

Answers "why was this field refused" and "which half of the document does this go in".

## Install

```
/plugin marketplace add retr0h/toneharness
/plugin install write-a-spec@toneharness
```

Or copy `.claude/skills/write-a-spec/` into any checkout.

## Usage

| Ask                                                  | You get                                                                       |
| ---------------------------------------------------- | ----------------------------------------------------------------------------- |
| _"Write me a ToneSpec for Geddy Lee"_                | the ask, with the era and the instruments it was played on, cited             |
| _"Why won't this rig build?"_                        | which field the contract refuses, and what it does take instead              |
| _"Where does the expression pedal go?"_              | the plan, and why a portable rig has nowhere to put one                      |
| _"Set the treble to 0.85"_                           | the four inputs that decide a value, and which one a number beats            |
| _"Make a version of this for one song"_              | a whole second document, because nothing merges                              |
| _"What did I think of round three?"_                 | the corrections on the ask, which is the only human ear in the project       |

## How it works

Three layers in two documents. A **ToneSpec** holds both of the first two: `ask:`
is what somebody means and `rig:` is the gear that answers it. A **Plan** is that
rig on one device.

Only the ToneSpec has a contract, and the rule behind that is worth stating:
becoming a file is not what earns a contract, being typed by somebody is. Nobody
writes a plan by hand, so a plan is a Go type and nothing more.

The contract is the grammar. `tonespec.openapi.yaml` is hand-authored and
everything downstream is generated from it, so this skill routes to the contract
rather than restating it. A list of fields written into a skill is right the day
it is written and wrong after the next change.

It follows the [Agent Skills] format: a slim `SKILL.md` that routes, with the
detail in reference files read only when the question calls for them.

[agent skills]: https://agentskills.io
