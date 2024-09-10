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
	_ "embed"
	"fmt"
	"go/format"
	"go/types"
	"os"
	"strings"
	"text/template"

	"golang.org/x/tools/go/packages"
)

const chaosPackageName = "github.com/cockroachdb/field-eng-powertools/chaos"

//go:embed intf.tmpl
var templateSource string

var intfTemplate = template.Must(template.New("").Funcs(template.FuncMap{
	"nl":  func() string { return "\n" },
	"sp":  func() string { return " " },
	"sub": func(a, b int) int { return a - b },
}).Parse(templateSource))

// universe contains ambient symbols, like "int" and "error".
var universe = &goPackage{}

type generator struct {
	Chaos   *goPackage            // A reference to the chaos impl package.
	Cmd     string                // How to invoke the generator again.
	Imports map[string]*goPackage // Prevent collisions in short import names.
	Package *goPackage            // The enclosing package for the generated type.
	Targets []*target             // The type(s) being generated.

	ctxTypes map[*types.Interface]struct{} // The context.Context type.
	errType  *types.Interface              // The built-in "error" type.

	packages map[*types.Package]*goPackage // Memoize template package values.
	types    map[types.Type]*typeName      // Memoize type data.
}

func newGenerator(ctx context.Context, cfg *packages.Config, destPkgName string, intfNames []string) (*generator, error) {
	request := map[string][]string{
		chaosPackageName: nil,
	}
	for _, intfName := range intfNames {
		dotIdx := strings.Index(intfName, ".")

		if dotIdx == -1 {
			// Relative to the config's working directory.
			request["."] = append(request["."], intfName)
			continue
		}
		pkgName := intfName[:dotIdx]
		request[pkgName] = append(request[pkgName], intfName[dotIdx+1:])
	}

	chaosPackage := &goPackage{
		Import: "chaos",
		Simple: true,
		Path:   chaosPackageName,
	}
	ret := &generator{
		Chaos: chaosPackage,
		Cmd:   "xyzzy",
		Package: &goPackage{
			Import: destPkgName,
			Simple: true,
		},
		Targets: make([]*target, 0, len(intfNames)),

		ctxTypes: map[*types.Interface]struct{}{},
		packages: map[*types.Package]*goPackage{},
		Imports: map[string]*goPackage{
			chaosPackage.Import: chaosPackage,
		},
		types: map[types.Type]*typeName{},
	}

	var toGenerate []*types.Named

	cfg.Context = ctx
	for pkgName, intfNames := range request {
		pkgs, err := packages.Load(cfg, pkgName)
		if err != nil {
			return nil, err
		}
		pkg := pkgs[0]
		for _, imported := range pkg.Imports {
			if imported.PkgPath == "context" {
				found, err := findType[*types.Interface](imported.Types.Scope(), "Context")
				if err != nil {
					return nil, err
				}
				ret.ctxTypes[found] = struct{}{}
			}
		}
		for _, intfName := range intfNames {
			found, err := findType[*types.Named](pkg.Types.Scope(), intfName)
			if err != nil {
				return nil, err
			}
			toGenerate = append(toGenerate, found)
		}
	}

	// Universe contains ambient types.
	var err error
	ret.errType, err = findType[*types.Interface](types.Universe, "error")
	if err != nil {
		return nil, err
	}

	for _, namedType := range toGenerate {
		tgtIntf, ok := namedType.Underlying().(*types.Interface)
		if !ok {
			return nil, fmt.Errorf("%s is not an interface type", namedType.Obj().Id())
		}
		implName := "chaotic" + namedType.Obj().Name() + "Impl"

		target := &target{
			Delegate: ret.typeNameFor(namedType),
			Impl: &typeName{
				Short:     implName,
				Qualified: implName,
			},
		}

		for i := range tgtIntf.NumMethods() {
			method := tgtIntf.Method(i)
			target.Methods = append(target.Methods, ret.methodFor(method))
		}

		ret.Targets = append(ret.Targets, target)
	}

	return ret, nil
}

func (g *generator) generate() ([]byte, error) {
	var buf bytes.Buffer
	if err := intfTemplate.Execute(&buf, g); err != nil {
		return nil, err
	}
	out, err := format.Source(buf.Bytes())
	if err != nil {
		_, _ = buf.WriteTo(os.Stdout)
	}
	return out, err
}

// hasContext returns true if the method accepts a context as the first
// argument.
func (g *generator) hasContext(sig *types.Signature) bool {
	params := sig.Params()
	if params.Len() == 0 {
		return false
	}
	first := params.At(0).Type()
	for ctxType := range g.ctxTypes {
		if types.Implements(first, ctxType) {
			return true
		}
	}
	return false
}

func (g *generator) methodFor(m *types.Func) *method {
	sig := m.Signature()
	return &method{
		Args:         g.tupleFor(sig.Params()),
		HasContext:   g.hasContext(sig),
		Name:         m.Name(),
		Rets:         g.tupleFor(sig.Results()),
		ReturnsError: g.returnsError(sig),
	}
}

func (g *generator) packageFor(pkg *types.Package) *goPackage {
	if found, ok := g.packages[pkg]; ok {
		return found
	}
	// We'll see this for "error" and other builtin types.
	if pkg == nil {
		return universe
	}
	ret := &goPackage{
		Import: pkg.Name(),
		Simple: true,
		Path:   pkg.Path(),
	}
	// Ensure short names are unique.
	for i := 1; true; i++ {
		if _, collision := g.Imports[ret.Import]; !collision {
			g.Imports[ret.Import] = ret
			break
		}
		ret.Import = fmt.Sprintf("%s%d", pkg.Name(), i)
		ret.Simple = false
	}
	g.packages[pkg] = ret
	return ret
}

// returnsError returns true if the method returns error as the last
// return type.
func (g *generator) returnsError(sig *types.Signature) bool {
	res := sig.Results()
	if res.Len() == 0 {
		return false
	}
	last := res.At(res.Len() - 1).Type()
	// The last return type must be exactly "error".
	return types.AssignableTo(g.errType, last) && types.AssignableTo(last, g.errType)
}

func (g *generator) tupleFor(tup *types.Tuple) []*param {
	ret := make([]*param, tup.Len())
	for i := range ret {
		ret[i] = &param{
			Name: tup.At(i).Name(),
			Type: g.typeNameFor(tup.At(i).Type()),
		}
	}
	return ret
}

func (g *generator) typeNameFor(typ types.Type) *typeName {
	if found, ok := g.types[typ]; ok {
		return found
	}
	// A fully-qualified representation: []*foo.Bar
	qName := types.TypeString(typ, func(p *types.Package) string {
		found := g.packageFor(p)
		return found.Import
	})

	// The short name of a named type: "Bar"
	var sName string
	if named, ok := typ.(*types.Named); ok {
		sName = named.Obj().Name()
	}

	return &typeName{
		Qualified: qName,
		Short:     sName,
	}
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
		Mode: packages.NeedTypes,
	}
}
