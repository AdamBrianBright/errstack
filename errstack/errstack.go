package errstack

import (
	"go/ast"
	"go/token"
	"go/types"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
	"golang.org/x/tools/go/packages"
	"path/filepath"
	"slices"
	"sync"
)

type PkgFuncs struct {
	Pkg   string   `mapstructure:"pkg"   yaml:"pkg"`
	Funcs []string `mapstructure:"funcs" yaml:"funcs"`
}

var (
	DefaultWrapperFunctions = []PkgFuncs{
		{Pkg: "github.com/pkg/errors", Funcs: []string{"New", "Errorf", "Wrap", "Wrapf", "WithStack"}},
	}
	DefaultCleanFunctions = []PkgFuncs{
		{Pkg: "github.com/pkg/errors", Funcs: []string{"WithMessage", "WithMessagef"}},
	}
	DefaultThreshold     = .5
	DefaultMaxStackDepth = 5
)

type Config struct {
	// WrapperFunctions - list of functions that are considered to wrap errors.
	// If you're using some fancy error wrapping library like github.com/pkg/errors,
	// you may want to add it to this list.
	// If you want to ignore some functions, simply don't add them to the list.
	WrapperFunctions []PkgFuncs `mapstructure:"wrapperFunctions" yaml:"wrapperFunctions"`
	// CleanFunctions - list of functions that are considered to clean errors without stacktrace.
	CleanFunctions []PkgFuncs `mapstructure:"cleanFunctions" yaml:"cleanFunctions"`
	// Threshold in percentage for the number of branches returning wrapped errors to be considered a violation.
	// Default value is 50%. Max is 100%.
	// That means that if there are 3 sources of error that are non-wrapped and one that is wrapped, ErrStack will report an error.
	// On the other hand, if there are 4 wrapped sources and only one non-wrapped source, ErrStack will not report an error.
	Threshold float64 `mapstructure:"threshold" yaml:"threshold"`
	// MaxStackDepth - how many stack frames to check for before giving up.
	// May impact performance on large codebases and high value.
	// Default value is 5. Max is 50.
	MaxStackDepth int `mapstructure:"maxStackDepth" yaml:"maxStackDepth"`
}

func NewDefaultConfig() Config {
	return Config{
		WrapperFunctions: DefaultWrapperFunctions,
		CleanFunctions:   DefaultCleanFunctions,
		Threshold:        DefaultThreshold,
		MaxStackDepth:    DefaultMaxStackDepth,
	}
}

type Settings struct {
	WrappedFunctions FunctionInfo
	CleanFunctions   FunctionInfo
	Threshold        float64
	MaxStackDepth    int
}

type ErrStack struct {
	m sync.RWMutex

	settings             Settings
	wrappedFunctionNames []string
	pkgs                 map[string]*packages.Package
	pass                 *analysis.Pass
	dir                  string
	isWrappedCache       sync.Map
	findNodeCache        sync.Map
}

func NewErrStack(config Config) *ErrStack {
	settings := Settings{
		WrappedFunctions: FunctionInfo{},
		CleanFunctions:   FunctionInfo{},
		Threshold:        config.Threshold,
		MaxStackDepth:    config.MaxStackDepth,
	}
	for _, fs := range config.WrapperFunctions {
		settings.WrappedFunctions[fs.Pkg] = fs.Funcs
	}
	for _, fs := range config.CleanFunctions {
		settings.CleanFunctions[fs.Pkg] = fs.Funcs
	}
	if settings.Threshold > 1 {
		settings.Threshold = 1
	}
	if settings.MaxStackDepth > 50 {
		settings.MaxStackDepth = 25
	} else if settings.MaxStackDepth < 1 {
		settings.MaxStackDepth = 1
	}

	wrappedFunctionNamesMap := make(map[string]struct{})
	for _, fs := range settings.WrappedFunctions {
		for _, f := range fs {
			wrappedFunctionNamesMap[f] = struct{}{}
		}
	}
	wrappedFunctionNames := make([]string, 0, len(wrappedFunctionNamesMap))
	for f := range wrappedFunctionNamesMap {
		wrappedFunctionNames = append(wrappedFunctionNames, f)
	}

	return &ErrStack{
		settings:             settings,
		wrappedFunctionNames: wrappedFunctionNames,
		pkgs:                 make(map[string]*packages.Package),
	}
}

func (es *ErrStack) Run(pass *analysis.Pass) (any, error) {
	es.m.Lock()
	es.pass = pass
	es.dir = filepath.Dir(pass.Fset.Position(pass.Files[0].Pos()).Filename)
	es.m.Unlock()
	// pass.ResultOf[inspect.Analyzer] will be set if we've added inspect.Analyzer to Requires.
	inspecting := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	ctx := es.NewCtx("")

	inspecting.Preorder([]ast.Node{(*ast.CallExpr)(nil)}, func(node ast.Node) {
		call := node.(*ast.CallExpr) // Get call expression
		// No arguments, no need to check
		if len(call.Args) < 1 {
			return
		}
		if se, ok := call.Fun.(*ast.SelectorExpr); ok {
			if !slices.Contains(es.wrappedFunctionNames, se.Sel.Name) {
				return
			}
		} else if se, ok := call.Fun.(*ast.Ident); ok {
			if !slices.Contains(es.wrappedFunctionNames, pass.TypesInfo.ObjectOf(se).Name()) {
				return
			}
		} else {
			return
		}

		fn := es.getFunc(ctx, call)
		if fn == nil {
			return
		}
		// Not a wrapped function, no need to check
		if fn.IsBuiltin() || !es.settings.WrappedFunctions.Contains(fn) {
			return
		}

		// If it's an error, we look for the source of the error and check if it's a wrapped error.
		wrapped, total := es.isWrapped(ctx, call.Args[0], 0, es.settings.MaxStackDepth, false)
		if total > 0 && float64(wrapped)/float64(total) >= es.settings.Threshold {
			pass.Reportf(call.Pos(), "%s call unnecessarily wraps error with stacktrace. Replace with errors.WithMessage() or fmt.Errorf()", fn.Name)
			return
		}
	})

	return nil, nil
}

type _wrapCache struct {
	wrapped int
	total   int
}

// isWrapped recursively checks if arg was wrapped before.
// If arg is a variable, finds it's source and checks if it was wrapped before.
// If arg is a function call, finds it's source and checks all return statements for wrapped errors.
func (es *ErrStack) isWrapped(ctx *Ctx, arg ast.Expr, varPos int, maxStackDepth int, skipVar bool) (wrapped int, total int) {
	if maxStackDepth <= 0 {
		return
	}
	if _cached, ok := es.isWrappedCache.Load(arg); ok {
		_val := _cached.(_wrapCache)
		return _val.wrapped, _val.total
	}
	defer func() {
		es.isWrappedCache.Store(arg, _wrapCache{wrapped: wrapped, total: total})
	}()

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
				wrapped, total = 0, 1
				return
			}
			parentCtx, parent := es.findNodeAtPosition(ctx, typedObj.Pkg().Path(), typedObj.Parent().Pos())
			if parent == nil {
				// Parent not found, skip. This can happen if statement is outside of function or method.
				return
			}
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
		fn := es.getFunc(ctx, typedArg)
		if fn == nil || fn.Decl == nil {
			return
		}
		if !fn.IsBuiltin() && es.settings.WrappedFunctions.Contains(fn) {
			wrapped, total = 1, 1
			return
		} else if es.settings.CleanFunctions.Contains(fn) {
			wrapped, total = 0, 1
			return
		}
		if varPos >= len(fn.Decl.Type.Results.List) {
			return
		}
		var namedArg *ast.Ident
		fieldNames := fn.Decl.Type.Results.List[varPos].Names
		if len(fieldNames) > 0 {
			namedArg = fieldNames[0]
		}
		for _, stmt := range fn.Decl.Body.List {
			if returnStmt, ok := stmt.(*ast.ReturnStmt); ok {
				idx := 0
				vpos := 0
				if len(returnStmt.Results) == 0 {
					assignments := es.findAssignments(fn.Ctx, fn.Decl, fn.Ctx, namedArg)
					for _, assignment := range assignments {
						wrapped2, _ := es.isWrapped(fn.Ctx, assignment.Stmt.Rhs[assignment.RPos], assignment.LPos, maxStackDepth, true)
						wrapped += wrapped2
					}
					total += 1
					continue
				}
				if len(returnStmt.Results) > 1 {
					idx = varPos
					vpos = idx
				}
				wrapped2, total2 := es.isWrapped(fn.Ctx, returnStmt.Results[idx], vpos, maxStackDepth, false)
				wrapped += wrapped2
				total += total2
			}
		}
		return
	default:
		return
	}
}

type _findNodeCache struct {
	ctx  *Ctx
	node ast.Node
}

// findNodeAtPosition finds the node at the given position. Useful for finding the parent of a variable.
func (es *ErrStack) findNodeAtPosition(ctx *Ctx, pkgPath string, position token.Pos) (*Ctx, ast.Node) {
	if _cached, ok := es.findNodeCache.Load(position); ok {
		_val := _cached.(_findNodeCache)
		return _val.ctx, _val.node
	}

	nodeCtx := es.NewCtx(pkgPath)
	pos := ctx.Fset.Position(position)
	var found ast.Node

	for _, file := range nodeCtx.Syntax {
		if nodeCtx.Fset.Position(file.Pos()).Filename != pos.Filename {
			continue
		}
		ast.Inspect(file, func(node ast.Node) bool {
			if node == nil {
				return true
			}
			if nodeCtx.Fset.Position(node.Pos()) == pos {
				found = node
				return false
			}

			return true
		})
		if found != nil {
			break
		}
	}

	es.findNodeCache.Store(position, _findNodeCache{ctx: nodeCtx, node: found})

	return nodeCtx, found
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
