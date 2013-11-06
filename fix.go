package main

import (
	"go/ast"
	"go/token"
	"path"
	"strconv"
	"strings"
)

func fixImports(f *ast.File) error {
	// refs are a set of possible package references currently unsatisified by imports.
	// first key: either base package (e.g. "fmt") or renamed package
	// second key: referenced package symbol (e.g. "Println")
	refs := make(map[string]map[string]bool)

	// decls are the current package imports. key is base package or renamed package.
	decls := make(map[string]*ast.ImportSpec)
	var genDecls []*ast.GenDecl

	// collect potential uses of packages.
	var visitor visitFn
	visitor = visitFn(func(node ast.Node) ast.Visitor {
		if node == nil {
			return visitor
		}
		switch v := node.(type) {
		case *ast.GenDecl:
			if v.Tok == token.IMPORT {
				genDecls = append(genDecls, v)
			}
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

	addImport := func(ipath string) {
		is := &ast.ImportSpec{
			Path: &ast.BasicLit{
				Kind:  token.STRING,
				Value: strconv.Quote(ipath),
			},
		}
		decls[path.Base(ipath)] = is

		if len(genDecls) == 0 {
			genDecls = append(genDecls, &ast.GenDecl{
				Tok: token.IMPORT,
			})
			f.Decls = append([]ast.Decl{genDecls[0]}, f.Decls...)
			f.Imports = append(f.Imports, is)
		}
		gd0 := genDecls[0]
		// Prepend onto gd0.Specs:
		// Make room for it (nil), slide everything down, then set [0].
		gd0.Specs = append(gd0.Specs, nil)
		copy(gd0.Specs[1:], gd0.Specs[:])
		gd0.Specs[0] = is

		if len(gd0.Specs) > 1 && gd0.Lparen == 0 {
			gd0.Lparen = 1 // something not zero
		}
	}

	// Search for imports matching potential package references.
	for pkgName, symbols := range refs {
		fullImport, err := findImport(pkgName, symbols)
		if err != nil {
			return err
		}
		if fullImport != "" {
			addImport(fullImport)
		}
	}

	// Nil out any unused ImportSpecs, to be removed in following passes
	unusedImport := map[*ast.ImportSpec]bool{}
	for pkg, is := range decls {
		if refs[pkg] == nil && pkg != "_" && pkg != "." {
			unusedImport[is] = true
		}
	}

	for _, gd := range genDecls {
		gd.Specs = filterUnusedSpecs(unusedImport, gd.Specs)
		if len(gd.Specs) == 1 {
			gd.Lparen = 0
		}
	}

	f.Decls = filterEmptyDecls(f.Decls)
	f.Imports = filterUnusedImports(unusedImport, f.Imports)
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

func filterUnusedSpecs(unused map[*ast.ImportSpec]bool, in []ast.Spec) (out []ast.Spec) {
	for _, spec := range in {
		if is, ok := spec.(*ast.ImportSpec); ok && unused[is] {
			continue
		}
		out = append(out, spec)
	}
	return
}

func filterUnusedImports(unused map[*ast.ImportSpec]bool, in []*ast.ImportSpec) (out []*ast.ImportSpec) {
	for _, spec := range in {
		if unused[spec] {
			continue
		}
		out = append(out, spec)
	}
	return
}

func filterEmptyDecls(in []ast.Decl) (out []ast.Decl) {
	for _, decl := range in {
		if gd, ok := decl.(*ast.GenDecl); ok && gd.Tok == token.IMPORT && len(gd.Specs) == 0 {
			continue
		}
		out = append(out, decl)
	}
	return
}

type visitFn func(node ast.Node) ast.Visitor

func (fn visitFn) Visit(node ast.Node) ast.Visitor {
	return fn(node)
}
