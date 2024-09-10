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

package generator

// This file contains the model types passed into the template.

type method struct {
	Args         []*param // Input arguments.
	HasContext   bool     // Enable call to Chaos(ctx)
	ReturnsError bool     // Enable call to Engine.Chaos()
	Name         string   // Name as it appears in the source code.
	Rets         []*param // Return types.
}

type goPackage struct {
	// A distinct import name like "chaos0".
	Import string
	// Set to true when the import name matches the package name.
	Simple bool
	// The fully-qualified package name used for imports.
	Path string
}

type param struct {
	Name string
	Type *typeName
}

// A Target is the interface proxy type being generated.
type target struct {
	Delegate *typeName // The interface type to be wrapped.
	Methods  []*method // The methods in the interface.
	Impl     *typeName // The type being generated.
}

type typeName struct {
	Qualified string // The type's name within the package.
	Short     string // The short name of the type.
}
