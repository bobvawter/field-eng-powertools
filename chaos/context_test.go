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
	"fmt"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestContext(t *testing.T) {
	r := require.New(t)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// No-op for undecorated contexts.
	r.Nil(Chaos(ctx))

	chaotic := WithEngine(ctx, New())

	// Validate default limit behavior.
	var stack []uintptr
	for i := range defaultLimit + 1 {
		err := chaos(chaotic)
		if i < defaultLimit {
			r.ErrorIs(err, ErrChaos)
			var impl *Error
			r.ErrorAs(err, &impl)
			stack = impl.Stack
		} else {
			r.NoError(err)
		}
	}

	// Verify we're capturing the expected top caller frame.
	e, ok := FromContext(chaotic)
	r.True(ok)
	r.NotNil(e)

	frames := runtime.CallersFrames(stack)
	top, _ := frames.Next()
	r.Truef(strings.HasSuffix(top.Function, "TestContext"), "%s", top.Function)
}

// An [Engine] may be associated with a Context, which means that chaos
// can be added to existing code by replacing the "return nil" case with
// "return Chaos(ctx)".
func Example_context() {
	// This function returns a constant value based on build tags.
	if Enabled() {
		fmt.Println("chaos enabled")
	}

	// This demonstrates that the Engine looks at the call stack and not
	// just an individual call site.
	doStuff := func(ctx context.Context) error {
		// This Chaos call will be a no-op if Enabled() returns false.
		return Chaos(ctx)
	}

	eng := New(
		WithLimit(2), // Emit 2 errors per unique stack trace.
		WithSkip(1),  // Don't emit errors the first time a call is made.
	)

	ctx := WithEngine(context.Background(), eng)
	for range 4 {
		err := doStuff(ctx)
		fmt.Printf("call1: %v\n", err != nil)
	}
	fmt.Println()
	for range 4 {
		// This call to Chaos has a different stack, so it will generate
		// different results.
		err := doStuff(ctx)
		fmt.Printf("call2: %v\n", err != nil)
	}

	// Output:
	// chaos enabled
	// call1: false
	// call1: true
	// call1: true
	// call1: false
	//
	// call2: false
	// call2: true
	// call2: true
	// call2: false
}
