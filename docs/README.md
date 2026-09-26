# docs

How tonestack works. For how to work *on* it, setup and conventions and testing,
see [CONTRIBUTING.md](../CONTRIBUTING.md). These pages cover the domain.

|                                      |                                                                                       |
| ------------------------------------ | ------------------------------------------------------------------------------------- |
| [workflows.md](workflows.md)         | **Start here.** How to use tonestack, in order, for the things people come here to do |
| [commands.md](commands.md)           | Every command and flag, generated from the CLI                                        |
| [tonespec.md](tonespec.md)           | What a request may say, generated from the contract                                   |
| [rigspec.md](rigspec.md)             | What a rig resolves to, generated from the contract                                   |
| [knowledge.md](knowledge.md)         | How a request becomes a signal chain, and the four problems that entails.             |
| [recipes.md](recipes.md)             | Writing a rig by hand, and the worked example beside it                               |
| [algorithm.md](algorithm.md)         | Turning a request into knob positions, and what a target is                           |
| [measuring.md](measuring.md)         | Pushing audio through a pedal and measuring what comes back                           |
| [measurements.md](measurements.md)   | What the device actually does, generated from the readings                            |
| [catalog.md](catalog.md)             | What a device can do, and where that knowledge comes from                             |
| [preset-format.md](preset-format.md) | How a `.hlx` file is laid out                                                         |
| [device.md](device.md)               | Reading and editing what a device holds, and what USB is for                          |
| [protocol.md](protocol.md)           | Talking to a device over USB: framing, calls, and the rules that keep one alive       |

Design records live under [superpowers/](superpowers/). They are dated, and
superseded rather than rewritten. The current architecture is
[ToneSpec is the ask](superpowers/specs/2026-09-19-tonespec-is-the-ask-design.md),
which supersedes
[RigSpec as the one model](superpowers/specs/2026-09-06-rigspec-as-the-one-model-design.md).
Two documents a person writes, one a machine resolves, and a preset compiled
from that.
