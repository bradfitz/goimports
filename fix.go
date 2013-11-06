package main

import (
	"go/ast"
	"path"
	"strings"

	"code.google.com/p/go.tools/astutil"
)

func fixImports(f *ast.File) error {
	// refs are a set of possible package references currently unsatisified by imports.
	// first key: either base package (e.g. "fmt") or renamed package
	// second key: referenced package symbol (e.g. "Println")
	refs := make(map[string]map[string]bool)

	// decls are the current package imports. key is base package or renamed package.
	decls := make(map[string]*ast.ImportSpec)

	// collect potential uses of packages.
	var visitor visitFn
	visitor = visitFn(func(node ast.Node) ast.Visitor {
		if node == nil {
			return visitor
		}
		switch v := node.(type) {
		case *ast.ImportSpec:
			if v.Name != nil {
				decls[v.Name.Name] = v
			} else {
				local := path.Base(strings.Trim(v.Path.Value, `\"`))
				decls[local] = v
			}
		case *ast.SelectorExpr:
			xident, ok := v.X.(*ast.Ident)
			if !ok {
				break
			}
			if xident.Obj != nil {
				// if the parser can resolve it, it's not a package ref
				break
			}
			pkgName := xident.Name
			if refs[pkgName] == nil {
				refs[pkgName] = make(map[string]bool)
			}
			if decls[pkgName] == nil {
				refs[pkgName][v.Sel.Name] = true
			}
		}
		return visitor
	})
	ast.Walk(visitor, f)

	// Search for imports matching potential package references.
	for pkgName, symbols := range refs {
		ipath, err := findImport(pkgName, symbols)
		if err != nil {
			return err
		}
		if ipath != "" {
			astutil.AddImport(f, ipath)
		}
	}

	// Nil out any unused ImportSpecs, to be removed in following passes
	unusedImport := map[string]bool{}
	for pkg, is := range decls {
		if refs[pkg] == nil && pkg != "_" && pkg != "." {
			unusedImport[strings.Trim(is.Path.Value, `"`)] = true
		}
	}
	for ipath := range unusedImport {
		astutil.DeleteImport(f, ipath)
	}

	return nil
}

// findImport searches for a package with the given symbols.
// If no package is found, findImport returns "".
// Declared as a variable rather than a function so goimports can be easily
// extended by adding a file with an init function.
var findImport = func(pkgName string, symbols map[string]bool) (string, error) {
	// TODO(crawshaw): Walk GOPATH, use the parser to match symbols.
	var sym string
	for k := range symbols {
		sym = k
		break
	}
	return common[pkgName + "." + sym], nil
}

type visitFn func(node ast.Node) ast.Visitor

func (fn visitFn) Visit(node ast.Node) ast.Visitor {
	return fn(node)
}
