"""Find out which control each parameter index actually is.

A parameter has no name on the wire. `turn` takes a position in the model's
own list, and every sweep this repository has taken is filed under that
number. The catalog claims to know that order, in `Symbol.Params`, "in the
order a device sends their values" — and nothing has ever held the claim to
the device.

It matters more than it sounds. `catalog show` prints the same parameters
sorted for a reader, so the two orders disagree on almost every model: the
listing for this amplifier begins Bass, Bias, BiasX and the wire begins Norm
Drive, Bass, Mid. Counting down the wrong one mislabels every curve, and the
numbers stay plausible while they do it.

So check rather than trust. Move an index, read back what the device is
playing, and see which named parameter changed.

Two values are tried per index, because a control already sitting at the
first one would show no change and be reported as missing.

Usage:
    just identify 1 12                  # block at slot 1, indices 0..11
"""

import argparse
import re
import subprocess
import sys
from pathlib import Path

REPO = Path(__file__).resolve().parent.parent

# What to move each index to.
#
# Two, so an index already resting on the first is still caught. Both are
# inside every float range this addresses and neither is a default.
PROBES = (0.123, 0.877)


def cli(*args: str) -> str:
    """One CLI call, failing loudly."""
    r = subprocess.run(
        ["go", "run", "main.go", *args],
        cwd=REPO, capture_output=True, text=True, timeout=600, check=False,
    )

    if r.returncode != 0:
        sys.exit(f"identify: {' '.join(args)} failed:\n{r.stderr.strip()}")

    return r.stdout


def params(at: int) -> dict[str, str]:
    """The named parameters of one block, as the device now holds them.

    Read out of the rig `presets current` prints. The block is found by its
    position rather than by name, and a slot is its position plus one.
    """
    rig = cli("presets", "current")
    blocks = re.split(r"^- enabled:", rig, flags=re.M)[1:]

    if at - 1 >= len(blocks):
        sys.exit(f"identify: the chain has {len(blocks)} blocks, so slot {at} "
                 "is not one of them")

    body = blocks[at - 1].split("params:", 1)[-1].split("\n  path:", 1)[0]

    return dict(
        re.findall(r"^\s{4}'?([^':\n]+)'?:\s*(.+)$", body, flags=re.M)
    )


def main() -> None:
    p = argparse.ArgumentParser(description="Name each parameter index.")
    p.add_argument("block", type=int, help="the block, by its device address")
    p.add_argument("count", type=int, help="how many indices to try")
    p.add_argument("--slot", type=int, default=40, help="reload this first")
    args = p.parse_args()

    found: dict[int, str] = {}

    for index in range(args.count):
        cli("presets", "select", "--slot", str(args.slot))
        before = params(args.block)

        for probe in PROBES:
            r = subprocess.run(
                ["go", "run", "main.go", "presets", "turn",
                 "--block", str(args.block), "--param", str(index),
                 "--value", str(probe)],
                cwd=REPO, capture_output=True, text=True, timeout=180,
                check=False,
            )

            if r.returncode != 0:
                print(f"  {index:>3}  refused a float")
                break

            moved = [k for k, v in params(args.block).items()
                     if before.get(k) != v]

            if len(moved) == 1:
                found[index] = moved[0]
                print(f"  {index:>3}  {moved[0]}")
                break

            if moved:
                print(f"  {index:>3}  moved {len(moved)}: {', '.join(moved)}")
                break
        else:
            print(f"  {index:>3}  nothing changed")

    cli("presets", "select", "--slot", str(args.slot))

    print("\n  index  control")
    for index in sorted(found):
        print(f"  {index:>5}  {found[index]}")


if __name__ == "__main__":
    main()
