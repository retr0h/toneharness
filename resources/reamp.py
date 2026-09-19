"""Push a dry signal through the pedal and keep what comes back.

The pedal is an audio interface as well as an amplifier, so a file can be
played into its signal chain and the result recorded, without anybody
playing anything. That is what makes a measurement repeatable: the same
bass, the same notes, the same everything, with one control moved.

Two ways in, and they differ only in which channel carries the signal:

    --route usb       computer plays on USB 5/6, the Input block's source.
                      Nothing to plug in. Needs the preset's Input block set
                      to USB 5/6, and see docs/measuring.md, because this
                      does not currently work on a class-compliant driver.

    --route analog    computer plays on USB 1/2, which the pedal sends to its
                      Main outs, and a cable carries that back to its own
                      input jack. Costs one cable and a conversion each way,
                      and uses only the channel pair that is known to work.

Either way the answer comes back on USB 1/2, which is what the pedal
records.

Usage:
    just reamp resources/dry/bass-di.wav out.wav
    uv run --with sounddevice --with numpy --with soundfile \
        resources/reamp.py in.wav out.wav --route analog
"""

import argparse
import sys

import numpy as np
import sounddevice as sd
import soundfile as sf

# The pedal runs at this and nothing else without Line 6's own driver, so a
# file at any other rate is resampled rather than played at the wrong pitch.
RATE = 48000

# Which channels carry the signal each way, counting from one.
#
# The return is USB 1/2 either way: that is the pair the pedal records its
# processed output on, and the only pair proven to carry audio in both
# directions on a class-compliant driver.
ROUTES = {
    "usb": {"send": [5, 6], "note": "into the Input block directly"},
    "analog": {"send": [1, 2], "note": "out of the Main outs, back in by cable"},
}
RETURN = [1, 2]

# How long to keep recording after the file ends.
#
# The pedal delays what it is given, and a reverb or a delay block goes on
# ringing after the input stops. Cutting at the end of the file would clip
# the tail off the thing being measured.
TAIL = 2.0

# Silence before the signal, so the start is findable.
#
# Latency is not known in advance and varies with buffer size. A known gap of
# nothing, then the signal, means the offset can be measured from the
# recording rather than assumed.
LEAD = 0.5


def resample(
    x: np.ndarray,
    have: int,
    want: int,
) -> np.ndarray:
    """Put a signal on the device's rate, linearly.

    Linear interpolation is not a good resampler and does not need to be.
    The bass here holds nothing above 8kHz worth arguing about, and both the
    dry file and the answer are measured the same way, so whatever this loses
    it loses from both sides.
    """
    if have == want:
        return x

    n = int(round(len(x) * want / have))

    return np.interp(
        np.linspace(0.0, len(x) - 1, n),
        np.arange(len(x)),
        x,
    ).astype(np.float32)


def find_device(
    want: str,
) -> int:
    """Find the pedal by name, or say what is attached instead."""
    for i, d in enumerate(sd.query_devices()):
        if want.lower() in d["name"].lower() and d["max_output_channels"] >= 2:
            return i

    have = ", ".join(
        d["name"] for d in sd.query_devices() if d["max_output_channels"]
    )
    sys.exit(f"reamp: no audio device called {want!r}. Attached: {have}")


def through(
    dry: np.ndarray,
    device: int,
    send: list[int],
) -> np.ndarray:
    """Play the signal and record the answer, in one pass.

    One pass rather than play-then-record, because the two have to share a
    clock. Recording separately would drift against the playback by however
    much the two streams disagree, and a measurement of a moving target is
    not a measurement.
    """
    lead = np.zeros(int(LEAD * RATE), dtype=np.float32)
    tail = np.zeros(int(TAIL * RATE), dtype=np.float32)
    padded = np.concatenate([lead, dry, tail])

    out = np.tile(padded[:, None], (1, len(send)))

    return sd.playrec(
        out,
        samplerate=RATE,
        device=device,
        output_mapping=send,
        input_mapping=RETURN,
        blocking=True,
    )


def report(
    name: str,
    x: np.ndarray,
) -> float:
    """Say how loud something is, and hand back its peak."""
    peak = float(np.max(np.abs(x)))
    rms = float(np.sqrt(np.mean(np.square(x))))
    db = 20 * np.log10(rms) if rms > 1e-12 else float("-inf")
    print(f"  {name:<9} peak {peak:8.5f}   rms {db:7.1f} dBFS")

    return peak


def main() -> None:
    p = argparse.ArgumentParser(description="Play a dry signal through the pedal.")
    p.add_argument("dry", help="the wav to push through")
    p.add_argument("out", help="where to write what comes back")
    p.add_argument("--route", choices=sorted(ROUTES), default="usb")
    p.add_argument("--device", default="HX Stomp")
    args = p.parse_args()

    x, rate = sf.read(args.dry, dtype="float32", always_2d=True)
    dry = resample(x[:, 0], rate, RATE)

    device = find_device(args.device)
    route = ROUTES[args.route]

    print(f"{sd.query_devices(device)['name']} @ {RATE}")
    print(f"  {args.route}: sending on USB {'/'.join(map(str, route['send']))}, "
          f"{route['note']}")
    print(f"  recording USB {'/'.join(map(str, RETURN))}, "
          f"{len(dry) / RATE:.1f}s plus {TAIL:.0f}s of tail\n")

    wet = through(dry, device, route["send"])

    report("sent", dry)
    peak = report("returned", wet)

    sf.write(args.out, wet, RATE)
    print(f"\n  wrote {args.out}")

    # A returned peak at the noise floor means the signal never arrived, and
    # measuring that would produce numbers describing a converter at rest.
    if peak < 1e-4:
        sys.exit(
            "\nreamp: nothing came back. The pedal received audio and did not "
            "pass it. See docs/measuring.md, which says what has been ruled "
            "out and what has not."
        )


if __name__ == "__main__":
    main()
