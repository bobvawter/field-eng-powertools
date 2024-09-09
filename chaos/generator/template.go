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
	_ "embed"
	"fmt"
	"go/types"
	"strings"
	"text/template"

	"golang.org/x/tools/go/packages"
)

//go:embed intf.tmpl
var templateSource string

var intfTemplate = template.Must(template.New("").Funcs(template.FuncMap{
	"nl":  func() string { return "\n" },
	"sp":  func() string { return " " },
	"sub": func(a, b int) int { return a - b },
}).Parse(templateSource))

type Package struct {
	// A distinct short name like "chaos0".
	Short string
	// Set to true when the import name matches the package name.
	Simple bool
	// The fully-qualified package name used for imports.
	Path string
}

// universe represents ambbient symbols.
var universe = &Package{}

type Method struct {
	Args         []*Param // Input arguments.
	HasContext   bool     // Enable call to Chaos(ctx)
	ReturnsError bool     // Enable call to Engine.Chaos()
	Name         string   // Name as it appears in the source code.
	Rets         []*Param // Return types.
}

type Param struct {
	Name string
	Type *TypeName
}

// A Target is the interface proxy type being generated.
type Target struct {
	Delegate *TypeName // The interface type to be wrapped.
	Methods  []*Method // The methods in the interface.
	Impl     *TypeName // The type being generated.
}

type TypeName struct {
	Package     *Package // The package which defines the type.
	Prefix      string   // Handles pointers and slices
	Name        string   // The type's name within the package.
	Unqualified bool     // The type is declared within the destination package.
}

// Returns the local import name for the type.
func (t *TypeName) String() string {
	name := t.Prefix
	if t.Unqualified || t.Package.Short == "" {
		name += t.Name
	} else {
		name += t.Package.Short + "." + t.Name
	}
	return name
}

type Template struct {
	Chaos   *Package            // A reference to the chaos impl package.
	Cmd     string              // How to invoke the generator again.
	Imports map[string]*Package // Prevent collisions in short import names.
	Package *Package            // The enclosing package for the generated type.
	Targets []*Target           // The type(s) being generated.

	ctxTypes map[*types.Interface]struct{} // The context.Context type.
	errType  *types.Interface              // The built-in "error" type.

	packages map[*types.Package]*Package // Memoize template package values.
	types    map[types.Type]*TypeName    // Memoize type data.
}

const chaosPackageName = "github.com/cockroachdb/field-eng-powertools/chaos"

func NewTemplate(ctx context.Context, cfg *packages.Config, destPkgName string, intfNames []string) (*Template, error) {
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

	chaosPackage := &Package{
		Short:  "chaos",
		Simple: true,
		Path:   chaosPackageName,
	}
	ret := &Template{
		Chaos: chaosPackage,
		Cmd:   "xyzzy",
		Package: &Package{
			Short:  destPkgName,
			Simple: true,
			Path:   "",
		},
		Targets: make([]*Target, 0, len(intfNames)),

		ctxTypes: map[*types.Interface]struct{}{},
		packages: map[*types.Package]*Package{},
		Imports: map[string]*Package{
			chaosPackage.Short: chaosPackage,
		},
		types: map[types.Type]*TypeName{},
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

		target := &Target{
			Delegate: ret.typeNameFor(namedType),
			Impl: &TypeName{
				Name:        "chaotic" + namedType.Obj().Name() + "Impl",
				Package:     ret.Package,
				Unqualified: true,
			},
		}

		for i := range tgtIntf.NumMethods() {
			target.Methods = append(target.Methods, ret.methodFor(tgtIntf.Method(i)))
		}

		ret.Targets = append(ret.Targets, target)
	}

	return ret, nil
}

func (t *Template) methodFor(m *types.Func) *Method {
	sig := m.Signature()
	return &Method{
		Args:         t.tupleFor(sig.Params()),
		HasContext:   t.hasContext(sig),
		Name:         m.Name(),
		Rets:         t.tupleFor(sig.Results()),
		ReturnsError: t.returnsError(sig),
	}
}

// hasContext returns true if the method accepts a context as the first
// argument.
func (t *Template) hasContext(sig *types.Signature) bool {
	params := sig.Params()
	if params.Len() == 0 {
		return false
	}
	first := params.At(0).Type()
	for ctxType := range t.ctxTypes {
		if types.Implements(first, ctxType) {
			return true
		}
	}
	return false
}

// returnsError returns true if the method returns error as the last
// return type.
func (t *Template) returnsError(sig *types.Signature) bool {
	res := sig.Results()
	if res.Len() == 0 {
		return false
	}
	last := res.At(res.Len() - 1).Type()
	// The last return type must be exactly "error".
	return types.AssignableTo(t.errType, last) && types.AssignableTo(last, t.errType)
}

func (t *Template) packageFor(pkg *types.Package) *Package {
	if found, ok := t.packages[pkg]; ok {
		return found
	}
	// We'll see this for "error" and other builtin types.
	if pkg == nil {
		return universe
	}
	ret := &Package{
		Short:  pkg.Name(),
		Simple: true,
		Path:   pkg.Path(),
	}
	// Ensure short names are unique.
	for i := 1; true; i++ {
		if _, collision := t.Imports[ret.Short]; !collision {
			t.Imports[ret.Short] = ret
			break
		}
		ret.Short = fmt.Sprintf("%s%d", pkg.Name(), i)
		ret.Simple = false
	}
	t.packages[pkg] = ret
	return ret
}

func (t *Template) tupleFor(tup *types.Tuple) []*Param {
	ret := make([]*Param, tup.Len())
	for i := range ret {
		ret[i] = &Param{
			Name: tup.At(i).Name(),
			Type: t.typeNameFor(tup.At(i).Type()),
		}
	}
	return ret
}

func (t *Template) typeNameFor(typ types.Type) *TypeName {
	if found, ok := t.types[typ]; ok {
		return found
	}

	var ret *TypeName
outer:
	for try := typ; try != nil; try = try.Underlying() {
		var prefix string
		switch k := try.(type) {
		case *types.Pointer:
			try = k.Elem()
			prefix += "*"

		case *types.Slice:
			try = k.Elem()
			prefix += "[]"
		}

		switch k := try.(type) {
		case *types.Named:
			obj := k.Obj()
			pkg := t.packageFor(obj.Pkg())
			ret = &TypeName{
				Name:        obj.Name(),
				Package:     pkg,
				Prefix:      prefix,
				Unqualified: pkg == t.Package,
			}
			break outer

		case *types.Basic:
			ret = &TypeName{
				Name:        prefix + k.Name(),
				Package:     universe,
				Unqualified: true,
			}
			break outer
		}
	}
	if ret == nil {
		panic(fmt.Errorf("unimplemented: %T", typ))
	}
	t.types[typ] = ret
	return ret
}
