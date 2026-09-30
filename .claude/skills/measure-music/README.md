# measure-music

Answers "what does this player actually sound like" with a number somebody else
can check, and says which word that number earns.

## Install

```
/plugin marketplace add retr0h/toneharness
/plugin install measure-music@toneharness
```

Or copy `.claude/skills/measure-music/` into any checkout.

## Usage

Name a player, a genre or a file. The skill fetches what it needs, measures it
and says which of four claims it has.

| Ask                                              | You get                                                                  |
| ------------------------------------------------ | ------------------------------------------------------------------------ |
| _"What does this recording measure?"_            | the figures, with the bass separated out of the mix first                 |
| _"Add three records for this bassist"_           | the manifest entry, the audio, and a length check proving it is the take  |
| _"Which words does this player earn?"_           | the words, both sides of each comparison, and the margin each one cleared |
| _"Can I ask for this genre yet?"_                | whether it clears eight records from three players, and who tagged them   |
| _"Are these the right records for this rig?"_    | which records fall outside the years the rig claims, and which half to fix |

## What it covers

Recordings to figures to words to genres, in that order, because each step is
only meaningful once the one before it is right.

A record is the far end of a signal chain, so it says what an answer should
sound like and cannot say what any one control did. A figure names nothing on
its own: it becomes a word by sitting somewhere among other artists measured the
same way, which means every player added changes what all the others earn. A
genre needs a different rule again, and a threshold before anybody may aim at
one.

Nothing here writes a word into a document for you. A term is an argument, and
the skill's job is to make the argument checkable.

It follows the [Agent Skills] format: a slim `SKILL.md` that routes, with the
detail in reference files read only when the question calls for them.

[agent skills]: https://agentskills.io
