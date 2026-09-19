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

# A prebuilt binary, so a sweep does not relink the CLI once per position.
BINARY = Path("/tmp/tonestack")


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
        "level": float(20.0 * np.log10(rms)) if rms > 1e-12 else -999.0,
        "low": band(0, 200),
        "mid": band(200, 2000),
        "high": band(2000, 20000),
    }


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
    exe = [str(BINARY)] if BINARY.exists() else ["go", "run", "main.go"]

    r = subprocess.run(
        [*exe, "presets", "current"],
        cwd=REPO, capture_output=True, text=True, timeout=300, check=False,
    )

    if r.returncode != 0:
        sys.exit("sweep: cannot read what the device is playing, so nothing "
                 f"measured here would be attributable:\n{r.stderr.strip()}")

    return r.stdout


def run(*args: str) -> str | None:
    """One CLI call. None when it worked, the complaint when it did not."""
    exe = [str(BINARY)] if BINARY.exists() else ["go", "run", "main.go"]

    r = subprocess.run(
        [*exe, *args],
        cwd=REPO, capture_output=True, text=True, timeout=300, check=False,
    )

    return None if r.returncode == 0 else (r.stderr or r.stdout).strip()


def turn(block: int, param: int, value: float) -> bool:
    """Move the control, and say whether the device took it."""
    return run("presets", "turn", "--block", str(block),
               "--param", str(param), "--value", str(value)) is None


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

    floor = {
        k: max(
            float(np.max([r[k] for r in rows]) - np.min([r[k] for r in rows])),
            FLOOR[k],
        )
        for k in rows[0]
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
    r = run("presets", "play", "--preset", preset)

    if r is not None:
        sys.exit(f"sweep: cannot put {preset} back, so the chain this would "
                 f"measure is whatever the last run left:\n{r}")


def main() -> None:
    p = argparse.ArgumentParser(description="Sweep one control and measure it.")
    p.add_argument("block", type=int, help="the block, by its device address")
    p.add_argument("param", type=int, help="the parameter, by its position")
    p.add_argument("--points", type=int, default=9, help="how many positions")
    p.add_argument("--low", type=float, default=0.0)
    p.add_argument("--high", type=float, default=1.0)
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

    print(f"block {args.block} parameter {args.param}, "
          f"{args.points} positions from {args.low} to {args.high}")

    noise, settled = steady(signal, dev, args.takes)
    print("  noise floor: " + "  ".join(f"{k} {v:.3f}" for k, v in noise.items()))
    print(f"  settled level: {settled:.2f} dB, "
          f"so silence is anything under {settled - SILENT:.2f}")

    points = []
    for v in np.linspace(args.low, args.high, args.points):
        if not turn(args.block, args.param, float(v)):
            print(f"  {v:.4f}  refused")
            continue

        time.sleep(0.4)
        got = figures(through(signal, dev))

        # Kept, because where a control goes silent is worth knowing, and
        # marked, because the other four figures describe the hiss.
        quiet = bool(got["level"] < settled - SILENT)
        points.append({"value": float(v), "silent": quiet, **got})

        note = "  SILENT, figures are of the noise" if quiet else ""
        print(f"  {v:.4f}  "
              + "  ".join(f"{k} {got[k]:8.3f}" for k in got) + note)

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
        "moves": {
            k: float(max(pt[k] for pt in heard) - min(pt[k] for pt in heard))
            for k in heard[0] if k not in ("value", "silent")
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
