"""The measuring rig: play a signal through the pedal and read what returns.

Shared by everything that measures, because the one time this logic was
written twice it silently produced two different answers.

The second copy was a few lines of numpy that looked right. Held against the
first on the same file, it read the reference bass as 98.6% low with its
centre of gravity at 95 Hz where the real measurement says 93% and 138 Hz. A
band boundary sat at 200 Hz instead of 250, the top band stopped at 20 kHz
instead of running to Nyquist, and the whole file was measured in one
transform rather than in overlapping windows with the rests gated out.

None of that is visible in a number. It made every block's reading
incomparable with every record's while both looked entirely reasonable, which
is the failure this project exists to not have.

So there is one measurement and it lives in Go, in pkg/sdk/audio, where the
records are measured. This records what comes back and hands it to
`tonestack measure --json`.
"""

import json
import subprocess
import sys
import tempfile
from pathlib import Path

import numpy as np
import sounddevice as sd
import soundfile as sf

RATE = 48000
REPO = Path(__file__).resolve().parent.parent
DRY = REPO / "resources" / "dry" / "bass-di.wav"
CATALOG_PATH = REPO / "resources" / "schemas" / "hx-stomp.catalog.json"

# A prebuilt binary, so a run does not relink the CLI once per reading.
BINARY = Path("/tmp/tonestack")

# Silence before and after.
#
# The front is because latency is not known in advance. The back is because a
# reverb goes on ringing after the input stops, and cutting at the end of the
# signal would clip the tail off the thing being measured.
LEAD, TAIL = 0.3, 1.0

# The figures a reading is kept as.
#
# The same names `tonestack measure` answers with, lowercased, so a block and
# a record are described in one vocabulary rather than two that look alike.
FIGURES = ("low", "mid", "high", "centroid", "transient", "decay",
           "dynamics", "harmonics", "lean")


def cli(*args: str, timeout: int = 900) -> tuple[bool, str]:
    """One CLI call, saying whether it worked rather than exiting."""
    exe = [str(BINARY)] if BINARY.exists() else ["go", "run", "main.go"]

    r = subprocess.run(
        [*exe, *args], cwd=REPO,
        capture_output=True, text=True, timeout=timeout, check=False,
    )

    return r.returncode == 0, (r.stderr or r.stdout).strip()


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

    have = ", ".join(
        d["name"] for d in sd.query_devices() if d["max_output_channels"])
    sys.exit(f"no HX Stomp. Attached: {have}")


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
    """What a recording reads as, in the figures a record is described in.

    Written to a file and handed to `tonestack measure`, rather than computed
    here, so there is one implementation of the measurement and not two that
    disagree. It costs a file and a process per reading, which is a second
    against the six the playback already took.

    Level comes back with them and is computed here, because it is not one of
    the figures a record carries. A record's loudness is the mastering
    engineer's decision and says nothing about the playing; a block's is a
    property of the block, and a sweep that could not see it would miss the
    one thing a volume control does.
    """
    with tempfile.NamedTemporaryFile(suffix=".wav", delete=False) as tmp:
        where = Path(tmp.name)

    try:
        sf.write(where, x, RATE)

        ok, said = cli("measure", "--file", str(where), "--json", timeout=300)
        if not ok:
            sys.exit(f"measuring what came back failed:\n{said}")

        read = json.loads(said)
    finally:
        where.unlink(missing_ok=True)

    rms = float(np.sqrt(np.mean(np.square(x))))

    return {
        # Shares, as percentages, which is how every other number here reads.
        "low": 100.0 * read["Low"],
        "mid": 100.0 * read["Mid"],
        "high": 100.0 * read["High"],
        "centroid": read["Centroid"],
        "transient": value(read["Transient"]),
        "decay": value(read["Decay"]),
        "dynamics": read["DynamicRange"],
        # Across the whole spectrum rather than per band, because a block is
        # one sound rather than three.
        "harmonics": 100.0 * sum(read["Harmonics"].values()) / 3.0,
        "lean": sum(read["EvenOdd"].values()) / 3.0,
        "level": float(20.0 * np.log10(rms)) if rms > 1e-12 else -999.0,
    }


def value(of: dict) -> float | None:
    """One of the figures that can be unknown.

    A transient needs a note starting and a decay needs one ending. A window
    holding neither has no answer, and zero would be an answer.

    None rather than a NaN, because a NaN is not JSON: written out it becomes
    the bare token `NaN`, which this can read back and nothing else can.
    """
    return float(of["Value"]) if of.get("Known") else None
