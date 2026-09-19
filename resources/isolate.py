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

Usage:
    just isolate "Ampeg SVT" amp        # loads it, prints the slot
    just isolate "Ampeg SVT" amp --slot 41

Slots start at 40 because the first ten banks are John's own presets and a
measurement is not worth overwriting somebody's work.
"""

import argparse
import re
import subprocess
import sys
import tempfile
from pathlib import Path

REPO = Path(__file__).resolve().parent.parent

# The first slot a scratch preset may be written to.
#
# Banks 01-10 are in use. A sweep writes and re-writes whatever slot it is
# given hundreds of times, so the floor is a guard rather than a preference.
SCRATCH = 40


def run(*args: str) -> str:
    """One CLI call, with the failure reported rather than swallowed."""
    r = subprocess.run(
        ["go", "run", "main.go", *args],
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
    p.add_argument("--slot", type=int, default=SCRATCH, help="where to put it")
    args = p.parse_args()

    if args.slot < SCRATCH:
        sys.exit(f"isolate: slot {args.slot} is below {SCRATCH}, which is "
                 f"somebody's own preset. Pick {SCRATCH} or higher.")

    with tempfile.TemporaryDirectory() as tmp:
        spec = Path(tmp) / "isolate.yaml"
        built = Path(tmp) / "isolate.hlx"
        spec.write_text(rig(args.gear, args.role))

        print(run("presets", "compile", "--rig", str(spec), "--out", str(built)))
        run("presets", "import", "--preset", str(built), "--slot", str(args.slot))
        run("presets", "select", "--slot", str(args.slot))

    # What actually landed, which is the thing to sweep. A substitution the
    # compiler made shows up here rather than being assumed away.
    print(run("presets", "show", "--slot", str(args.slot)))
    print(f"slot {args.slot} holds {args.gear} and nothing else. Sweep it with:")
    print(f"  just sweep 1 0 --slot {args.slot} --isolated")


if __name__ == "__main__":
    main()
