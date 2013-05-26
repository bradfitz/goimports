package main

import (
	"go/ast"
	"go/token"
	"path"
	"strconv"
	"strings"
)

func fixImports(f *ast.File) {
	sawShort := map[string]bool{} // "fmt" => true, "pathpkg" (renamed) => true
	var imports []*ast.GenDecl

	addImport := func(ipath string) {
		sawShort[path.Base(ipath)] = true
		is := &ast.ImportSpec{
			Path: &ast.BasicLit{
				Kind:  token.STRING,
				Value: strconv.Quote(ipath),
			},
		}
		if len(imports) == 0 {
			imports = append(imports, &ast.GenDecl{
				Tok: token.IMPORT,
			})
			f.Decls = append([]ast.Decl{imports[0]}, f.Decls...)
			f.Imports = append(f.Imports, is)
		}
		imports[0].Specs = append(imports[0].Specs, is)
		if len(imports[0].Specs) > 1 && imports[0].Lparen == 0 {
			imports[0].Lparen = 1 // something not zero
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
				imports = append(imports, v)
			}
		case *ast.ImportSpec:
			if v.Name != nil {
				sawShort[v.Name.Name] = true
			} else {
				local := path.Base(strings.Trim(v.Path.Value, `\"`))
				sawShort[local] = true
			}
		case *ast.SelectorExpr:
			if xident, ok := v.X.(*ast.Ident); ok {
				pkgName := xident.Name
				if !sawShort[pkgName] {
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
}

type visitFn func(node ast.Node) ast.Visitor

func (fn visitFn) Visit(node ast.Node) ast.Visitor {
	return fn(node)
}
