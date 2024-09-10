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

import "errors"

// ErrChaos can be used with [errors.Is] to detect errors returned by
// this package.
var ErrChaos = errors.New("chaos")

// Error is returned by [Chaos] and [Engine.Chaos].
type Error struct {
	// The call stack which resulted in the error. This can be passed to
	// [runtime.CallersFrames] for further inspection.
	Stack []uintptr
}

// Error implements error.
func (e *Error) Error() string { return "chaos" }

// Is returns true if the argument is [ErrChaos].
func (e *Error) Is(err error) bool { return err == ErrChaos }

// Unwrap returns [ErrChaos].
func (e *Error) Unwrap() error { return ErrChaos }
