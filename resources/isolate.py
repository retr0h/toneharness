"""Put one block on the device with nothing else around it.

A control's slope is not a property of the control. Treble on an amplifier
into a 4x12 and the same Treble into a 1x15 are two different numbers, and so
is the same amplifier with a drive pedal ahead of it. Sweeping a control in a
preset that holds a whole chain measures the chain.

That is fine when the chain is the thing being tuned, and useless as a
library. A library entry has to say what *this block* does, which means
nothing else may be in the path while it is measured.

So: build a rig holding one block, compile it, put it in a scratch slot and
load it. What comes out is attributable, because this wrote it.

Nothing is written. The chain goes in front of the device through `presets
play`, which replaces the edit buffer and leaves every slot holding what it
held. That matters because a slot is flash, and
[the rules that keep a device alive](../docs/protocol.md#rules-that-keep-a-device-alive)
say a burst of flash writes has corrupted a setlist past what a power cycle
could clear. Measuring means loading a different chain hundreds of times, and
none of those are worth keeping.

The compiled preset is kept on disk so a sweep can put the same chain back
between controls. A live edit writes nothing back, so without that each sweep
would run on a chain the previous one left skewed.

Usage:
    just isolate "Ampeg SVT" amp        # plays it, prints the file
    just isolate "Ampeg SVT" amp --out /tmp/svt.hlx
"""

import argparse
import re
import subprocess
import sys
from pathlib import Path

REPO = Path(__file__).resolve().parent.parent

# A prebuilt binary, so a run does not relink the CLI once per call.
BINARY = Path("/tmp/tonestack")


def run(*args: str) -> str:
    """One CLI call, with the failure reported rather than swallowed."""
    exe = [str(BINARY)] if BINARY.exists() else ["go", "run", "main.go"]

    r = subprocess.run(
        [*exe, *args],
        cwd=REPO, capture_output=True, text=True, timeout=600, check=False,
    )

    if r.returncode != 0:
        sys.exit(f"isolate: {' '.join(args)} failed:\n{r.stderr.strip()}")

    return r.stdout


def rig(gear: str, role: str) -> str:
    """A rig holding one block and nothing else.

    `insist` is not set, so the compiler may substitute the nearest thing the
    device models. It says what it did, and the sweep records the chain it
    actually got rather than the one asked for, which is why an approximate
    match here does not make the measurement wrong.
    """
    ident = re.sub(r"[^a-z0-9]+", "-", gear.lower()).strip("-")

    return (
        "# Written by resources/isolate.py. One block, so a sweep of it\n"
        "# measures the block rather than a chain.\n"
        "schema: RigSpec\n"
        "version: 2\n"
        f"id: isolate-{role}-{ident}\n"
        "subject:\n"
        "  kind: sound\n"
        f"  name: {gear} alone\n"
        "instrument: bass\n"
        "chain:\n"
        f"  - role: {role}\n"
        f"    gear: {gear}\n"
    )


def main() -> None:
    p = argparse.ArgumentParser(description="Load one block, alone, for measuring.")
    p.add_argument("gear", help='what to load, as a person says it: "Ampeg SVT"')
    p.add_argument("role", help="what it does: amp, cab, drive, comp, eq, ...")
    p.add_argument("--out", default="/tmp/isolated.hlx",
                   help="where to keep the compiled preset, so a sweep can "
                        "put the same chain back between controls")
    args = p.parse_args()

    built = Path(args.out)
    spec = built.with_suffix(".yaml")
    spec.write_text(rig(args.gear, args.role))

    print(run("presets", "compile", "--rig", str(spec), "--out", str(built)))
    run("presets", "play", "--preset", str(built))

    # What actually landed, which is the thing to sweep. A substitution the
    # compiler made shows up here rather than being assumed away.
    print(run("presets", "current"))
    print(f"the device is playing {args.gear} and nothing else, and holds what")
    print("it held. Sweep it with:")
    print(f"  just sweep 1 0 --isolated --preset {built}")


if __name__ == "__main__":
    main()
