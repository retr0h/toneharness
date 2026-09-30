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
package solve

import (
	"errors"
	"fmt"
	"math"
)

// ErrSingular is a system no amount of damping made solvable.
var ErrSingular = errors.New("the slopes describe no solvable system")

// least solves min |Ax - b| for x, preferring small x.
//
// Through the normal equations with a ridge term: (AᵀA + λI)x = Aᵀb. Written
// here rather than taken from a library because the system is nine figures by a
// dozen controls at most, so the whole thing is a dozen-square symmetric solve
// and a dependency would be the larger cost.
//
// The ridge is not optional. Two controls that do the same thing make AᵀA
// singular, and two tone stacks in one chain come close enough that the
// difference is arithmetic noise. λ also does the second job the design asks
// for, which is preferring the smaller of two moves that close the same gap.
func least(
	a [][]float64,
	b []float64,
	lambda float64,
) ([]float64, error) {
	n := len(a[0])

	// AᵀA, symmetric, with the ridge on the diagonal.
	//
	// scale is the diagonal's own size, because the ridge has to be relative to
	// it. An absolute λ damps a control with a small slope hard and one with a
	// large slope barely at all, which made the answer depend on the units a
	// figure happens to be reported in: half a hertz of centroid per hertz of
	// MidFreq undershot by 9% while a tone control did not.
	normal := make([][]float64, n)

	var scale float64

	for i := range normal {
		normal[i] = make([]float64, n+1)

		for j := range n {
			var sum float64
			for r := range a {
				sum += a[r][i] * a[r][j]
			}

			normal[i][j] = sum
		}

		scale += normal[i][i]

		// Aᵀb in the last column, so the elimination carries it along.
		var rhs float64
		for r := range a {
			rhs += a[r][i] * b[r]
		}

		normal[i][n] = rhs
	}

	ridge := lambda * scale / float64(n)
	if ridge <= 0 {
		ridge = lambda
	}

	for i := range n {
		normal[i][i] += ridge
	}

	return eliminate(normal, n)
}

// eliminate solves an augmented square system by Gaussian elimination with
// partial pivoting.
//
// Pivoting on the largest remaining row rather than the first: without it a
// zero or tiny leading entry divides by almost nothing and the answer is
// whatever the rounding said, which for a control that moves one figure only is
// the ordinary case rather than a rare one.
func eliminate(
	m [][]float64,
	n int,
) ([]float64, error) {
	for col := range n {
		pivot := col

		for row := col + 1; row < n; row++ {
			if math.Abs(m[row][col]) > math.Abs(m[pivot][col]) {
				pivot = row
			}
		}

		if math.Abs(m[pivot][col]) < tiny {
			return nil, fmt.Errorf("%w: column %d has no pivot", ErrSingular, col)
		}

		m[col], m[pivot] = m[pivot], m[col]

		for row := col + 1; row < n; row++ {
			f := m[row][col] / m[col][col]
			for k := col; k <= n; k++ {
				m[row][k] -= f * m[col][k]
			}
		}
	}

	out := make([]float64, n)

	for row := n - 1; row >= 0; row-- {
		sum := m[row][n]
		for k := row + 1; k < n; k++ {
			sum -= m[row][k] * out[k]
		}

		out[row] = sum / m[row][row]
	}

	return out, nil
}

// tiny is the pivot below which a column is treated as empty rather than
// divided by. Well under the ridge term, so a ridged system always has one.
const tiny = 1e-12
