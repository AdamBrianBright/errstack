package errstack

import (
	"go/ast"
	"go/token"

	"github.com/AdamBrianBright/errstack/internal/config"
	"github.com/AdamBrianBright/errstack/internal/log"
	"github.com/AdamBrianBright/errstack/internal/model"
	"github.com/AdamBrianBright/errstack/internal/passes/preload_packages"

	"golang.org/x/tools/go/analysis/passes/ctrlflow"
	"golang.org/x/tools/go/cfg"
)

type Result struct {
	OriginalFunctions   []*model.Function
	FunctionsWithErrors map[token.Position]*model.Function
	conf                *config.Config
	loader              *preload_packages.Result
}

// TryAddCallExpr tries to parse an AST node as a function call and add its decl to the list of functions with errors.
// Returns the position of the function declaration if it was added successfully, nil otherwise.
func (res *Result) TryAddCallExpr(info *model.Info, cfgs *ctrlflow.CFGs, call ast.Node) *model.Function {
	if call == nil {
		log.Log("Trying to add nil call\n")
	}
	switch fun := call.(type) {
	case *ast.CallExpr:
		if fun.Fun == nil {
			return nil
		}
		return res.TryAddCallExpr(info, cfgs, fun.Fun)
	case *ast.Ident:
		if fun == nil {
			return nil
		}
		if fun.Obj != nil && fun.Obj.Decl != nil {
			return res.TryAddFunction(info, cfgs, fun.Obj.Decl)
		}
		return nil
	case *ast.SelectorExpr:
		if fun.Sel == nil {
			return nil
		}

		// Handle external package function calls
		obj := info.Types.ObjectOf(fun.Sel)
		if obj != nil {
			// Get the package path from the object
			pkg := obj.Pkg()
			if pkg != nil {
				pkgPath := pkg.Path()
				funcName := fun.Sel.Name

				// Check if this is a known wrapper or clean function
				if res.conf.WrapperFunctions.Match(pkgPath, funcName) || res.conf.CleanFunctions.Match(pkgPath, funcName) {
					// Create a virtual function entry for external package functions
					pos := info.Fset.Position(fun.Sel.Pos())
					if v, ok := res.FunctionsWithErrors[pos]; ok {
						return v
					}

					fn := &model.Function{
						Name:       funcName,
						Node:       fun,
						Type:       nil,
						Body:       nil,
						Block:      nil,
						Pos:        pos,
						IsWrapping: res.conf.WrapperFunctions.Match(pkgPath, funcName),
						CalledBy:   model.Stack[*model.Function]{},
						Pkg:        pkgPath,
						Info:       info,
					}
					res.FunctionsWithErrors[pos] = fn
					return fn
				}
			}
		}

		// Try to load the actual function declaration if it's not a known wrapper/clean function
		if fun.Sel.Obj != nil && fun.Sel.Obj.Decl != nil {
			return res.TryAddFunction(info, cfgs, fun.Sel.Obj.Decl)
		}
		var decl ast.Node
		if obj != nil {
			info, decl = res.loader.LoadObject(info, obj)
		} else {
			info, decl = res.loader.LoadSelector(info, info.FormatNode(fun.X), fun.Sel.String())
		}
		if decl == nil {
			return nil
		}
		return res.TryAddFunction(info, cfgs, decl)
	case *ast.StarExpr:
		if fun.X == nil {
			return nil
		}
		return res.TryAddCallExpr(info, cfgs, fun.X)
	case *ast.ParenExpr:
		if fun.X == nil {
			return nil
		}
		return res.TryAddCallExpr(info, cfgs, fun.X)
	case *ast.IndexExpr:
		if fun.X == nil || fun.Index == nil {
			return nil
		}
		return res.TryAddCallExpr(info, cfgs, fun.X)
	}
	return nil
}

// TryAddFunction tries to parse an AST node as a function and add it to the list of functions with errors.
// Returns the position of the function declaration if it was added successfully, nil otherwise.
// If a function is already in the list, it returns the existing position.
func (res *Result) TryAddFunction(info *model.Info, cfgs *ctrlflow.CFGs, fun any) *model.Function {
	switch decl := fun.(type) {
	case *ast.FuncDecl:
		if decl.Type.Results == nil {
			return nil
		}
		pos := info.Fset.Position(decl.Pos())
		if v, ok := res.FunctionsWithErrors[pos]; ok {
			return v
		}

		var foundError bool
		for _, f := range decl.Type.Results.List {
			if fieldType := info.Types.TypeOf(f.Type); isErrorType(fieldType) {
				foundError = true
				break
			}
		}
		if !foundError {
			return nil
		}
		fn := &model.Function{
			Name:       decl.Name.Name,
			Node:       decl,
			Type:       decl.Type,
			Body:       decl.Body,
			Block:      getCFGBlock(cfgs, decl),
			Pos:        pos,
			IsWrapping: false,
			CalledBy:   model.Stack[*model.Function]{},
			Pkg:        res.conf.GetPkgPath(info.Fset.Position(decl.Pos()).Filename),
			Info:       info,
		}
		res.FunctionsWithErrors[pos] = fn
		return fn
	case *ast.FuncLit:
		if decl.Type.Results == nil {
			return nil
		}
		pos := info.Fset.Position(decl.Pos())
		if v, ok := res.FunctionsWithErrors[pos]; ok {
			return v
		}

		var foundError bool
		for _, f := range decl.Type.Results.List {
			if fieldType := info.Types.TypeOf(f.Type); isErrorType(fieldType) {
				foundError = true
				break
			}
		}
		if !foundError {
			return nil
		}
		fn := &model.Function{
			Name:       "anonymous",
			Node:       decl,
			Type:       decl.Type,
			Body:       decl.Body,
			Block:      getCFGBlock(cfgs, decl),
			Pos:        pos,
			IsWrapping: false,
			CalledBy:   model.Stack[*model.Function]{},
			Pkg:        res.conf.GetPkgPath(info.Fset.Position(decl.Pos()).Filename),
			Info:       info,
		}
		res.FunctionsWithErrors[pos] = fn
		return fn
	}

	return nil
}

// getCFGBlock returns the first block of the CFG for the given node.
func getCFGBlock(cfgs *ctrlflow.CFGs, node ast.Node) *cfg.Block {
	defer func() {
		_ = recover() // Ignore any panics
	}()
	switch n := node.(type) {
	case *ast.FuncDecl:
		return cfgs.FuncDecl(n).Blocks[0]
	case *ast.FuncLit:
		return cfgs.FuncLit(n).Blocks[0]
	default:
		return nil
	}
}
