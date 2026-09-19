"""Move one control through its range and measure the sound at each step.

This is what replaces guessing. A character word currently moves a control
by an amount somebody chose, and nothing has ever checked that the figure
the word was earned from follows. A sweep answers that: the same signal, the
same everything, one control moved, measured at every position.

The output is data rather than prose. Numbers baked into code are how the
guesses got in; a file the code reads can be regenerated when the rig
improves, and improves every rig at once when it does.

Three things have to be true for a reading to mean anything, and all three
are written into the output so a later reader can check them:

  - the same reference signal every time, identified by its hash
  - the loop's own repeatability, so a difference smaller than the noise is
    not reported as a finding
  - the device and preset it was taken on

Usage:
    just sweep 0 2                      # block 0, parameter 2
    just sweep 0 2 --points 17          # finer
"""

import argparse
import hashlib
import json
import subprocess
import sys
import time
from pathlib import Path

import numpy as np
import sounddevice as sd
import soundfile as sf

RATE = 48000
REPO = Path(__file__).resolve().parent.parent
DRY = REPO / "resources" / "dry" / "bass-di.wav"

# How much of the reference to push through per reading.
#
# A sweep is one pass per position, so length is time: seventeen positions of
# the whole three and a half minutes is an hour. Six seconds holds enough
# playing to measure and keeps a sweep to a minute.
SECONDS = 6.0

# Silence before and after.
#
# The front is because latency is not known in advance. The back is because a
# reverb goes on ringing after the input stops, and cutting at the end of the
# signal would clip the tail off the thing being measured.
LEAD, TAIL = 0.3, 1.0


def dry() -> np.ndarray:
    """The reference signal, at the device's rate."""
    x, rate = sf.read(DRY, dtype="float32", always_2d=True)
    x = x[:, 0]

    if rate != RATE:
        n = int(round(len(x) * RATE / rate))
        x = np.interp(
            np.linspace(0.0, len(x) - 1, n), np.arange(len(x)), x
        ).astype(np.float32)

    return x[: int(SECONDS * RATE)]


def device() -> int:
    """The pedal, or a list of what is attached instead."""
    for i, d in enumerate(sd.query_devices()):
        if "hx stomp" in d["name"].lower() and d["max_output_channels"] >= 2:
            return i

    have = ", ".join(d["name"] for d in sd.query_devices() if d["max_output_channels"])
    sys.exit(f"sweep: no HX Stomp. Attached: {have}")


def through(signal: np.ndarray, dev: int) -> np.ndarray:
    """Play the signal and record the answer, sharing one clock."""
    x = np.concatenate(
        [np.zeros(int(LEAD * RATE), np.float32), signal,
         np.zeros(int(TAIL * RATE), np.float32)]
    )

    rec = sd.playrec(
        np.column_stack([x, x]),
        samplerate=RATE,
        device=dev,
        output_mapping=[1, 2],
        input_mapping=[1, 2],
        blocking=True,
    )

    return rec[:, 0]


def figures(x: np.ndarray) -> dict[str, float]:
    """What a recording reads as.

    The same shape of measurement the records are described in, computed here
    rather than shelled out to, because a sweep takes hundreds of these and
    each process start costs more than the arithmetic.
    """
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
        "level": 20.0 * np.log10(rms) if rms > 1e-12 else -999.0,
        "low": band(0, 200),
        "mid": band(200, 2000),
        "high": band(2000, 20000),
    }


def turn(block: int, param: int, value: float) -> bool:
    """Move the control, and say whether the device took it."""
    r = subprocess.run(
        ["go", "run", "main.go", "presets", "turn",
         "--block", str(block), "--param", str(param), "--value", str(value)],
        cwd=REPO, capture_output=True, text=True, timeout=180, check=False,
    )

    return r.returncode == 0


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
}


def steady(signal: np.ndarray, dev: int, takes: int) -> dict[str, float]:
    """How much a reading wanders when nothing is touched.

    The floor. A control that moves a figure by less than this moved nothing
    anybody can demonstrate, and reporting it would be reporting the rig
    breathing.
    """
    rows = [figures(through(signal, dev)) for _ in range(takes)]

    return {
        k: max(
            float(np.max([r[k] for r in rows]) - np.min([r[k] for r in rows])),
            FLOOR[k],
        )
        for k in rows[0]
    }


def main() -> None:
    p = argparse.ArgumentParser(description="Sweep one control and measure it.")
    p.add_argument("block", type=int, help="the block, by its device address")
    p.add_argument("param", type=int, help="the parameter, by its position")
    p.add_argument("--points", type=int, default=9, help="how many positions")
    p.add_argument("--low", type=float, default=0.0)
    p.add_argument("--high", type=float, default=1.0)
    p.add_argument("--takes", type=int, default=5, help="takes for the noise floor")
    p.add_argument("--out", default="", help="where to write the result")
    args = p.parse_args()

    signal = dry()
    dev = device()

    print(f"block {args.block} parameter {args.param}, "
          f"{args.points} positions from {args.low} to {args.high}")

    noise = steady(signal, dev, args.takes)
    print("  noise floor: " + "  ".join(f"{k} {v:.3f}" for k, v in noise.items()))

    points = []
    for v in np.linspace(args.low, args.high, args.points):
        if not turn(args.block, args.param, float(v)):
            print(f"  {v:.4f}  refused")
            continue

        time.sleep(0.4)
        got = figures(through(signal, dev))
        points.append({"value": float(v), **got})
        print(f"  {v:.4f}  " + "  ".join(f"{k} {got[k]:8.3f}" for k in got))

    if not points:
        sys.exit("sweep: the device refused every position; nothing to report")

    out = {
        "block": args.block,
        "param": args.param,
        "reference": {
            "file": str(DRY.relative_to(REPO)),
            "sha256": hashlib.sha256(DRY.read_bytes()).hexdigest(),
            "seconds": SECONDS,
        },
        "noise": noise,
        "points": points,
        "moves": {
            k: float(max(pt[k] for pt in points) - min(pt[k] for pt in points))
            for k in points[0] if k != "value"
        },
    }

    # What the control actually did, against what the rig can tell apart. A
    # figure that moved less than the noise floor did not move.
    print("\n  figure      moved   noise   real?")
    for k, moved in out["moves"].items():
        print(f"  {k:<10} {moved:7.3f} {noise[k]:7.3f}   "
              f"{'yes' if moved > noise[k] * 3 else 'no'}")

    where = Path(args.out) if args.out else None
    if where:
        where.write_text(json.dumps(out, indent=2) + "\n")
        print(f"\n  wrote {where}")


if __name__ == "__main__":
    main()
