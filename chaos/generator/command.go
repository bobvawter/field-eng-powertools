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
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

// Command returns the generator.
func Command() *cobra.Command {
	cfg := packagesConfig()
	var dir, pkgOverride, outFile string
	ret := &cobra.Command{
		Use:  filepath.Base(os.Args[0]) + " < interface name > ...",
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			absDir, err := filepath.Abs(cfg.Dir)
			if err != nil {
				return err
			}
			cfg.Dir = absDir

			absOut, err := filepath.Abs(filepath.Join(absDir, outFile))
			if err != nil {
				return err
			}

			pkgName := pkgOverride
			if pkgName == "" {
				pkgName = filepath.Base(filepath.Dir(absOut))
			}

			gen, err := newGenerator(cmd.Context(), cfg, pkgName, args)
			if err != nil {
				return err
			}

			data, err := gen.generate()
			if err != nil {
				return err
			}

			return os.WriteFile(absOut, data, 0644)
		},
	}
	ret.Flags().StringArrayVarP(&cfg.BuildFlags, "build", "b", nil,
		"arguments to pass to the golang build tool")
	ret.Flags().StringVarP(&dir, "dir", "d", ".", "a source directory")
	ret.Flags().StringVarP(&outFile, "out", "o", "chaos_gen.go",
		"the name of the generated file")
	ret.Flags().StringVarP(&pkgOverride, "package", "p", "",
		"override the generated package name")
	return ret
}
