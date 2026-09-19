"""Measure one block completely: every control, alone, folded into a matrix.

A fingerprint says what a block sounds like sitting at its defaults, which is
what picking one out of 665 needs. It says nothing about what its controls do,
and a chain that has been picked still has to be set.

This is the other half. Put the block in front of the device on its own, sweep
each of its float controls across the range the catalog gives it, and fold the
lot into the one document [algorithm.md](../docs/algorithm.md) solves with.

It is the expensive half, which is why it is aimed rather than swept across
everything. A control is about two minutes and a twelve-control amplifier most
of an hour, so the 4,835 float controls on an HX Stomp are a hundred and sixty
hours. Measure the blocks a chain actually reaches for.

Nothing is written to a slot. Every load goes through `presets play`, and
[the rules that keep a device alive](../docs/protocol.md#rules-that-keep-a-device-alive)
say why that matters when a run loads a chain once per control.

Usage:
    just campaign HD2_AmpUSDripmanNorm amp
    just campaign HD2_EQSimple3Band eq --points 7
"""

import argparse
import gzip
import json
import re
import subprocess
import sys
from pathlib import Path

REPO = Path(__file__).resolve().parent.parent
CATALOG = REPO / "resources" / "schemas" / "hx-stomp.catalog.json"
OUT = REPO / "resources" / "sweeps" / "hx-stomp"

# A prebuilt binary, so a campaign does not relink the CLI hundreds of times.
BINARY = Path("/tmp/tonestack")

# Where the block sits once it is the only thing in the chain.
#
# A block's slot is its position plus one, and a chain holding one block puts
# it at position zero. See Addressing a control in docs/measuring.md.
ALONE = 1


def cli(*args: str, timeout: int = 900) -> tuple[bool, str]:
    """One CLI call, saying whether it worked."""
    exe = [str(BINARY)] if BINARY.exists() else ["go", "run", "main.go"]

    r = subprocess.run(
        [*exe, *args], cwd=REPO,
        capture_output=True, text=True, timeout=timeout, check=False,
    )

    return r.returncode == 0, (r.stderr or r.stdout).strip()


def entry(model: str) -> tuple[dict, list[str]]:
    """What the catalog holds for one block, and its parameters in wire order.

    The order comes from `symbols`, which lists them "in the order a device
    sends their values". The per-model map beside it is keyed by name and has
    no order, and `catalog show` prints that one sorted, so counting down the
    printout files every curve under the wrong control.

    An empty order is not an error. The equalisers have parameters and no
    symbol entry at all, and `just identify` is how that gets settled against
    the device rather than guessed.
    """
    with gzip.open(CATALOG) as f:
        catalog = json.load(f)

    if model not in catalog["blocks"]:
        sys.exit(f"campaign: the catalog has no {model}")

    order = next(
        (e["params"] for e in catalog.get("symbols", []) if e["id"] == model),
        [],
    )

    return catalog["blocks"][model], order


def rig(block: dict, role: str) -> str:
    """A rig holding one block, addressed by model rather than by name."""
    ident = re.sub(r"[^a-z0-9]+", "-", block["id"].lower()).strip("-")

    return (
        "# Written by resources/campaign.py. One block, so a sweep of it\n"
        "# measures the block rather than a chain.\n"
        "schema: RigSpec\n"
        "version: 2\n"
        f"id: campaign-{ident}\n"
        "subject:\n"
        "  kind: sound\n"
        f"  name: {block['name']} alone\n"
        "instrument: bass\n"
        "chain:\n"
        f"  - role: {role}\n"
        f"    gear: {block['name']}\n"
        "    models:\n"
        f"      HX Stomp: {block['id']}\n"
    )


def main() -> None:
    p = argparse.ArgumentParser(description="Measure one block completely.")
    p.add_argument("model", help="the block, by its model identifier")
    p.add_argument("role", help="what it does: amp, cab, drive, comp, eq, ...")
    p.add_argument("--points", type=int, default=9, help="positions per control")
    p.add_argument("--takes", type=int, default=3, help="takes for the floor")
    p.add_argument("--work", default="/tmp/campaign", help="where sweeps land")
    args = p.parse_args()

    block, order = entry(args.model)

    work = Path(args.work) / args.model
    work.mkdir(parents=True, exist_ok=True)

    preset = work / "chain.hlx"
    (work / "chain.yaml").write_text(rig(block, args.role))

    ok, said = cli("presets", "compile", "--rig", str(work / "chain.yaml"),
                   "--out", str(preset))
    if not ok:
        sys.exit(f"campaign: {args.model} will not compile alone:\n{said}")

    ok, said = cli("presets", "play", "--preset", str(preset))
    if not ok:
        sys.exit(f"campaign: {args.model} will not load:\n{said}")

    # Dials and lists, and only the ones the catalog can place. A switch is
    # left out because two positions is not a curve, and an unplaced index
    # would be swept over a guessed range and filed under a guessed name.
    want = [
        (i, name, block["params"][name]["type"])
        for i, name in enumerate(order)
        if block["params"].get(name, {}).get("type") in ("float", "int")
    ]

    if not want:
        sys.exit(f"campaign: the catalog gives {args.model} no controls it "
                 "can place in wire order, so nothing here knows which index "
                 f"is which. Run `just identify {ALONE} "
                 f"{len(block['params'])}` against the device first.")

    print(f"  {block['name']} ({args.model}), {len(want)} controls\n")

    for i, name, kind in want:
        print(f"  == {name} (index {i}, {kind})")

        done, _ = sweep(i, args, preset, work)
        if not done:
            print(f"     {name} produced no curve")

    # Back to where it started, so whatever runs next inherits a known chain
    # rather than the last control at the top of its range.
    cli("presets", "play", "--preset", str(preset))

    print(f"\n  fold it with:\n    just fold {work} "
          f"{OUT / (args.model + '.json')}")


def sweep(index: int, args, preset: Path, work: Path) -> tuple[bool, str]:
    """One control, through resources/sweep.py, into the campaign's directory."""
    uvx = subprocess.run(
        ["uvx", "--with", "sounddevice", "--with", "numpy", "--with",
         "soundfile", "python3", "resources/sweep.py",
         str(ALONE), str(index),
         "--points", str(args.points), "--takes", str(args.takes),
         "--isolated", "--preset", str(preset),
         "--out", str(work / f"p{index}.json")],
        cwd=REPO, capture_output=True, text=True, timeout=3600, check=False,
    )

    print("\n".join(uvx.stdout.splitlines()[-10:]))

    return uvx.returncode == 0, uvx.stderr


if __name__ == "__main__":
    main()
