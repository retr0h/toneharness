"""Measure every block the device has, once, at its own defaults.

A sweep says what one control does and costs a couple of minutes. There are
665 blocks on an HX Stomp carrying about five thousand float controls between
them, so sweeping all of them is a hundred and sixty hours and not a plan.

It is also the wrong thing to want. [algorithm.md](../docs/algorithm.md)
measures the matrix fresh for whatever chain is being tuned, because a slope
belongs to its chain, so a stored matrix for every block would be rebuilt
before it was used. What cannot be worked out at solve time is which blocks
are worth putting in the chain at all, and that question needs one number per
block rather than twelve.

So: load each block alone, play the reference through it, keep the five
figures. That is a fingerprint. Ranking 224 amplifiers by how close their
fingerprint sits to a target is what turns "sound like this record" into a
shortlist, and it costs one reading per block instead of a sweep.

Everything is written as it goes. The pedal has dropped off the USB bus
mid-campaign before, and a run that keeps its results in memory until the end
loses hours to that.

Usage:
    just fingerprint                        # every block
    just fingerprint --category amp         # one category
    just fingerprint --resume               # skip what is already measured
"""

import argparse
import gzip
import hashlib
import json
import re
import subprocess
import sys
import tempfile
import time
from pathlib import Path

import numpy as np
import sounddevice as sd
import soundfile as sf

RATE = 48000
REPO = Path(__file__).resolve().parent.parent
DRY = REPO / "resources" / "dry" / "bass-di.wav"
CATALOG = REPO / "resources" / "schemas" / "hx-stomp.catalog.json"
OUT = REPO / "resources" / "sweeps" / "hx-stomp" / "fingerprints.json"

LEAD, TAIL = 0.3, 1.0

# A prebuilt binary, so a run does not relink the CLI once per block.
BINARY = Path("/tmp/tonestack")


def cli(*args: str, timeout: int = 300) -> tuple[bool, str]:
    """One CLI call, saying whether it worked rather than exiting.

    A block that will not load is a result rather than a failure: some of what
    the catalog lists is a split or a utility that means nothing on its own,
    and the run has to get past them to reach the rest.
    """
    exe = [str(BINARY)] if BINARY.exists() else ["go", "run", "main.go"]

    r = subprocess.run(
        [*exe, *args], cwd=REPO,
        capture_output=True, text=True, timeout=timeout, check=False,
    )

    return r.returncode == 0, (r.stderr or r.stdout).strip()


def blocks(category: str) -> list[dict]:
    """Every block worth trying, from the catalog."""
    with gzip.open(CATALOG) as f:
        catalog = json.load(f)

    out = []
    for block in catalog["blocks"].values():
        if category and block["category"] != category:
            continue

        out.append({
            "id": block["id"],
            "name": block["name"],
            "category": block["category"],
        })

    return sorted(out, key=lambda b: (b["category"], b["id"]))


def rig(block: dict) -> str:
    """A rig holding one block, addressed by model rather than by name.

    By model because 665 models share 469 names: "Ampeg SVT" matches both of
    its channels, so a name would measure whichever the compiler picked and
    file it under both.
    """
    ident = re.sub(r"[^a-z0-9]+", "-", block["id"].lower()).strip("-")

    return (
        "# Written by resources/fingerprint.py.\n"
        "schema: RigSpec\n"
        "version: 2\n"
        f"id: fp-{ident}\n"
        "subject:\n"
        "  kind: sound\n"
        f"  name: {block['name']} alone\n"
        "instrument: bass\n"
        "chain:\n"
        f"  - role: {block['category']}\n"
        f"    gear: {block['name']}\n"
        "    models:\n"
        f"      HX Stomp: {block['id']}\n"
    )


def dry(seconds: float) -> np.ndarray:
    """The reference signal, at the device's rate."""
    x, rate = sf.read(DRY, dtype="float32", always_2d=True)
    x = x[:, 0]

    if rate != RATE:
        n = int(round(len(x) * RATE / rate))
        x = np.interp(
            np.linspace(0.0, len(x) - 1, n), np.arange(len(x)), x
        ).astype(np.float32)

    return x[: int(seconds * RATE)]


def device() -> int:
    """The pedal, or a list of what is attached instead."""
    for i, d in enumerate(sd.query_devices()):
        if "hx stomp" in d["name"].lower() and d["max_output_channels"] >= 2:
            return i

    have = ", ".join(d["name"] for d in sd.query_devices() if d["max_output_channels"])
    sys.exit(f"fingerprint: no HX Stomp. Attached: {have}")


def through(signal: np.ndarray, dev: int) -> np.ndarray:
    """Play the signal and record the answer, sharing one clock."""
    x = np.concatenate(
        [np.zeros(int(LEAD * RATE), np.float32), signal,
         np.zeros(int(TAIL * RATE), np.float32)]
    )

    rec = sd.playrec(
        np.column_stack([x, x]), samplerate=RATE, device=dev,
        output_mapping=[1, 2], input_mapping=[1, 2], blocking=True,
    )

    return rec[:, 0]


def figures(x: np.ndarray) -> dict[str, float]:
    """What a recording reads as."""
    w = x * np.hanning(len(x))
    mag = np.abs(np.fft.rfft(w))
    freq = np.fft.rfftfreq(len(w), 1.0 / RATE)
    power = mag ** 2
    total = float(np.sum(power)) or 1.0

    def band(lo: float, hi: float) -> float:
        return 100.0 * float(np.sum(power[(freq > lo) & (freq < hi)])) / total

    rms = float(np.sqrt(np.mean(np.square(x))))

    return {
        "centroid": float(np.sum(freq * power) / total),
        "level": float(20.0 * np.log10(rms)) if rms > 1e-12 else -999.0,
        "low": band(0, 200),
        "mid": band(200, 2000),
        "high": band(2000, 20000),
    }


# Where a reading stops describing the block and starts describing the
# converters running out of headroom.
#
# A clipped recording's spectrum is the clipping's, not the chain's: flat tops
# generate harmonics that were never in the signal, so the centroid and the
# band shares are wrong in a way that looks like a bright block.
CLIPPED = -0.5


def empty(tmp: Path) -> str:
    """A chain with nothing in it, for the baseline.

    A fingerprint means nothing on its own. 95 Hz is not "what this equaliser
    does", it is what the bass already was, and an equaliser flat at its
    defaults passes it through unchanged. Measuring the empty loop once says
    which of the two any later number is.

    One bypassed block rather than no blocks, because the contract puts a
    minimum of one item on a chain and a rig with nothing in it is not a rig.
    Bypassed is the same signal path either way.
    """
    spec = tmp / "empty.yaml"
    built = tmp / "empty.hlx"
    spec.write_text(
        "# Written by resources/fingerprint.py, for the baseline.\n"
        "schema: RigSpec\n"
        "version: 2\n"
        "id: fp-empty\n"
        "subject:\n"
        "  kind: sound\n"
        "  name: nothing at all\n"
        "instrument: bass\n"
        "chain:\n"
        "  - role: eq\n"
        "    gear: Simple EQ\n"
        "    enabled: false\n"
        "    models:\n"
        "      HX Stomp: HD2_EQSimple3Band\n"
    )

    for step in (
        ("presets", "compile", "--rig", str(spec), "--out", str(built)),
        ("presets", "play", "--preset", str(built)),
    ):
        ok, said = cli(*step)
        if not ok:
            return f"{step[1]}: {said.splitlines()[0] if said else 'failed'}"

    return ""


def load(block: dict, tmp: Path) -> str:
    """Put one block in front of the device alone, storing nothing.

    Through `presets play`, which replaces the edit buffer, rather than
    `presets import`, which writes a slot. That choice is what makes this
    campaign possible at all: a slot is flash, a burst of flash writes has
    corrupted a setlist past what a power cycle could clear, and this loads a
    different chain 665 times. Through slots that is 665 flash writes for
    readings nobody wanted to keep.

    Empty string when it worked.
    """
    spec = tmp / "fp.yaml"
    built = tmp / "fp.hlx"
    spec.write_text(rig(block))

    for step in (
        ("presets", "compile", "--rig", str(spec), "--out", str(built)),
        ("presets", "play", "--preset", str(built)),
    ):
        ok, said = cli(*step)
        if not ok:
            return f"{step[1]}: {said.splitlines()[0] if said else 'failed'}"

    return ""


def main() -> None:
    p = argparse.ArgumentParser(description="Measure every block once.")
    p.add_argument("--category", default="", help="only this category")
    p.add_argument("--seconds", type=float, default=4.0,
                   help="how much reference to push through per block")
    p.add_argument("--resume", action="store_true",
                   help="skip blocks already in the output")
    p.add_argument("--out", default=str(OUT))
    args = p.parse_args()

    out = Path(args.out)
    done: dict[str, dict] = {}

    if args.resume and out.exists():
        done = json.loads(out.read_text()).get("blocks", {})
        print(f"  resuming with {len(done)} already measured")

    want = [b for b in blocks(args.category) if b["id"] not in done]
    signal = dry(args.seconds)
    dev = device()

    print(f"  {len(want)} blocks to measure, "
          f"{args.seconds:.0f}s of reference each")

    header = {
        "device": "HX Stomp",
        "reference": {
            "file": str(DRY.relative_to(REPO)),
            "sha256": hashlib.sha256(DRY.read_bytes()).hexdigest(),
            "seconds": args.seconds,
        },
        # Every fingerprint is one block alone. Nothing here describes a
        # chain, which is the whole reason they are comparable to each other.
        "isolated": True,
    }

    started = time.time()

    with tempfile.TemporaryDirectory() as raw:
        tmp = Path(raw)

        # The empty loop, first, so every block below is a difference from
        # something rather than a number on its own.
        why = empty(tmp)
        if why:
            sys.exit(f"fingerprint: cannot measure the empty loop ({why}), so "
                     "nothing below it would mean anything")

        time.sleep(0.3)
        header["baseline"] = figures(through(signal, dev))
        print("  baseline: " + "  ".join(
            f"{k} {v:.2f}" for k, v in header["baseline"].items()))

        for i, block in enumerate(want, 1):
            why = load(block, tmp)

            if why:
                done[block["id"]] = {**block, "refused": why}
                print(f"  {i:>4}/{len(want)}  {block['id']:<40} refused")
            else:
                time.sleep(0.3)
                got = figures(through(signal, dev))

                # Flagged rather than dropped. Where a block clips is worth
                # knowing, and its figures are still not what it does.
                got["clipped"] = got["level"] > CLIPPED
                done[block["id"]] = {**block, **got}

                mark = "  CLIPPED" if got["clipped"] else ""
                print(f"  {i:>4}/{len(want)}  {block['id']:<40} "
                      f"centroid {got['centroid']:8.1f}  "
                      f"level {got['level']:7.2f}{mark}")

            # Written every time. A run that keeps results in memory loses the
            # lot when the pedal drops off the bus, which it has done.
            out.write_text(json.dumps(
                {**header, "blocks": done}, indent=2, sort_keys=True) + "\n")

    took = time.time() - started
    measured = [b for b in done.values() if "centroid" in b]
    clipped = [b for b in measured if b.get("clipped")]
    print(f"\n  {len(measured)} measured, {len(done) - len(measured)} refused, "
          f"{len(clipped)} clipped, in {took / 60:.0f} minutes")
    print(f"  wrote {out}")


if __name__ == "__main__":
    main()
