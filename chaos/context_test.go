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
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestContext(t *testing.T) {
	r := require.New(t)

	r.True(Enabled(), "chaos build not enabled")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// No-op for undecorated contexts.
	r.Nil(Chaos(ctx))

	chaotic := WithContext(ctx)

	// Once there is a chaos context, subsequent calls to WithContext
	// should be a no-op.
	mid := context.WithValue(chaotic, "foo", "bar")
	r.Same(mid, WithContext(mid))

	// Validate default limit behavior.
	for i := range defaultLimit + 1 {
		err := Chaos(chaotic)
		if i < defaultLimit {
			r.ErrorIs(err, ErrChaos)
		} else {
			r.NoError(err)
		}
	}

	// Verify we're capturing the expected top caller frame.
	e, ok := FromContext(chaotic)
	r.True(ok)
	r.NotNil(e)
	e.entries.Range(func(k, _ any) bool {
		callers := k.(key)
		frames := runtime.CallersFrames(callers[:])
		top, _ := frames.Next()
		r.Truef(strings.HasSuffix(top.Function, "TestContext"), "%s", top.Function)
		return true
	})

	r.Same(Background(), WithContext(Background()))
	backgroundEngine, ok := FromContext(Background())
	r.True(ok)
	r.Equal(int32(defaultLimit), backgroundEngine.limit)
}
