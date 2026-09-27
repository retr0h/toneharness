# docs

How tonestack works. For how to work *on* it, setup and conventions and testing,
see [CONTRIBUTING.md](../CONTRIBUTING.md). These pages cover the domain.

|                                      |                                                                                       |
| ------------------------------------ | ------------------------------------------------------------------------------------- |
| [workflows.md](workflows.md)         | **Start here.** How to use tonestack, in order, for the things people come here to do |
| [commands.md](commands.md)           | Every command and flag, generated from the CLI                                        |
| [mcp.md](mcp.md)                     | The tools an agent gets, generated from the MCP server                                |
| [tonespec.md](tonespec.md)           | What a request may say, generated from the contract                                   |
| [vocabulary.md](vocabulary.md)       | Every word an ask may use and what each moves, generated from the vocabulary          |
| [rigspec.md](rigspec.md)             | What a rig resolves to, generated from the contract                                   |
| [knowledge.md](knowledge.md)         | How a request becomes a signal chain, and the four problems that entails.             |
| [recipes.md](recipes.md)             | Writing a rig and the ask beside it by hand, with the worked examples                 |
| [algorithm.md](algorithm.md)         | Turning a request into knob positions, and what a target is                           |
| [measuring.md](measuring.md)         | Pushing audio through a pedal and measuring what comes back                           |
| [measurements.md](measurements.md)   | What the device actually does, generated from the readings                            |
| [catalog.md](catalog.md)             | What a device can do, and where that knowledge comes from                             |
| [preset-format.md](preset-format.md) | How a `.hlx` file is laid out                                                         |
| [device.md](device.md)               | Reading and editing what a device holds, and what USB is for                          |
| [protocol.md](protocol.md)           | Talking to a device over USB: framing, calls, and the rules that keep one alive       |

Design records live under [superpowers/](superpowers/). They are dated, and
superseded rather than rewritten. The current architecture is
[ToneSpec is the ask](superpowers/specs/2026-09-19-tonespec-is-the-ask-design.md)
with
[A rig is a plan, for one device](superpowers/specs/2026-09-19-a-rig-is-a-plan-for-one-device-design.md)
beside it, together superseding
[RigSpec as the one model](superpowers/specs/2026-09-06-rigspec-as-the-one-model-design.md).
A ToneSpec is what somebody means, a RigSpec is the gear that answers it, and a
PlanSpec is that rig on one device. The third does not exist yet, so a RigSpec
still carries the Helix half.
