# resources

The data this project reads and ships, as opposed to the Go that reads it.

| Path         |                                                                           |
| ------------ | ------------------------------------------------------------------------- |
| `schemas/`   | the generated catalog and gear map, and the corpus they come from         |
| `music/`     | records measured to describe how somebody plays, one directory per artist |
| `dry/`       | bass straight to the converter, the signal a device is measured with      |
| `sweeps/`    | what each control does, measured through that signal                      |
| `reference/` | one of each document with every optional field filled in                  |

Nothing here is embedded in the binary. What ships lives beside the package that
reads it: `pkg/sdk/catalog/data/`, `pkg/sdk/corpus/data/`,
`pkg/sdk/preset/data/`, `pkg/sdk/rig/data/`, `pkg/sdk/internal/compile/data/`
and `pkg/sdk/internal/wire/data/`. `go:embed` cannot reach a parent directory,
and a package that needs a file from elsewhere is a package nobody can move.

This tree is the working material those files are built from. The RigSpec
contract, the one format anybody hand-authors, is
`pkg/sdk/rig/data/rigspec.openapi.yaml`. The rigs people use and send are in
[../marketplace/](../marketplace/README.md).

`reference/` is the exception to all of that: a ToneSpec, a RigSpec and a Plan
with every optional field written out, so there is one place to see a field in
use. The tests read these rather than the marketplace, because a rig somebody
submits or retracts must not change what a test asserts.

## What may be redistributed

The rigs in `../marketplace/` are ours, and `pkg/sdk/shipped/` is the generated
copy of its core tier that the binary carries.

`schemas/hx-stomp.catalog.json` and `schemas/gear-map.json` are generated from a
licensed HX Edit installation, and `schemas/corpus/` is other people's presets.
Neither travels with a release. See [schemas/README.md](schemas/README.md) for
where each came from.

`music/` and `dry/` are recordings, and neither may be redistributed. Both are
ignored by git and described by a README carrying the urls and hashes to fetch
them again. See [dry/README.md](dry/README.md), which also says why a record
cannot do the job a dry signal does.
