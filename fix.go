package main

import (
	"go/ast"
	"go/token"
	"path"
	"strconv"
	"strings"
)

func fixImports(f *ast.File) {
	declShort := map[string]*ast.ImportSpec{} // key: either base package "fmt", "http" or renamed package
	usedShort := map[string]bool{}            // Same key
	var genDecls []*ast.GenDecl

	addImport := func(ipath string) {
		is := &ast.ImportSpec{
			Path: &ast.BasicLit{
				Kind:  token.STRING,
				Value: strconv.Quote(ipath),
			},
		}
		declShort[path.Base(ipath)] = is

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

	var visitor visitFn
	depth := 0
	visitor = visitFn(func(node ast.Node) ast.Visitor {
		if node == nil {
			depth--
			return visitor
		}
		depth++
		switch v := node.(type) {
		case *ast.GenDecl:
			if v.Tok == token.IMPORT {
				genDecls = append(genDecls, v)
			}
		case *ast.ImportSpec:
			if v.Name != nil {
				declShort[v.Name.Name] = v
			} else {
				local := path.Base(strings.Trim(v.Path.Value, `\"`))
				declShort[local] = v
			}
		case *ast.SelectorExpr:
			if xident, ok := v.X.(*ast.Ident); ok {
				pkgName := xident.Name
				usedShort[pkgName] = true
				if declShort[pkgName] == nil {
					key := pkgName + "." + v.Sel.Name
					if fullImport, ok := common[key]; ok {
						addImport(fullImport)
					}
				}
			}
		}
		// fmt.Printf("%ssaw a %T\n", indent, node)
		return visitor
	})
	ast.Walk(visitor, f)

	// Nil out any unused ImportSpecs, to be removed in following passes
	unusedImport := map[*ast.ImportSpec]bool{}
	for pkg, is := range declShort {
		if !usedShort[pkg] && pkg != "_" && pkg != "." {
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
