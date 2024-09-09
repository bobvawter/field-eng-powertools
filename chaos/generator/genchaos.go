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

import (
	"fmt"
	"go/types"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"golang.org/x/tools/go/packages"
)

func Cmd() *cobra.Command {
	cfg := packagesConfig()
	ret := &cobra.Command{
		Use:  filepath.Base(os.Args[0]) + " < interface name > ...",
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
	}
	ret.Flags().StringVarP(&cfg.Dir, "dir", "d", ".", "the source directory")
	ret.Flags().StringArrayVarP(&cfg.BuildFlags, "build", "b", []string{"-mod=mod"},
		"arguments to pass to the golang build tool")
	ret.Flags().BoolVarP(&cfg.Tests, "tests", "t", false, "include test code")
	return ret
}

// findType locates a named type within the package and unwraps it until
// the desired return type is found.
func findType[T types.Type](scope *types.Scope, name string) (T, error) {
	found := scope.Lookup(name)
	if found == nil {
		return *new(T), fmt.Errorf("unknown type %s in %s", name, scope.String())
	}
	for typ := found.Type(); typ != nil; typ = typ.Underlying() {
		ret, ok := typ.(T)
		if ok {
			return ret, nil
		}
	}
	return *new(T), fmt.Errorf("type %s: expecting %T, was %T", name, *new(T), found)
}

func packagesConfig() *packages.Config {
	return &packages.Config{
		Mode: packages.NeedDeps | packages.NeedName | packages.NeedTypes | packages.NeedTypesInfo | packages.NeedImports,
	}
}
