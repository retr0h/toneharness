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

package plan

import (
	"fmt"
	"maps"
	"slices"

	"github.com/retr0h/toneharness/pkg/sdk/catalog"
)

// ValidateParams reports the first parameter in s that the catalog does not
// declare, or whose value does not fit the declared kind or range. Parameters
// are checked in sorted key order so the same rig always reports the same
// failure.
func ValidateParams(
	l BlockLookup,
	s Plan,
) error {
	for _, sb := range s.Blocks {
		blk, ok := l.Block(sb.Model)
		if !ok {
			return &UnknownBlockError{Model: string(sb.Model)}
		}

		for _, key := range slices.Sorted(maps.Keys(sb.Params)) {
			if err := checkParam(blk, key, sb.Params[key]); err != nil {
				return err
			}
		}
	}

	return nil
}

func checkParam(
	blk catalog.Block,
	key string,
	val catalog.ParamValue,
) error {
	p, ok := blk.Params[key]
	if !ok {
		return &catalog.BadParamError{
			Model: string(blk.ID), Key: key, Reason: "no such parameter",
		}
	}

	switch p.Type {
	case catalog.ParamFloat:
		f, ok := whole(val)
		if !ok {
			return mismatch(blk, key, p.Type, val.Type())
		}

		return checkRange(blk, key, f, p)
	case catalog.ParamInt:
		i, ok := val.Int()
		if !ok {
			return mismatch(blk, key, p.Type, val.Type())
		}

		return checkRange(blk, key, float64(i), p)
	case catalog.ParamBool:
		if _, ok := val.Bool(); !ok {
			return mismatch(blk, key, p.Type, val.Type())
		}

		return nil
	case catalog.ParamEnum:
		e, ok := val.Enum()
		if !ok {
			return mismatch(blk, key, p.Type, val.Type())
		}

		if !slices.Contains(p.Enum, e) {
			return &catalog.BadParamError{
				Model: string(blk.ID), Key: key,
				Reason: fmt.Sprintf("%q is not a declared member", e),
			}
		}

		return nil
	default:
		return &catalog.BadParamError{
			Model: string(blk.ID), Key: key,
			Reason: fmt.Sprintf("catalog declares unknown kind %q", p.Type),
		}
	}
}

// whole reads a float, accepting a whole number written without a point.
//
// Neither YAML nor JSON can tell 6 from 6.0. ParamValue writes a whole-numbered
// float with the point for exactly that reason, and sigs.k8s.io/yaml marshals
// through JSON and then flattens it away again, so a Deluxe Comp's Knee of 6
// left the device as a float and came back as an integer.
//
// The catalog is what knows which it was, which is the same reason typesOf
// exists on the way in. A plan is refused for naming a parameter that does not
// exist or a value out of range; it is not refused for a format that cannot
// spell the difference.
func whole(
	val catalog.ParamValue,
) (float64, bool) {
	if f, ok := val.Float(); ok {
		return f, true
	}

	if i, ok := val.Int(); ok {
		return float64(i), true
	}

	return 0, false
}

// mismatch reports a value whose kind is not the kind the catalog declares.
func mismatch(
	blk catalog.Block,
	key string,
	want catalog.ParamType,
	got catalog.ParamType,
) error {
	return &catalog.BadParamError{
		Model: string(blk.ID), Key: key,
		Reason: fmt.Sprintf("expected %s, got %s", want, got),
	}
}

func checkRange(
	blk catalog.Block,
	key string,
	v float64,
	p catalog.Param,
) error {
	// Compared in the precision the device speaks. A Helix sends a parameter as
	// a float32, so a cabinet's LowCut sitting exactly on its minimum of 19.9
	// comes back as 19.899999618530273 and is below it by an amount that exists
	// only because this comparison is happening in float64.
	//
	// A plan read off a device and written straight back was refused for holding
	// the value the device itself reported, which is the round trip this format
	// exists to survive.
	if float32(v) < float32(p.Min) || float32(v) > float32(p.Max) {
		return &catalog.BadParamError{
			Model: string(blk.ID), Key: key,
			Reason: fmt.Sprintf("%v out of range [%v, %v]", v, p.Min, p.Max),
		}
	}

	return nil
}
