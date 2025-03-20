package main

import (
	"fmt"
	"go/ast"
	"reflect"

	"golang.org/x/tools/go/ast/inspector"
	"golang.org/x/tools/go/packages"
)

func main() {
	loadPackage("/Users/adam/work/personal/errstack/vendor/...")
}

func loadPackage(patterns ...string) {
	pkgs, err := packages.Load(&packages.Config{
		Mode: packages.LoadAllSyntax,
	}, patterns...)
	if err != nil {
		panic(err)
	}
	for _, pkg := range pkgs {
		inspector.New(pkg.Syntax).Preorder([]ast.Node{(*ast.CallExpr)(nil)}, func(node ast.Node) {
			call := node.(*ast.CallExpr)
			fmt.Println("CallExpr", reflect.TypeOf(call), reflect.TypeOf(call.Fun))
			printCallExpr(pkg, call, "__")
		})
	}
}

func printCallExpr(pkg *packages.Package, call *ast.CallExpr, prefix string) {
	switch fn := call.Fun.(type) {
	case *ast.SelectorExpr:
		fmt.Println(prefix, "SelectorExpr", reflect.TypeOf(fn.X), reflect.TypeOf(fn.Sel), pkg.Fset.Position(fn.Pos()))
	case *ast.Ident:
		fmt.Println(prefix, "Ident", reflect.TypeOf(fn.Obj), pkg.Fset.Position(fn.Pos()))
	case *ast.ParenExpr:
		fmt.Println(prefix, "ParenExpr", reflect.TypeOf(fn.X), pkg.Fset.Position(fn.Pos()))
	case *ast.FuncLit:
		fmt.Println(prefix, "FuncLit", reflect.TypeOf(fn.Type), reflect.TypeOf(fn.Body), pkg.Fset.Position(fn.Pos()))
	case *ast.IndexExpr:
		fmt.Println(prefix, "IndexExpr", reflect.TypeOf(fn.Index), reflect.TypeOf(fn.X), pkg.Fset.Position(fn.Pos()))
	case *ast.ArrayType:
		fmt.Println(prefix, "ArrayType", reflect.TypeOf(fn.Len), reflect.TypeOf(fn.Elt), pkg.Fset.Position(fn.Pos()))
	case *ast.IndexListExpr:
		fmt.Println(prefix, "IndexListExpr", reflect.TypeOf(fn.X), reflect.TypeOf(fn.Indices), pkg.Fset.Position(fn.Pos()))
	case *ast.CallExpr:
		fmt.Print(prefix, "CallExpr.CallExpr ")
		printCallExpr(pkg, fn, prefix+"__")
	default:
		fmt.Println(prefix, "Unknown", reflect.TypeOf(fn), pkg.Fset.Position(fn.Pos()))
	}
}
