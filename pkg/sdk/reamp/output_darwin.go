//go:build darwin

// Copyright (c) 2026 John Dewey

// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to
// deal in the Software without restriction, including without limitation the
// rights to use, copy, modify, merge, publish, distribute, sublicense, and/or
// sell copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:

// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.

// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING
// FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER
// DEALINGS IN THE SOFTWARE.

package reamp

/*
#cgo LDFLAGS: -framework CoreAudio -framework CoreFoundation
#include <CoreAudio/CoreAudio.h>
#include <CoreFoundation/CoreFoundation.h>

// defaultOut answers the system's current default output device.
static AudioDeviceID defaultOut(void) {
  AudioDeviceID id = kAudioObjectUnknown;
  UInt32 size = sizeof(id);
  AudioObjectPropertyAddress at = {
    kAudioHardwarePropertyDefaultOutputDevice,
    kAudioObjectPropertyScopeGlobal,
    kAudioObjectPropertyElementMain,
  };
  AudioObjectGetPropertyData(kAudioObjectSystemObject, &at, 0, NULL, &size, &id);
  return id;
}

// setDefaultOut makes one device the system's default output.
static OSStatus setDefaultOut(AudioDeviceID id) {
  AudioObjectPropertyAddress at = {
    kAudioHardwarePropertyDefaultOutputDevice,
    kAudioObjectPropertyScopeGlobal,
    kAudioObjectPropertyElementMain,
  };
  return AudioObjectSetPropertyData(
    kAudioObjectSystemObject, &at, 0, NULL, sizeof(id), &id);
}

// outCount answers how many devices the system has, output or not.
static UInt32 outCount(void) {
  UInt32 size = 0;
  AudioObjectPropertyAddress at = {
    kAudioHardwarePropertyDevices,
    kAudioObjectPropertyScopeGlobal,
    kAudioObjectPropertyElementMain,
  };
  if (AudioObjectGetPropertyDataSize(kAudioObjectSystemObject, &at, 0, NULL, &size) != noErr) {
    return 0;
  }
  return size / sizeof(AudioDeviceID);
}

// outAt answers the nth device's identifier.
static AudioDeviceID outAt(UInt32 n) {
  UInt32 size = 0;
  AudioObjectPropertyAddress at = {
    kAudioHardwarePropertyDevices,
    kAudioObjectPropertyScopeGlobal,
    kAudioObjectPropertyElementMain,
  };
  if (AudioObjectGetPropertyDataSize(kAudioObjectSystemObject, &at, 0, NULL, &size) != noErr) {
    return kAudioObjectUnknown;
  }
  UInt32 held = size / sizeof(AudioDeviceID);
  if (n >= held) {
    return kAudioObjectUnknown;
  }
  AudioDeviceID *ids = (AudioDeviceID *)malloc(size);
  if (ids == NULL) {
    return kAudioObjectUnknown;
  }
  AudioDeviceID got = kAudioObjectUnknown;
  if (AudioObjectGetPropertyData(kAudioObjectSystemObject, &at, 0, NULL, &size, ids) == noErr) {
    got = ids[n];
  }
  free(ids);
  return got;
}

// outName copies a device's name into buf, and answers whether it could.
static int outName(AudioDeviceID id, char *buf, int cap) {
  CFStringRef said = NULL;
  UInt32 size = sizeof(said);
  AudioObjectPropertyAddress at = {
    kAudioObjectPropertyName,
    kAudioObjectPropertyScopeGlobal,
    kAudioObjectPropertyElementMain,
  };
  if (AudioObjectGetPropertyData(id, &at, 0, NULL, &size, &said) != noErr || said == NULL) {
    return 0;
  }
  int ok = CFStringGetCString(said, buf, cap, kCFStringEncodingUTF8) ? 1 : 0;
  CFRelease(said);
  return ok;
}

// playsOut answers whether a device has any output channels, which is what
// makes it a candidate: a microphone is a device and cannot be an output.
static int playsOut(AudioDeviceID id) {
  UInt32 size = 0;
  AudioObjectPropertyAddress at = {
    kAudioDevicePropertyStreamConfiguration,
    kAudioObjectPropertyScopeOutput,
    kAudioObjectPropertyElementMain,
  };
  if (AudioObjectGetPropertyDataSize(id, &at, 0, NULL, &size) != noErr || size == 0) {
    return 0;
  }
  AudioBufferList *list = (AudioBufferList *)malloc(size);
  if (list == NULL) {
    return 0;
  }
  int channels = 0;
  if (AudioObjectGetPropertyData(id, &at, 0, NULL, &size, list) == noErr) {
    for (UInt32 i = 0; i < list->mNumberBuffers; i++) {
      channels += list->mBuffers[i].mNumberChannels;
    }
  }
  free(list);
  return channels > 0 ? 1 : 0;
}
*/
import "C"

import (
	"strings"
	"unsafe"
)

// CoreAudio rather than AppleScript, which is the one difference from the output
// level beside this. AppleScript owns the slider and has nothing to say about
// which device the slider belongs to, and the alternatives are scripting the
// Settings window through the accessibility API or asking a person.
//
// Why it is here at all: the output device is part of the measuring rig in
// exactly the way the level is. A Mac with its default on a pair of Bluetooth
// headphones pinned those to 38 while the headphone jack feeding the pedal sat
// wherever it was left, and the loop read 66dB of loss. The level was pinned and
// the pin reached nothing in the signal path.

// nameCap is how much room a device name is given, in bytes.
//
// CoreFoundation will not say how long a name is without being handed somewhere
// to put it, and the longest on this machine is 28 characters.
const nameCap = 256

// outputs answers every device that can play, by name.
func outputs() []string {
	out := []string(nil)

	for n := range uint32(C.outCount()) {
		id := C.outAt(C.UInt32(n))
		if id == C.kAudioObjectUnknown || C.playsOut(id) == 0 {
			continue
		}

		if name, ok := named(id); ok {
			out = append(out, name)
		}
	}

	return out
}

// named answers one device's name.
func named(
	id C.AudioDeviceID,
) (string, bool) {
	buf := make([]byte, nameCap)

	if C.outName(id, (*C.char)(unsafe.Pointer(&buf[0])), C.int(nameCap)) == 0 {
		return "", false
	}

	return string(buf[:clen(buf)]), true
}

// clen is where a C string ends inside a buffer.
func clen(
	buf []byte,
) int {
	for i, b := range buf {
		if b == 0 {
			return i
		}
	}

	return len(buf)
}

// output answers the name of the system's default output device.
func output() (string, error) {
	id := C.defaultOut()
	if id == C.kAudioObjectUnknown {
		return "", &OutputError{Doing: "reading", Said: errNoOutput}
	}

	name, ok := named(id)
	if !ok {
		return "", &OutputError{Doing: "naming", Said: errNoOutput}
	}

	return name, nil
}

// setOutput makes the device whose name contains want the default output.
//
// Matched the way `--hardware` matches, as a case-folded substring, so the same
// string selects the same device in both places. A name that fits two devices
// takes the first, which is the same rule and the same risk.
func setOutput(
	want string,
) error {
	for n := range uint32(C.outCount()) {
		id := C.outAt(C.UInt32(n))
		if id == C.kAudioObjectUnknown || C.playsOut(id) == 0 {
			continue
		}

		name, ok := named(id)
		if !ok || !strings.Contains(
			strings.ToLower(name), strings.ToLower(want)) {
			continue
		}

		if status := C.setDefaultOut(id); status != 0 {
			return &OutputError{Doing: "setting", Said: errRefused}
		}

		return nil
	}

	return &NoOutputError{Want: want, Had: outputs()}
}
