# docs

How toneharness works. For how to work *on* it, setup and conventions and
testing, see [CONTRIBUTING.md](../CONTRIBUTING.md). These pages cover the
domain.

| page                               | what it covers                                                                  |
| ---------------------------------- | ------------------------------------------------------------------------------- |
| [architecture.md](architecture.md) | How the whole system works and fits together. The one file to read first.       |
| [authoring.md](authoring.md)       | Writing a rig and the ask beside it by hand, with the worked examples           |
| [algorithm.md](algorithm.md)       | Turning a request into knob positions, and what a target is                     |
| [measuring.md](measuring.md)       | Pushing audio through a pedal and measuring what comes back                     |
| [catalog.md](catalog.md)           | What a device can do, and where that knowledge comes from                       |
| [device.md](device.md)             | Reading and editing what a device holds, and what USB is for                    |
| [protocol.md](protocol.md)         | Talking to a device over USB: framing, calls, and the rules that keep one alive |

Design records live under [superpowers/](superpowers/). They are dated, and
superseded rather than rewritten. The current architecture is
[ToneSpec is the ask](superpowers/specs/2026-09-19-tonespec-is-the-ask-design.md)
with
[A rig is a plan, for one device](superpowers/specs/2026-09-19-a-rig-is-a-plan-for-one-device-design.md)
beside it, together superseding
[RigSpec as the one model](superpowers/specs/2026-09-06-rigspec-as-the-one-model-design.md).
A ToneSpec is what somebody means, a RigSpec is the gear that answers it, and a
Plan is that rig on one device. A person writes the first two; the compiler
produces the third.
