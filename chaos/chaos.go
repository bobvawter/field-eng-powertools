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

// Package chaos improves test coverage by deterministically injecting
// errors based on unique call stacks.
//
// The behavior in this package is guarded by a build tag. This ensures
// that production builds cannot be negatively impacted. See [Enabled].
package chaos

// Enabled returns true if the "chaos_enabled" build tag was set or if
// the "chaos_env" build tag was set and a "CHAOS_ENABLED" environment
// variable is defined.
func Enabled() bool {
	return enabled
}
