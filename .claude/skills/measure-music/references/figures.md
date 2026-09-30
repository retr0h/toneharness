# Read a recording as figures

```bash
mise exec -- just stems resources/music/bass/<player> resources/music/bass/<player>/stems
mise exec -- go run main.go measure recordings --file take.wav --json
mise exec -- go run main.go measure recordings --dir <stems>/htdemucs \
  --manifest resources/music/bass/<player>/corpus.yaml --json
```

WAV only; convert on the way in with `ffmpeg -i take.mp3 take.wav`. The bass has
to come out of the mix first, because a mix measures the band.

**Do not skip the separation and filter instead.** A centre-channel extraction
with a 250Hz low-pass looks like it works, and both its headline numbers are
circular, because everything above the cutoff was thrown away. The tell is decay
coming back around a tenth of a second, which is faster than a bass note stops.
Kick and bass share those frequencies and no filter separates them.

## What a record is

The far end of a signal chain: the bass, the amplifier, the mic, the desk and the
master, all in one number. So it says what the answer should sound like, and it
cannot say what any one control did, because every one of them is in it and none
of them can be moved. That is what `caveat` on audio evidence is for, and
`--evidence` writes the block so nobody types the numbers. One entry per record,
each with its own url: a single entry averaging four records is the one thing
nobody could check.

## A figure names nothing on its own

91% of the energy below 250Hz is not "scooped", because every isolated bass stem
is mostly low. That figure describes the instrument. A measurement becomes a
word only by sitting somewhere among other artists measured the same way, which
is [words.md](words.md).

## Read the width, not just the middle

A narrow width around a middle is a habit. A wide one is several different
decisions averaged into a number nobody played. With three or four records the
ends of a width are the extreme records rather than a tenth in from them, so one
unusual take is the whole of one end: treat a figure that disagrees with the rest
as a question rather than an answer.

## Two measured axes are deliberately not derived

`attack` is not derived because what is measured is how fast a level rises, while
the click of a pick is spectral, and the bass stems carry almost nothing above
1kHz, so the figure that axis needs is not in the recording. `decay` is not derived
because it moves by a third of a second depending on which tracks were picked,
which measures the choice rather than the player. The reasoning is in
`pkg/sdk/audio/derive.go`, beside the code that would have to change.

## The manifest is the statement of what was measured

The disk is incidental. A record dropped from a manifest is one somebody decided
not to measure, and its stems are still there because separating is expensive.
Nothing here deletes audio. Anything the manifest and the stems disagree about is
reported on stderr, so stdout stays clean to redirect.

Neither the records nor the measurements are in this repository: the stems are cut
from somebody else's audio files. So a figure in an ask is a claim about a
recording nobody else here can replay, and the url is the only thing that makes it
checkable.
