# The contract's own version

`info.version` at the top of
[tonespec.openapi.yaml](../../../../pkg/sdk/tone/data/tonespec.openapi.yaml) is
the version of the contract itself. It moves whenever that file does, and a test
refuses a change that leaves it where it was.

## Which number moves

| Change                                                              | Bump  |
| ------------------------------------------------------------------- | ----- |
| a field removed or renamed, or a value that used to validate refused | major |
| a field added, a value added to an enum, a constraint loosened       | minor |
| wording that changes nothing about what is accepted                  | patch |

Minor is the common one, and the test for it is somebody else's document: if
every document already written still validates, the change is additive however
large it looks.

**A `description:` is the grammar for its field, not a comment on it.** That is
the trap. Rewriting a sentence so it rules something out is a major change even
though no `type:` or `required:` moved, because the thing reading that sentence
is an agent deciding what to write, and documents written against the old
sentence are now wrong. Fixing a typo in the same sentence is a patch. Ask what
a careful reader would do differently.

## How the gate works

`pkg/sdk/tone/version_public_test.go` holds two constants: the version, and the
sha256 of the contract that version describes. Change the contract and the test
fails, printing the hash it got. Then:

1. decide which number moves, from the table above
2. set `info.version`
3. record the new version and hash in that test

Nothing regenerates the pair, and that is deliberate. A `just` recipe that
rewrote the hash would move it on every edit and the decision would stop
happening, which is the whole thing being guarded.

## Why it jumps from 2.0.0 to 2.4.1

The version sat at 2.0.0 from the change that merged the two contracts until the
gate existed, and the contract changed in four merged pull requests in between
with nobody touching it: #225, #231, #232 and #246. All four were additive, and
the only line any of them removed was a `minItems: 1` on the chain, which
loosens rather than tightens, so four minors is what they come to. The patch on
top is the paragraph in `info.description` that says the version moves at all.

That reconstruction is the reason the gate is a hash rather than a note in a
skill. Four people, or the same agent four times, read a contract with a version
field in it and changed the file without the thought occurring.

## What a version does not do

**A document does not say which version it was written against.** `schema:
ToneSpec` is a name, with no version beside it, so a reader has no way to tell a
document that predates a field from one that omits it. That is a gap rather than
a decision, and it is why the major number matters: it is the only warning
anybody gets.

**Nothing migrates.** There is no upgrade path from version 1, and there was no
need for one: version 2 merged two contracts whose pairing was one-to-one, and
every document in the tree was rewritten in the same change. A major bump that
cannot be made that way is a bigger piece of work than editing a schema.
