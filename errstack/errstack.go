package errstack

import (
	"go/ast"
	"go/token"
	"go/types"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
	"golang.org/x/tools/go/packages"
	"strings"
)

var (
	DefaultWrappedFunctions = map[string][]string{
		"github.com/pkg/errors": {"New", "Errorf", "Wrap", "Wrapf", "WithStack"},
	}
	DefaultCleanFunctions = map[string][]string{
		"fmt":                   {"Errorf"},
		"errors":                {"New"},
		"github.com/pkg/errors": {"WithMessage", "WithMessagef"},
	}
	DefaultThreshold     = .25
	DefaultMaxStackDepth = 25
)

type Settings struct {
	// WrappedFunctions - list of functions that are considered to wrap errors.
	// If you're using some fancy error wrapping library like github.com/pkg/errors,
	// you may want to add it to this list.
	// If you want to ignore some functions, simply don't add them to the list.
	WrappedFunctions FunctionInfo `mapstructure:"wrappedFunctions" yaml:"wrappedFunctions"`
	// CleanFunctions - list of functions that are considered to clean errors without stacktrace.
	CleanFunctions FunctionInfo `mapstructure:"cleanFunctions" yaml:"cleanFunctions"`
	// Threshold in percentage for the number of branches returning wrapped errors to be considered a violation.
	// Default value is 25%.
	// That means that if there are 3 sources of error that are non-wrapped and one that is wrapped, ErrStack will report an error.
	// On the other hand, if there are 4 wrapped sources and only one non-wrapped source, ErrStack will not report an error.
	Threshold float64 `mapstructure:"threshold" yaml:"threshold"`
	// MaxStackDepth - how many stack frames to check for before giving up.
	// May impact performance on large codebases and high value.
	// Default value is 25.
	MaxStackDepth int `mapstructure:"maxStackDepth" yaml:"maxStackDepth"`
}

func NewDefaultConfig() Settings {
	return Settings{
		WrappedFunctions: DefaultWrappedFunctions,
		Threshold:        DefaultThreshold,
		MaxStackDepth:    DefaultMaxStackDepth,
	}
}

type ErrStack struct {
	settings Settings
	pkgs     map[string]*packages.Package
}

func NewErrStack(settings Settings) *ErrStack {
	return &ErrStack{
		settings: settings,
		pkgs:     make(map[string]*packages.Package),
	}
}

func (es *ErrStack) getPackage(pkgPath string) *packages.Package {
	if strings.HasPrefix(pkgPath, "_/") {
		pkgPath = strings.TrimPrefix(pkgPath, "_")
	}
	pkg, ok := es.pkgs[pkgPath]
	if ok {
		return pkg
	}

	pkgs, err := packages.Load(&packages.Config{
		Mode: packages.NeedSyntax | packages.NeedTypes | packages.NeedTypesInfo | packages.NeedImports | packages.NeedDeps,
	}, pkgPath)
	if err != nil || len(pkgs) != 1 {
		return nil
	}
	es.pkgs[pkgPath] = pkgs[0]
	return pkgs[0]
}

func (es *ErrStack) Run(pass *analysis.Pass) (any, error) {
	// pass.ResultOf[inspect.Analyzer] will be set if we've added inspect.Analyzer to Requires.
	inspecting := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	nodeFilter := []ast.Node{ // filter needed nodes: visit only them
		(*ast.CallExpr)(nil),
	}

	inspecting.Preorder(nodeFilter, func(node ast.Node) {
		call := node.(*ast.CallExpr) // Get call expression
		se, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return
		}
		ctx := &Ctx{Info: pass.TypesInfo, Fset: pass.Fset}
		// No arguments, no need to check
		if len(call.Args) < 1 {
			return
		}
		// Not a wrapped function, no need to check
		if !es.settings.WrappedFunctions.Contains(ctx, call) {
			return
		}
		arg := call.Args[0]

		// If it's an error, we look for the source of the error and check if it's a wrapped error.
		wrapped, total := es.isWrapped(ctx, arg, 0, es.settings.MaxStackDepth, false)
		if float64(wrapped)/float64(total) >= es.settings.Threshold {
			pass.Reportf(call.Pos(), "%s.%s call unnecessarily wraps error with stacktrace. Replace with errors.WithMessage() or fmt.Errorf()", se.X.(*ast.Ident).Name, se.Sel.Name)
			return
		}
	})

	return nil, nil
}

// isWrapped recursively checks if arg was wrapped before.
// If arg is a variable, finds it's source and checks if it was wrapped before.
// If arg is a function call, finds it's source and checks all return statements for wrapped errors.
func (es *ErrStack) isWrapped(ctx *Ctx, arg ast.Expr, varPos int, maxStackDepth int, skipVar bool) (wrapped int, total int) {
	if maxStackDepth <= 0 {
		return
	}
	maxStackDepth = maxStackDepth - 1
	switch typedArg := arg.(type) {
	case *ast.Ident:
		// It's a variable, find the source and check if it was wrapped before.
		obj := ctx.Info.ObjectOf(typedArg)
		if obj == nil {
			return
		}
		switch typedObj := obj.(type) {
		case *types.Var:
			if skipVar {
				return 0, 1
			}
			parentCtx, parent := es.findNodeAtPosition(ctx, typedObj.Pkg().Path(), typedObj.Parent().Pos())
			assignments := es.findAssignments(parentCtx, parent, ctx, typedArg)
			for _, assignment := range assignments {
				wrapped2, _ := es.isWrapped(parentCtx, assignment.Stmt.Rhs[assignment.RPos], assignment.LPos, maxStackDepth, true)
				wrapped += wrapped2
			}
			total += 1
			return
		default:
			return
		}
	case *ast.CallExpr:
		// It's a function call, find the source and check all return statements for wrapped errors.
		if typedArg.Fun == nil {
			return
		}
		if es.settings.WrappedFunctions.Contains(ctx, typedArg) {
			return 1, 1
		}
		if es.settings.CleanFunctions.Contains(ctx, typedArg) {
			return 0, 1
		}

		funcCtx, funcDecl := es.findCallFuncDecl(ctx, typedArg)
		if funcDecl == nil {
			return
		}
		for _, stmt := range funcDecl.Body.List {
			if returnStmt, ok := stmt.(*ast.ReturnStmt); ok {
				wrapped2, total2 := es.isWrapped(funcCtx, returnStmt.Results[varPos], 0, maxStackDepth, false)
				wrapped += wrapped2
				total += total2
			}
		}
		return
	default:
		return
	}
}

// findNodeAtPosition finds the node at the given position. Useful for finding the parent of a variable.
func (es *ErrStack) findNodeAtPosition(ctx *Ctx, pkgPath string, position token.Pos) (*Ctx, ast.Node) {
	pos := ctx.Fset.Position(position)
	pkg := es.getPackage(pkgPath)
	var found ast.Node

	for _, file := range pkg.Syntax {
		if pkg.Fset.Position(file.Pos()).Filename != pos.Filename {
			continue
		}
		ast.Inspect(file, func(node ast.Node) bool {
			if node == nil {
				return true
			}
			if pkg.Fset.Position(node.Pos()) == pos {
				found = node
				return false
			}

			return true
		})
	}

	return &Ctx{Info: pkg.TypesInfo, Fset: pkg.Fset}, found
}

type Assignment struct {
	Stmt *ast.AssignStmt
	LPos int
	RPos int
}

func (es *ErrStack) findAssignments(ctx *Ctx, parent ast.Node, objCtx *Ctx, obj *ast.Ident) []Assignment {
	var assignments []Assignment
	name := obj.Name
	objPos := objCtx.Fset.Position(obj.NamePos)
	ast.Inspect(parent, func(node ast.Node) bool {
		if node == nil {
			return true
		}
		if assign, ok := node.(*ast.AssignStmt); ok {
			assignPos := ctx.Fset.Position(assign.Pos())
			if assignPos.Line > objPos.Line || (assignPos.Line == objPos.Line && assignPos.Column < objPos.Column) {
				return true
			}
			for i, expr := range assign.Lhs {
				if ident, ok := expr.(*ast.Ident); ok && ident.Name == name {
					pos := 0
					if len(assign.Lhs) <= len(assign.Rhs) {
						pos = i
					}
					assignments = append(assignments, Assignment{Stmt: assign, LPos: i, RPos: pos})
				}
			}
		}

		return true
	})

	return assignments
}
