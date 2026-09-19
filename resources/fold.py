"""Fold a block's sweeps into the one file the algorithm reads.

A sweep measures one control. What solves for a setting is the whole block at
once: a column per control, a row per figure, each cell the slope of that
figure against that control. `docs/algorithm.md` calls it the matrix, and
without it a request is a search over every combination, which for five
controls at five positions each is 1,953,125 chains and 226 days of playing.

So this is the step between the two. It reads every sweep taken on one block,
checks they agree about what they were taken on, and writes one document.

What it refuses is the point. Sweeps taken through different chains, or
against different reference signals, do not belong in one matrix: their slopes
are not comparable and stacking them produces a matrix that solves confidently
for the wrong thing.

Usage:
    just fold sweeps/ resources/sweeps/hx-stomp/us-dripman-norm.json
"""

import argparse
import gzip
import json
import sys
from pathlib import Path

import numpy as np

REPO = Path(__file__).resolve().parent.parent

FIGURES = ("centroid", "level", "low", "mid", "high")

# The catalog, which is the only thing that knows a parameter's name.
#
# `symbols` lists each model's parameters in the order the device sends their
# values, which is the order `turn` addresses them by. The per-model `params`
# map beside it is keyed by name and has no order at all, and `catalog show`
# prints it sorted for a reader. Reading the sorted one and counting down it
# mislabels nearly every curve: this amplifier prints Bass, Bias, BiasX and
# sends Norm Drive, Bass, Mid.
CATALOG = REPO / "resources" / "schemas" / "hx-stomp.catalog.json"


def wire(model: str) -> list[str]:
    """The model's parameters, in the order the device addresses them.

    Empty when the catalog has no entry, which leaves the indices as their own
    labels rather than inventing names for them.
    """
    if not CATALOG.exists():
        return []

    with gzip.open(CATALOG) as f:
        catalog = json.load(f)

    for entry in catalog.get("symbols", []):
        if entry["id"] == model:
            return entry["params"]

    return []


def slope(points: list[dict], figure: str) -> float:
    """How much one figure moves per full turn of the control.

    A least-squares fit rather than the difference between the ends, because
    the ends are two readings and the fit uses every one. A control that is
    not a straight line is still summarised by its average slope, and the
    points travel with it so the curve is not lost.
    """
    x = np.array([p["value"] for p in points], dtype=float)
    y = np.array([p[figure] for p in points], dtype=float)

    if len(x) < 2 or float(np.ptp(x)) == 0.0:
        return 0.0

    return float(np.polyfit(x, y, 1)[0])


def agree(sweeps: list[tuple[Path, dict]]) -> tuple[str, bool]:
    """The chain every sweep shares, or an exit saying which one differs.

    Two sweeps taken on different chains describe different blocks however
    similar their numbers look, and a matrix built from both is not wrong in a
    way anything downstream could notice.
    """
    first, doc = sweeps[0]
    rig = doc["chain"]["rig"]
    ref = doc["reference"]["sha256"]

    for path, other in sweeps[1:]:
        if other["chain"]["rig"] != rig:
            sys.exit(f"fold: {path.name} was taken on a different chain from "
                     f"{first.name}, so their slopes are not comparable")

        if other["reference"]["sha256"] != ref:
            sys.exit(f"fold: {path.name} used a different reference signal "
                     f"from {first.name}, which invalidates the comparison")

    return rig, all(d["chain"]["isolated"] for _, d in sweeps)


def named(rig: str, at: int) -> tuple[str, str]:
    """The gear and model of the block being swept, out of the recorded rig.

    Read back rather than passed in, so the name on the file is the name the
    device gave and not one somebody typed beside it.
    """
    blocks = rig.split("\n- enabled:")[1:]

    if at - 1 >= len(blocks):
        sys.exit(f"fold: the recorded chain has {len(blocks)} blocks, so slot "
                 f"{at} is not one of them")

    body = blocks[at - 1]
    gear = model = "unknown"

    for line in body.splitlines():
        if line.startswith("  gear:"):
            gear = line.split(":", 1)[1].strip()
        if line.strip().startswith("HX Stomp:"):
            model = line.split(":", 1)[1].strip()

    return gear, model


def main() -> None:
    p = argparse.ArgumentParser(description="Fold sweeps into one matrix.")
    p.add_argument("directory", help="where the sweeps are")
    p.add_argument("out", help="where to write the folded document")
    p.add_argument("--names", default="",
                   help="index=name pairs overriding the catalog, comma "
                        "separated, as `just identify` prints them")
    args = p.parse_args()

    sweeps = []
    for f in sorted(Path(args.directory).glob("*.json")):
        doc = json.loads(f.read_text())

        # A folded document is JSON in the same directory and globs the same,
        # so folding twice into one place picks up the last answer as if it
        # were a reading. Said rather than crashed on.
        if not isinstance(doc.get("chain"), dict) or "param" not in doc:
            print(f"  skipping {f.name}, which is not a sweep")

            continue

        sweeps.append((f, doc))

    if not sweeps:
        sys.exit(f"fold: no sweeps in {args.directory}")

    rig, isolated = agree(sweeps)
    at = sweeps[0][1]["block"]
    gear, model = named(rig, at)

    # The catalog's own order, overridden by anything given by hand, so a
    # model the catalog does not carry can still be labelled.
    names = dict(enumerate(wire(model)))
    names.update(
        (int(k), v) for k, v in
        (pair.split("=", 1) for pair in args.names.split(",") if pair)
    )

    params = {}
    for path, doc in sweeps:
        heard = [pt for pt in doc["points"] if not pt.get("silent")]
        index = doc["param"]

        params[names.get(index, f"index {index}")] = {
            "index": index,
            "slopes": {f: slope(heard, f) for f in FIGURES},
            "noise": doc["noise"],
            "silent_below": doc.get("silent_below"),
            "muted_at": [pt["value"] for pt in doc["points"]
                         if pt.get("silent")],
            "points": heard,
        }

    out = {
        "device": "HX Stomp",
        "gear": gear,
        "block": model,
        "slot": at,
        # Whether these describe the block or a chain it happened to be in.
        # Nothing downstream may treat the second as the first.
        "isolated": isolated,
        "chain": rig,
        "reference": sweeps[0][1]["reference"],
        "params": params,
    }

    Path(args.out).write_text(json.dumps(out, indent=2) + "\n")

    print(f"  {gear} ({model}), slot {at}, "
          f"{'isolated' if isolated else 'IN A CHAIN'}")
    print(f"\n  {'control':<14}" + "".join(f"{f:>12}" for f in FIGURES))
    for name, entry in params.items():
        row = "".join(f"{entry['slopes'][f]:>12.2f}" for f in FIGURES)
        print(f"  {name:<14}{row}")

    muted = {n: e["muted_at"] for n, e in params.items() if e["muted_at"]}
    if muted:
        print("\n  went silent, and those positions are left out:")
        for name, at_values in muted.items():
            print(f"    {name}: " + ", ".join(f"{v:.2f}" for v in at_values))

    print(f"\n  wrote {args.out}")


if __name__ == "__main__":
    main()
