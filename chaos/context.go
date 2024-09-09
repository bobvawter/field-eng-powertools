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

import "context"

type engineKey struct{}

var background = context.WithValue(context.Background(),
	engineKey{}, &Engine{limit: defaultLimit})

// Background is a pre-defined global, analogous to
// [context.Background]. The [Engine] associated with the context has a
func Background() context.Context {
	return background
}

// Chaos may return [ErrChaos] if the build is [Enabled] and the context
// is associated with an [Engine].
func Chaos(ctx context.Context) error {
	if !Enabled() {
		return nil
	}
	if e, ok := FromContext(ctx); ok {
		return e.chaos(3)
	}
	return nil
}

// FromContext returns the [Engine] associated with the given context.
func FromContext(ctx context.Context) (*Engine, bool) {
	if !Enabled() {
		return nil, false
	}
	found := ctx.Value(engineKey{})
	e, ok := found.(*Engine)
	return e, ok
}

// WithContext returns a context that has an associated [Engine] if a
// chaos build is [Enabled]. This function will return the argument if
// it is already associated with an [Engine].
func WithContext(ctx context.Context, opts ...Option) context.Context {
	if !Enabled() {
		return ctx
	}
	if _, hasEngine := FromContext(ctx); hasEngine {
		return ctx
	}
	return context.WithValue(ctx, engineKey{}, New(opts...))
}
