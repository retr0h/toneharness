# Holding the device, and letting it go

## A session has to be closed

Every command opens three channels and, when it is done, tells the device on each
of them that the session is over. Leaving that out **does not fail**. The command
works, the device answers, and the pedal is left believing an editor is still
attached: turn the dial and the footswitches stop changing with the preset,
because the front panel waits for an editor to tell it what to show.

The panel is dead while a session is held and comes back once it closes.
Confirmed on hardware both ways. So a pedal whose screen has stopped following
its knobs is not broken. Something is still holding it.

Several operations in a row should share one session, which costs one handshake
instead of one per call. The MCP server holds one across device tool calls and
closes it ten seconds after the last one, or when a call fails on the bus.

`device hardware` deliberately does not go through the session. It lists the bus
even while a session holds the editor interface, because it enumerates by
descriptor and opens nothing. It reads what is attached and claims nothing.

## When the interface is claimed and nothing of yours has it

A claim refused with `kIOReturnExclusiveAccess` says "another client -- usually
a kernel driver -- holds the device open", and the tool waits three seconds
before reporting it. The refusal says how long it waited and how many tries that
was, so a hold of a moment and a hold of minutes read differently.

Quit HX Edit first, because it is usually that. When nothing of yours has it, ask
who does before rerunning:

```bash
ioreg -c IOUSBHostInterface -l -w 0 | grep -B 2 -A 6 -i "HX Stomp"
```

The editor interface is number 0. Healthy, `bInterfaceNumber = 0` carries nothing
but `AppleUSBHostCompositeDevice`, which splits a composite device rather than
claiming anything, and Apple's audio driver sits on interfaces 1 and 2 with
`AppleUSBAudioStreamPropertiesReady = Yes`. A driver attached to interface 0 is
the thing to find: it means macOS matched something to the editor interface and
is holding it.

Seen once, on 2026-09-28: five consecutive runs refused, then the sixth ran. It
began a few runs after the pedal was power cycled, which is when macOS
re-enumerates the device and matches drivers to it again. Nothing of ours was
alive, HX Edit was not open, and no MCP server was running. An abrupt kill is not
the cause: the kernel releases a dead process's claim, and two runs killed that
way were followed by clean ones.

## The six rules that keep a device alive

Each was learned by ignoring it, and each cost hardware.

1. **Never call USB reset.** It takes the pedal off the bus and it does not come
   back without a physical unplug.
2. **Always have a read posted.** The device sends notifications unasked. With
   nothing draining its outgoing endpoint its queue fills, at which point it
   stops draining the incoming one and the next write times out.
3. **Pace deferred work on the completion notification.** Racing commits is
   tolerated about a dozen times, and then writes stop being accepted at all.
4. **Handshake once per session.** Repeating it on an open channel wedges the
   device, so a timeout is to be reported, not retried by reconnecting.
5. **Let flash settle.** A burst of renames once corrupted a setlist past what a
   power cycle could clear.
6. **Quit HX Edit first.** It claims interface 0 exclusively while running.

**A wedged device needs the 9V adapter unplugged.** Pulling USB alone is not
enough, because the unit stays powered and keeps its session across a replug.

Which is why Ctrl-C does not cut a write off halfway. A half-sent message stalls
the pedal, so the write finishes and the session closes, which can take twenty
seconds. A third Ctrl-C quits on the spot and leaves an editor attached.

## It is not MIDI

Worth stating, because it is the first place anybody looks. Line 6 filter SysEx
out; the MIDI documentation has program change, control change and clock and
nothing else. MIDI switches presets and cannot read or write one. The editor
channel is raw vendor-specific USB somewhere else entirely, which also settles a
portability question: iOS cannot reach it at all.

If the job really is only switching presets, MIDI program 0 is 01A and each bank
holds three, so a slot is `(bank − 1) × 3 + letter`.

## Platforms

Device access is macOS only; on anything else the device commands say so, and
describing, validating and writing preset files works everywhere. No special
privileges are needed, because listing devices reads the registry and never opens
one.
