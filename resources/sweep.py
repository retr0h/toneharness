"""Move one control through its range and measure the sound at each step.

This is what replaces guessing. A character word currently moves a control
by an amount somebody chose, and nothing has ever checked that the figure
the word was earned from follows. A sweep answers that: the same signal, the
same everything, one control moved, measured at every position.

The output is data rather than prose. Numbers baked into code are how the
guesses got in; a file the code reads can be regenerated when the rig
improves, and improves every rig at once when it does.

Four things have to be true for a reading to mean anything, and all four are
written into the output so a later reader can check them:

  - the same reference signal every time, identified by its hash
  - the loop's own repeatability, so a difference smaller than the noise is
    not reported as a finding
  - the whole chain it was taken through, because a control's slope belongs
    to the chain and not to the control
  - whether that chain held anything but the block being swept, which is the
    difference between a curve that describes a block and one that describes
    a preset

Usage:
    just isolate "Ampeg SVT" amp        # one block, nothing else
    just sweep 1 2 --isolated           # block at slot 1, its parameter 2
    just sweep 1 2 --isolated --points 17
"""

import argparse
import gzip
import hashlib
import json
import sys
import time
from pathlib import Path

import numpy as np

sys.path.insert(0, str(Path(__file__).resolve().parent))

from rig import (  # noqa: E402
    CATALOG_PATH, DRY, REPO, cli, device, dry, figures, through,
)

def chain() -> str:
    """The whole signal chain the measurement is being taken through.

    Without this a sweep is unreadable a week later. A control's slope is not
    a property of the control: Treble on an amplifier into a 4x12 and the same
    Treble into a 1x15 are two different numbers, and so is the same amplifier
    with a drive pedal in front of it. A file that records the move but not
    the chain records a number nobody can say the meaning of.

    The edit buffer rather than a slot, because that is what is being played
    and what `turn` moves. Reading the slot would answer with the stored
    document and describe a chain nobody measured.

    Taken as the rig, which is the same document `presets compile` reads, so a
    reading can be rebuilt into the chain it was taken on rather than
    described in prose beside it.
    """
    ok, said = cli("presets", "current", timeout=300)

    if not ok:
        sys.exit("sweep: cannot read what the device is playing, so nothing "
                 f"measured here would be attributable:\n{said}")

    return said


def span(rig: str, at: int, index: int) -> tuple[str, str, float, float] | None:
    """The control's name and the range it actually runs over.

    Sweeping every control from zero to one is right for an amplifier, whose
    knobs all run 0..1, and wrong for most of what else is on the device. A
    Simple EQ's Mid Freq runs 125 to 4000 Hz and its High Gain runs -12 to
    +12 dB. Swept 0..1 the first never leaves its bottom stop and the second
    covers a twenty-fourth of its travel, and both report as controls that
    barely do anything.

    The kind comes back with it, because a device does not coerce. A
    cabinet's Mic is one of twelve microphones rather than a position on a
    dial, and it refuses a float with the same error a block that is not
    there gives. It is also that cabinet's first control and changes its
    sound more than any of its knobs, so a sweep that could only send floats
    measured the dials and reported the one that matters as unreachable.

    None when the catalog cannot name the control. That is not rare: the
    equalisers have parameters and no entry in the symbol list at all, so
    nothing here knows which index is which, and `just identify` is how that
    gets settled against the device rather than guessed.
    """
    if not CATALOG_PATH.exists():
        return None

    model = ""
    blocks = rig.split("\n- enabled:")[1:]

    if at - 1 < len(blocks):
        for line in blocks[at - 1].splitlines():
            if line.strip().startswith("HX Stomp:"):
                model = line.split(":", 1)[1].strip()

    if not model:
        return None

    with gzip.open(CATALOG_PATH) as f:
        catalog = json.load(f)

    order = next(
        (e["params"] for e in catalog.get("symbols", []) if e["id"] == model),
        [],
    )

    if index >= len(order):
        return None

    name = order[index]
    spec = catalog["blocks"].get(model, {}).get("params", {}).get(name)

    if not spec or spec.get("type") not in ("float", "int"):
        return None

    return name, spec["type"], float(spec["min"]), float(spec["max"])


def turn(block: int, param: int, value: float, kind: str) -> bool:
    """Move the control, and say whether the device took it.

    The flag is the value's type, because the device's refusal for the wrong
    one is indistinguishable from its refusal for a block that is not there.
    """
    flag = "--choice" if kind == "int" else "--value"
    said = str(int(round(value))) if kind == "int" else str(value)

    ok, _ = cli("presets", "turn", "--block", str(block),
                "--param", str(param), flag, said)

    return ok


# The smallest spread a noise floor is allowed to claim.
#
# Back-to-back takes through a settled loop can agree exactly, and a floor of
# zero would make every difference significant, including the last digit of a
# float. These are what the converters and the arithmetic can honestly tell
# apart, and a measured floor wider than one of them replaces it.
FLOOR = {
    "centroid": 0.2,
    "level": 0.05,
    "low": 0.02,
    "mid": 0.02,
    "high": 0.005,
    "transient": 0.005,
    "decay": 0.01,
    "dynamics": 0.05,
    "harmonics": 0.05,
    "lean": 0.005,
}


# How far under the settled level a reading may sit and still be a reading.
#
# Below this nothing came through, and the figures stop describing the chain
# and start describing the converters. A centroid computed on silence is the
# centroid of the noise, which is broadband, so it reads high and steady and
# looks exactly like a control that moved the sound somewhere.
#
# That is not hypothetical. A sweep of this amplifier reported its Bass
# control moving the centroid by 2959 Hz; the move was the chain going silent
# at one end, and the 2959 Hz was hiss.
SILENT = 30.0


def steady(
    signal: np.ndarray, dev: int, takes: int
) -> tuple[dict[str, float], float]:
    """How much a reading wanders when nothing is touched, and how loud it is.

    The floor. A control that moves a figure by less than this moved nothing
    anybody can demonstrate, and reporting it would be reporting the rig
    breathing.

    The level comes back with it because the floor alone cannot catch silence:
    two takes of silence agree perfectly, so a figure computed from them looks
    more trustworthy than one computed from music.
    """
    rows = [figures(through(signal, dev)) for _ in range(takes)]

    # Only the figures every take answered with. A transient needs a note
    # starting and a decay needs one ending, so either can come back unknown,
    # and a floor computed across a None is not a floor.
    floor = {
        k: max(
            float(np.max([r[k] for r in rows]) - np.min([r[k] for r in rows])),
            FLOOR[k],
        )
        for k in rows[0]
        if all(r.get(k) is not None for r in rows)
    }

    return floor, float(np.median([r["level"] for r in rows]))


def reload(preset: str) -> None:
    """Put the chain back, so a sweep starts from a known state.

    `turn` does not write anything back, so the control a sweep finishes with
    is left wherever the sweep left it: at the top of its range. Sweeping a
    second control after that measures it on a chain the first one skewed, and
    a campaign of eleven sweeps measures the eleventh on an amplifier with
    four controls pinned at maximum.

    Playing the preset again replaces the edit buffer and undoes every move.
    Nothing is written: a slot is flash and this happens once per control, so
    doing it through a slot would spend a flash write per sweep on a chain
    nobody wanted to keep.
    """
    ok, said = cli("presets", "play", "--preset", preset)

    if not ok:
        sys.exit(f"sweep: cannot put {preset} back, so the chain this would "
                 f"measure is whatever the last run left:\n{said}")


def main() -> None:
    p = argparse.ArgumentParser(description="Sweep one control and measure it.")
    p.add_argument("block", type=int, help="the block, by its device address")
    p.add_argument("param", type=int, help="the parameter, by its position")
    p.add_argument("--points", type=int, default=9, help="how many positions")
    p.add_argument("--low", type=float, default=None,
                   help="the bottom of the range; the catalog's, by default")
    p.add_argument("--high", type=float, default=None,
                   help="the top of the range; the catalog's, by default")
    p.add_argument("--takes", type=int, default=5, help="takes for the noise floor")
    p.add_argument("--out", default="", help="where to write the result")
    p.add_argument("--isolated", action="store_true",
                   help="the chain holds this block and nothing else")
    p.add_argument("--preset", default="",
                   help="play this preset first, so the sweep starts from a "
                        "known chain rather than from what the last one left")
    args = p.parse_args()

    signal = dry()
    dev = device()

    # Before anything is read, so the chain recorded below is the chain
    # measured rather than the one the previous sweep finished on.
    if args.preset:
        reload(args.preset)

    rig = chain()

    # The catalog's range unless somebody gave one, because most controls do
    # not run zero to one and a sweep over the wrong span measures a stop.
    known = span(rig, args.block, args.param)
    kind = known[1] if known else "float"
    low = args.low if args.low is not None else (known[2] if known else 0.0)
    high = args.high if args.high is not None else (known[3] if known else 1.0)
    called = known[0] if known else f"index {args.param}"

    # Every setting, for a control that is a list. Twelve microphones are
    # twelve sounds and nothing sits between two of them, so asking for nine
    # evenly spaced positions would measure some of them twice and miss
    # others entirely.
    points = args.points
    if kind == "int":
        points = int(high - low) + 1

    if known is None and (args.low is None or args.high is None):
        print("  the catalog cannot name this control, so the range is a "
              "guess of 0..1; `just identify` settles which index it is")

    print(f"block {args.block} parameter {args.param} ({called}, {kind}), "
          f"{points} positions from {low} to {high}")

    noise, settled = steady(signal, dev, args.takes)
    print("  noise floor: " + "  ".join(
        f"{k} {v:.3f}" for k, v in noise.items()))
    print(f"  settled level: {settled:.2f} dB, "
          f"so silence is anything under {settled - SILENT:.2f}")

    points = []
    for v in np.linspace(low, high, points):
        if not turn(args.block, args.param, float(v), kind):
            print(f"  {v:.4f}  refused")
            continue

        time.sleep(0.4)
        got = figures(through(signal, dev))

        # Kept, because where a control goes silent is worth knowing, and
        # marked, because the other four figures describe the hiss.
        quiet = bool(got["level"] < settled - SILENT)
        points.append({"value": float(v), "silent": quiet, **got})

        note = "  SILENT, figures are of the noise" if quiet else ""
        print(f"  {v:.4f}  " + "  ".join(
            f"{k} {got[k]:8.3f}" for k in ("centroid", "level", "low", "high")
        ) + note)

    if not points:
        sys.exit("sweep: the device refused every position; nothing to report")

    # Only the positions that carried signal. A move measured against silence
    # is a move between the chain and the converters.
    heard = [pt for pt in points if not pt["silent"]]
    if len(heard) < 2:
        sys.exit(f"sweep: only {len(heard)} of {len(points)} positions carried "
                 "signal, so there is no curve here to report")

    out = {
        "block": args.block,
        "param": args.param,
        # What the catalog calls this index and the span it was swept over.
        # A curve filed without them is a curve nobody can place: the same
        # index is a different control on a different model, and the same
        # control means nothing swept over the wrong range.
        "control": called,
        "kind": kind,
        "range": {"low": low, "high": high,
                  "from_catalog": known is not None},
        # What it was measured through. `isolated` is the difference between a
        # curve that describes this block and one that describes this chain:
        # only the first belongs in a library, and only a chain holding one
        # block can claim it.
        "chain": {
            "isolated": args.isolated,
            "rig": rig,
        },
        "reference": {
            "file": str(DRY.relative_to(REPO)),
            "sha256": hashlib.sha256(DRY.read_bytes()).hexdigest(),
            "seconds": SECONDS,
        },
        "noise": noise,
        # What a position has to clear to count. Kept so a later reader can
        # see which readings were excluded and why, rather than finding a
        # curve with holes in it and no explanation.
        "settled": settled,
        "silent_below": settled - SILENT,
        "points": points,
        # Only where every position answered. A figure that is unknown at
        # one setting and known at another has no move: the difference is
        # between a reading and the absence of one.
        "moves": {
            k: float(max(pt[k] for pt in heard) - min(pt[k] for pt in heard))
            for k in heard[0]
            if k not in ("value", "silent")
            and all(pt.get(k) is not None for pt in heard)
        },
    }

    # What the control actually did, against what the rig can tell apart. A
    # figure that moved less than the noise floor did not move.
    print("\n  figure      moved   noise   real?")
    for k, moved in out["moves"].items():
        if k not in noise:
            continue

        print(f"  {k:<10} {moved:7.3f} {noise[k]:7.3f}   "
              f"{'yes' if moved > noise[k] * 3 else 'no'}")

    where = Path(args.out) if args.out else None
    if where:
        where.write_text(json.dumps(out, indent=2) + "\n")
        print(f"\n  wrote {where}")


if __name__ == "__main__":
    main()
