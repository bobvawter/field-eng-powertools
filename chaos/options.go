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

// An Option may be passed to [New] or [WithContext] to configure an
// [Engine].
type Option interface{ option(e *Engine) }

type optLimit int

func (o optLimit) option(e *Engine) { e.limit = int32(o) }

// WithLimit sets a limit on the number of times that a unique call
// stack can receive [ErrChaos]. The default limit is 1.
func WithLimit(limit int) Option { return optLimit(limit) }

type optCallback func()

func (o optCallback) option(e *Engine) { e.onChaos = o }

// WithCallback sets a function that will be invoked if [ErrChaos] is
// about to be returned to a caller. The function should be safe to call
// concurrently.
func WithCallback(fn func()) Option { return optCallback(fn) }
