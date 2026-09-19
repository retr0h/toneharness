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

package tone

// LoadSchema is the contract reader, exported for tests that hand it a
// document other than the one this binary ships.
var LoadSchema = load

// Contracts is the cached pair of schemas, exported so a test can stand in a
// failure for them. The real ones are embedded and cannot fail; what needs
// covering is what happens to a document if they ever did.
var Contracts = &contracts

// Invalid turns a schema failure into one that names the field, exposed so a
// test can hand it a failure the library does not currently produce.
var Invalid = invalid

// Against holds a document to the schema of that name, exposed so a test can
// pass one the generated types could never build.
var Against = against

// Check marshals a document and holds it to the schema of that name, exposed
// so a test can hand it a value that cannot be marshalled at all.
//
// Neither ToneSpec nor Setup can reach that: every field is a string, a
// number, a boolean or a slice of those. The guard is here for the field that
// is not, the way a rig carries the raw state a device wrote, and a test is
// how it stays a guard rather than becoming dead code somebody deletes.
var Check = check
