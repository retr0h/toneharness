"""Read a video's captions as plain text, so a spoken claim can be checked.

A rig rundown is an interview that happens to have been filmed, and it is
often the only place an amplifier is named. The written companion usually
covers the instruments and stops: Premier Guitar's 2015 ZZ Top article lists
Dusty Hill's basses and never his amplifier, while the video has his tech
saying "he uses the JMP-1s and the Marshall Valvestate" out loud. That claim
had rested on nothing for as long as anybody had looked.

So this is not a way to judge a sound. It is a way to read what somebody said.
Footage of a backline settles nothing, which is why the source ladder rates
`video` low; a tech naming a model is a citation, and belongs on the ladder
where interviews sit.

Two things to carry into the note every time.

The captions are automatic, so model numbers arrive mangled in predictable
ways. A Valvestate 8008 comes out "the 808s". A Micro Thumpinator comes out
"Micro Thumpin". Quote them as the transcript renders them and say that is
what happened, rather than silently correcting to the number you expect.

And the bass is usually not the subject. Of eight players this was tried for,
one was settled and one corroborated. Fat Mike's Big Five is about a
Jazzmaster through a Silvertone, the Sum 41 gear rundown covers the
guitarists' Soldano and Marshalls and mentions the bassist only to say which
bass he swaps to, and the Slayer rundown is Kerry King's heads throughout.
Search first, read second, and give up early.

Usage:
    just transcript https://www.youtube.com/watch?v=1x_kg6pOvGU
    just transcript-search "Rig Rundown Pantera Rex Brown bass"
"""

import argparse
import glob
import os
import re
import subprocess
import sys
import tempfile

# How many results a search prints. Enough to see whether a rundown exists for
# a band at all, and few enough to read the titles rather than skim them.
RESULTS = 6


def search(query):
    """Print candidate videos as `id | title`, newest-ranked first.

    Titles are the filter that matters. "Rig Rundown" in the title is a tech
    walking a rack; "Bass Play Through" is somebody's hands and names no gear.
    """
    out = subprocess.run(
        [
            "yt-dlp",
            "--no-warnings",
            "--flat-playlist",
            "--print",
            "%(id)s | %(title)s",
            f"ytsearch{RESULTS}:{query}",
        ],
        capture_output=True,
        text=True,
        check=False,
    )

    if out.stdout.strip():
        print(out.stdout.rstrip())
        return 0

    print(f"no videos found for {query!r}", file=sys.stderr)

    return 1


def plain(vtt):
    """Turn a WebVTT file into one paragraph of readable text.

    Auto-captions roll: each cue repeats the tail of the one before it, so a
    naive strip gives every phrase three times and a search for a model number
    matches noise. Dropping a line equal to the one before collapses that.
    """
    lines = []

    for line in open(vtt, encoding="utf-8").read().splitlines():
        if "-->" in line or line.startswith(("WEBVTT", "Kind:", "Language:")):
            continue

        # Inline timing tags sit inside the cue text itself.
        text = re.sub(r"<[^>]+>", "", line).strip()

        if text and (not lines or lines[-1] != text):
            lines.append(text)

    return re.sub(r"\s+", " ", " ".join(lines))


def transcript(url):
    """Print one video's captions, or say why there are none.

    Captions only. Nothing here downloads a video: the claim being checked is
    something a person said, and the audio and pictures are somebody else's
    property with no reason to be on this disk.
    """
    with tempfile.TemporaryDirectory() as into:
        subprocess.run(
            [
                "yt-dlp",
                "--write-auto-subs",
                "--write-subs",
                "--sub-langs",
                "en.*",
                "--skip-download",
                "--no-warnings",
                "-o",
                os.path.join(into, "%(id)s.%(ext)s"),
                url,
            ],
            capture_output=True,
            text=True,
            check=False,
        )

        found = sorted(glob.glob(os.path.join(into, "*.vtt")))

        if not found:
            print(f"no captions published for {url}", file=sys.stderr)

            return 1

        print(plain(found[0]))

    return 0


def main():
    """Read arguments and dispatch."""
    ap = argparse.ArgumentParser(description=__doc__)
    ap.add_argument("target", help="a video url, or a query with --search")
    ap.add_argument(
        "--search",
        action="store_true",
        help="treat the argument as a search and print candidates",
    )

    args = ap.parse_args()

    return search(args.target) if args.search else transcript(args.target)


if __name__ == "__main__":
    sys.exit(main())
