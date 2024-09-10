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
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/tools/go/packages"
)

func TestTemplate(t *testing.T) {
	r := require.New(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	testdata, err := filepath.Abs("./testdata")
	r.NoError(err)

	cfg := packagesConfig()
	cfg.Dir = testdata
	gen, err := newGenerator(ctx, cfg, "foobar", []string{
		"MyInterface",         // Test method patterns.
		"net.Conn",            // Verify arbitrary interfaces.
		"io.ReadCloser",       // Verify composite interfaces.
		"math/rand/v2.Source", // Verify versioned import.
	})
	r.NoError(err)

	out, err := gen.generate()
	r.NoError(err)

	// Perform a complete analysis of the generated file. This will
	// verify that the generated file would be accepted by the compiler.
	testConfig := &packages.Config{
		Dir:  testdata,
		Mode: packages.NeedSyntax | packages.NeedTypes,
		Overlay: map[string][]byte{
			filepath.Join(testdata, "foobar", "foobar.go"): out,
		},
	}
	reloaded, err := packages.Load(testConfig, "foobar/foobar.go")
	r.NoError(err)
	r.Empty(reloaded[0].Errors)
	r.Empty(reloaded[0].TypeErrors)
}
