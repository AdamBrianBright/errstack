package errstack

import (
	"go/ast"
	"go/types"
	"log"
	"strings"
)

type Func struct {
	Ctx     *Ctx
	CallCtx *Ctx
	Name    string
	Method  string
	Pkg     *types.Package
	Call    *ast.CallExpr
	Decl    *ast.FuncDecl

	builtin *bool
}

func (es *ErrStack) NewFunc(callCtx *Ctx, call *ast.CallExpr, name, method string, pkg *types.Package) *Func {
	pkgPath := ""
	if pkg != nil {
		pkgPath = pkg.Path()
	}

	fn := &Func{
		Ctx:     es.NewCtx(pkgPath),
		CallCtx: callCtx,
		Name:    name,
		Method:  method,
		Pkg:     pkg,
		Call:    call,
		Decl:    nil,
	}
	fn.LoadDecl()

	return fn
}

var True = true
var False = false

func (fn *Func) IsBuiltin() bool {
	if fn == nil {
		return false
	}
	if fn.builtin != nil {
		return *fn.builtin
	}

	if fn.Pkg == nil {
		fn.builtin = &True
		return true
	}
	path := strings.SplitN(fn.Pkg.Path(), "/", 1)[0]
	if path == "_" {
		fn.builtin = &False
		return false
	}
	if path == "command-line-arguments" {
		fn.builtin = &False
		return false
	}
	if path == "golang.org" {
		fn.builtin = &True
		return true
	}
	if !strings.Contains(path, ".") {
		fn.builtin = &True
		return true
	}

	fn.builtin = &False
	return false
}

// LoadDecl assigns *ast.FuncDecl for function type.
func (fn *Func) LoadDecl() {
	if fn == nil {
		return
	}
	if fn.IsBuiltin() {
		return
	}

	for _, file := range fn.Ctx.Syntax {
		for _, decl := range file.Decls {
			switch decl := decl.(type) {
			case *ast.FuncDecl:
				t := fn.Ctx.Info.ObjectOf(decl.Name).(*types.Func).Type().(*types.Signature)
				recv := t.Recv()
				r := ""
				if recv != nil {
					r = recv.Type().String()
				}
				if fn.Method != "" && !strings.HasSuffix(r, fn.Method) {
					continue
				}
				if decl.Name.Name == fn.Name {
					fn.Decl = decl
					return
				}
			}
		}
	}

	return
}

type FunctionInfo map[string][]string

func (f FunctionInfo) Contains(fn *Func) bool {
	if fn == nil {
		return false
	}
	if fn.Pkg == nil {
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

// getFunc returns imported package of function.
func (es *ErrStack) getFunc(ctx *Ctx, call *ast.CallExpr) *Func {
	if ctx == nil {
		log.Fatalln("getFunc context is nil")
	}

	callCtx := ctx
	switch se := call.Fun.(type) {
	case *ast.SelectorExpr:
		seSelObj := ctx.Info.ObjectOf(se.Sel)
		if seSelObj != nil && seSelObj.Pkg() != nil {
			return es.NewFunc(callCtx, call, se.Sel.Name, "", seSelObj.Pkg())
		}

		x := se.X
	_for:
		for {
			switch xt := x.(type) {
			case *ast.SelectorExpr:
				x = xt.Sel
			case *ast.CallExpr:
				fn := es.getFunc(ctx, xt)
				if fn == nil || fn.IsBuiltin() {
					return nil
				}
				if fn.Decl == nil {
					return nil
				}
				if fn.Decl.Type == nil {
					return nil
				}
				if fn.Decl.Type.Results == nil {
					return nil
				}
				if fn.Decl.Type.Results.List == nil {
					return nil
				}
				x = fn.Decl.Type.Results.List[0].Type
				ctx = fn.Ctx
			case *ast.StarExpr:
				x = xt.X
			default:
				break _for
			}
		}

		if x == nil {
			return nil
		}

		if id, isIdent := x.(*ast.Ident); isIdent {
			if selObj := ctx.Info.ObjectOf(id); selObj != nil {
				switch selObjTyped := selObj.(type) {
				case *types.PkgName:
					return es.NewFunc(callCtx, call, se.Sel.Name, "", selObjTyped.Imported())
				default:
				}
			} else {
			}
		} else {
		}
		if t, ok := ctx.Info.TypeOf(x).(*types.Named); ok && t != nil {
			return es.NewFunc(callCtx, call, se.Sel.Name, t.String(), t.Obj().Pkg())
		}
		return nil
	case *ast.Ident:
		obj := ctx.Info.ObjectOf(se)
		if obj == nil {
			return nil
		}
		return es.NewFunc(callCtx, call, obj.Name(), "", obj.Pkg())
	}

	return nil
}
