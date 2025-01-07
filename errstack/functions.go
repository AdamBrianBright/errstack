package errstack

import (
	"go/ast"
	"go/types"
	"golang.org/x/tools/go/packages"
	"strings"
)

type FunctionInfo map[string][]string

func (f FunctionInfo) Contains(ctx *Ctx, call *ast.CallExpr) bool {
	fn, _ := getFunPkg(ctx, call)
	if fn == nil {
		return false
	}

	for fPkg, fNames := range f {
		if !strings.HasPrefix(fn.Pkg.Path(), fPkg) {
			continue
		}
		for _, fName := range fNames {
			if strings.Contains(fn.Name, fName) {
				return true
			}
		}
	}

	return false
}

// findCallFuncDecl returns *ast.FuncDecl for called function.
func (es *ErrStack) findCallFuncDecl(ctx *Ctx, call *ast.CallExpr) (*Ctx, *ast.FuncDecl) {
	if call.Fun == nil {
		return nil, nil
	}
	fn, _ := getFunPkg(ctx, call)
	if fn == nil {
		return nil, nil
	}
	packageInfo := es.getPackage(fn.Pkg.Path())

	return &Ctx{Info: packageInfo.TypesInfo, Fset: packageInfo.Fset}, findFuncDecl(packageInfo, fn.Name)
}

// findFuncDecl returns *ast.FuncDecl for function type.
func findFuncDecl(pkg *packages.Package, name string) *ast.FuncDecl {
	for _, file := range pkg.Syntax {
		for _, decl := range file.Decls {
			switch decl := decl.(type) {
			case *ast.FuncDecl:
				if decl.Name.Name == name {
					return decl
				}
			}
		}
	}
	return nil
}

type Func struct {
	Pkg  *types.Package
	Name string
}

// getFunPkg returns imported package of function.
func getFunPkg(ctx *Ctx, call *ast.CallExpr) (*Func, bool) {
	switch se := call.Fun.(type) {
	case *ast.SelectorExpr:
		if id, isIdent := se.X.(*ast.Ident); isIdent {
			if selObj := ctx.Info.ObjectOf(id); selObj != nil {
				if pkg, isPkgName := selObj.(*types.PkgName); isPkgName {
					return &Func{Pkg: pkg.Imported(), Name: se.Sel.Name}, true
				}
			}
		}
		t, ok2 := ctx.Info.TypeOf(se.X).(*types.Named)
		if !ok2 || t == nil || t.Obj() == nil {
			return nil, false
		}
		return &Func{Pkg: t.Obj().Pkg(), Name: se.Sel.Name}, false
	case *ast.Ident:
		obj := ctx.Info.ObjectOf(se)
		return &Func{Pkg: obj.Pkg(), Name: obj.Name()}, true
	}

	return nil, false
}
