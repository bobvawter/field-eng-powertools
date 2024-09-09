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
	"bytes"
	"context"
	"encoding/json"
	"go/format"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTemplate(t *testing.T) {
	r := require.New(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := packagesConfig()
	cfg.Dir = "./testdata"
	temp, err := NewTemplate(ctx, cfg, "foobar", []string{"MyInterface", "io.Reader", "database/sql/driver.Conn"})
	r.NoError(err)

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", " ")
	r.NoError(enc.Encode(temp))

	var buf bytes.Buffer
	r.NoError(intfTemplate.Execute(&buf, temp))
	out, err := format.Source(buf.Bytes())
	if err != nil {
		t.Log(buf.String())
	}
	r.NoError(err)
	t.Log(string(out))
}
