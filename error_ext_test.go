// Copyright (c) 2020 Uber Technologies, Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package multierr_test

import (
	"errors"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/multierr"
)

type errGreatSadness struct{ id int }

func (errGreatSadness) Error() string {
	return "great sadness"
}

type errUnprecedentedFailure struct{ id int }

func (errUnprecedentedFailure) Error() string {
	return "unprecedented failure"
}

func (e errUnprecedentedFailure) Unwrap() error {
	return errRootCause{e.id}
}

type errRootCause struct{ i int }

func (errRootCause) Error() string {
	return "root cause"
}

func TestErrorsWrapping(t *testing.T) {
	err := multierr.Append(
		errGreatSadness{42},
		errUnprecedentedFailure{43},
	)

	t.Run("left", func(t *testing.T) {
		t.Run("As", func(t *testing.T) {
			var got errGreatSadness
			require.True(t, errors.As(err, &got))
			assert.Equal(t, 42, got.id)
		})

		t.Run("Is", func(t *testing.T) {
			assert.False(t, errors.Is(err, errGreatSadness{41}))
			assert.True(t, errors.Is(err, errGreatSadness{42}))
		})
	})

	t.Run("right", func(t *testing.T) {
		t.Run("As", func(t *testing.T) {
			var got errUnprecedentedFailure
			require.True(t, errors.As(err, &got))
			assert.Equal(t, 43, got.id)
		})

		t.Run("Is", func(t *testing.T) {
			assert.False(t, errors.Is(err, errUnprecedentedFailure{42}))
			assert.True(t, errors.Is(err, errUnprecedentedFailure{43}))
		})
	})

	t.Run("top-level", func(t *testing.T) {
		t.Run("As", func(t *testing.T) {
			var got interface{ Errors() []error }
			require.True(t, errors.As(err, &got))
			assert.Len(t, got.Errors(), 2)
		})

		t.Run("Is", func(t *testing.T) {
			assert.True(t, errors.Is(err, err))
		})
	})

	t.Run("root cause", func(t *testing.T) {
		t.Run("As", func(t *testing.T) {
			var got errRootCause
			require.True(t, errors.As(err, &got))
			assert.Equal(t, 43, got.i)
		})

		t.Run("Is", func(t *testing.T) {
			assert.False(t, errors.Is(err, errRootCause{42}))
			assert.True(t, errors.Is(err, errRootCause{43}))
		})
	})

	t.Run("mismatch", func(t *testing.T) {
		t.Run("As", func(t *testing.T) {
			var got *os.PathError
			assert.False(t, errors.As(err, &got))
		})

		t.Run("Is", func(t *testing.T) {
			assert.False(t, errors.Is(err, errors.New("great sadness")))
		})
	})
}

func TestErrorsWrappingSameType(t *testing.T) {
	err := multierr.Combine(
		errGreatSadness{1},
		errGreatSadness{2},
		errGreatSadness{3},
	)

	t.Run("As returns first", func(t *testing.T) {
		var got errGreatSadness
		require.True(t, errors.As(err, &got))
		assert.Equal(t, 1, got.id)
	})

	t.Run("Is matches all", func(t *testing.T) {
		assert.True(t, errors.Is(err, errGreatSadness{1}))
		assert.True(t, errors.Is(err, errGreatSadness{2}))
		assert.True(t, errors.Is(err, errGreatSadness{3}))
	})
}

type dummyCustomGroup struct {
	errors []error
}

func (d dummyCustomGroup) Error() string   { return "custom group" }
func (d dummyCustomGroup) Unwrap() []error { return d.errors }

func TestMultierrAs(t *testing.T) {
	t.Run("nil error", func(t *testing.T) {
		var got errGreatSadness
		assert.False(t, multierr.As(nil, &got))
	})

	t.Run("single matching error", func(t *testing.T) {
		var got errGreatSadness
		err := errGreatSadness{42}
		require.True(t, multierr.As(err, &got))
		assert.Equal(t, 42, got.id)
	})

	t.Run("single mismatching error", func(t *testing.T) {
		var got errGreatSadness
		err := errors.New("other error")
		assert.False(t, multierr.As(err, &got))
	})

	t.Run("combined multierr left and right", func(t *testing.T) {
		err := multierr.Append(
			errGreatSadness{42},
			errUnprecedentedFailure{43},
		)

		var sadness errGreatSadness
		require.True(t, multierr.As(err, &sadness))
		assert.Equal(t, 42, sadness.id)

		var failure errUnprecedentedFailure
		require.True(t, multierr.As(err, &failure))
		assert.Equal(t, 43, failure.id)

		var rootCause errRootCause
		require.True(t, multierr.As(err, &rootCause))
		assert.Equal(t, 43, rootCause.i)

		var pathErr *os.PathError
		assert.False(t, multierr.As(err, &pathErr))
	})

	t.Run("combined same type returns first", func(t *testing.T) {
		err := multierr.Combine(
			errGreatSadness{1},
			errGreatSadness{2},
			errGreatSadness{3},
		)

		var got errGreatSadness
		require.True(t, multierr.As(err, &got))
		assert.Equal(t, 1, got.id)
	})

	t.Run("custom errorGroup implementation", func(t *testing.T) {
		grp := dummyCustomGroup{
			errors: []error{
				errors.New("generic"),
				errGreatSadness{99},
			},
		}

		var got errGreatSadness
		require.True(t, multierr.As(grp, &got))
		assert.Equal(t, 99, got.id)
	})
}
