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
	"runtime"
	"sync"
	"sync/atomic"
)

const (
	defaultLimit = 1
	defaultSkip  = 0
)

// StackDepth contains the maximum number of call stack entries that
// will be used to determine uniqueness.
const StackDepth = 25

type key [StackDepth]uintptr
type entry struct {
	count atomic.Int32
	done  atomic.Bool
}

// An Engine tracks unique call stacks.
type Engine struct {
	entries sync.Map // Use-case 1: A cache that only grows.
	limit   int32    // Return this many chaos errors.
	skip    int32    // Number of chaos calls that won't return an error.
	onChaos Callback // User-defined behavior.
}

// New constructs a new Engine with the provided options.
func New(opts ...Option) *Engine {
	ret := &Engine{limit: defaultLimit, skip: defaultSkip}
	for _, opt := range opts {
		opt.option(ret)
	}
	return ret
}

// Chaos will return an *[Error] a configured number of times for each
// unique call stack that invokes the Chaos method. The number of frames
// considered for uniqueness is set by [StackDepth].
func (e *Engine) Chaos() error {
	return e.chaos(3)
}

// This is also called from the top-level [Chaos] function.
func (e *Engine) chaos(callers int) error {
	var stack key
	frames := runtime.Callers(callers, stack[:])

	found, ok := e.entries.Load(stack)
	if !ok {
		proposed := &entry{}
		proposed.count.Add(-e.skip)
		found, _ = e.entries.LoadOrStore(stack, proposed)
	}

	counter := found.(*entry)

	// Short-circuit to prevent unbounded counting.
	if counter.done.Load() {
		return nil
	}
	next := counter.count.Add(1)
	if next <= 0 {
		// Check for skipping.
		return nil
	}
	if next > e.limit {
		// It's possible that the counter could over-shoot slightly if
		// multiple callers hit Add() call concurrently.
		counter.done.Store(true)
		return nil
	}
	err := &Error{Stack: stack[:frames]}
	// Invoke user callback, if defined.
	if fn := e.onChaos; fn != nil {
		return fn(err)
	}
	return err
}
