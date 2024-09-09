// Copyright 2024 The Cockroach Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// SPDX-License-Identifier: Apache-2.0

package chaos

import (
	"context"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"
)

func TestEngine(t *testing.T) {
	const expected = 1024
	r := require.New(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var count atomic.Int32
	e := New(
		WithLimit(expected),
		WithCallback(func() {
			count.Add(1)
		}))

	var errorsSeen atomic.Int32
	var sawNil atomic.Bool
	eg, _ := errgroup.WithContext(ctx)
	for range expected + 1 {
		eg.Go(func() error {
			err := callChaos(e)
			if err != nil {
				r.ErrorIs(err, ErrChaos)
				errorsSeen.Add(1)
			} else {
				sawNil.Store(true)
			}
			return nil
		})
	}
	r.NoError(eg.Wait())
	r.True(sawNil.Load())
	r.Equal(int32(expected), count.Load())

	// This is a different call stack, so we expect to see an error.
	r.ErrorIs(callChaos(e), ErrChaos)
	r.Equal(int32(expected+1), count.Load())

	// We have two distinct call stacks.
	entryCount := 0
	e.entries.Range(func(k, _ any) bool {
		callers := k.(key)
		frames := runtime.CallersFrames(callers[:])
		top, _ := frames.Next()
		r.Truef(strings.HasSuffix(top.Function, "chaos.callChaos"), "%s", top.Function)
		entryCount++
		return true
	})
	r.Equal(2, entryCount)
}

// callChaos ensures that we have a stably-named call site for ensuring
// that we're trimming the correct number of callers from the captured
// stacks.
func callChaos(e *Engine) error {
	return e.Chaos()
}
