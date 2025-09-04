package errstack

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"reflect"

	"github.com/AdamBrianBright/errstack/internal/config"
	"github.com/AdamBrianBright/errstack/internal/helpers"
	"github.com/AdamBrianBright/errstack/internal/log"
	"github.com/AdamBrianBright/errstack/internal/model"
	"github.com/AdamBrianBright/errstack/internal/passes/preload_packages"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/ctrlflow"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
	"golang.org/x/tools/go/cfg"
)

const _doc = `ErrStack finds unnecessary error wrapping with stacktraces.

The analyzer identifies cases where errors are wrapped multiple times with 
stacktrace information (e.g., using errors.Wrap, errors.WithStack), which 
can lead to duplicated stack traces in logs.

Algorithm:
1. Identifies all functions that return errors
2. Marks functions that return wrapped errors (with stack traces)
3. Traces error flow through the call graph
4. Reports unnecessary wrapping where errors are already wrapped

Configuration:
- wrapperFunctions: Functions that add stacktraces (e.g., errors.Wrap)
- cleanFunctions: Functions that don't add stacktraces (e.g., errors.New)

The analyzer supports suggested fixes to replace unnecessary wrapping 
with simpler alternatives like errors.WithMessage.
`

var Analyzer = &analysis.Analyzer{
	Name:       "errstack",
	Doc:        _doc,
	Run:        helpers.WrapRun(run),
	ResultType: reflect.TypeOf((*helpers.Result[*Result])(nil)),
	Requires:   []*analysis.Analyzer{inspect.Analyzer, ctrlflow.Analyzer, config.Analyzer, preload_packages.Analyzer},
}

func run(pass *analysis.Pass) (*Result, error) {
	log.Log("Run\n")
	loader, _ := helpers.GetResult[*preload_packages.Result](pass, preload_packages.Analyzer)
	conf, _ := helpers.GetResult[*config.Config](pass, config.Analyzer)
	defer log.Sync()

	var result = &Result{
		OriginalFunctions:   []*model.Function{},
		FunctionsWithErrors: map[token.Position]*model.Function{},
		conf:                conf,
		loader:              loader,
	}

	log.Log("FindFunctionsWithErrors\n")
	result.FindFunctionsWithErrors(pass)
	log.Log("MarkTaintedFunctions\n")
	result.MarkTaintedFunctions()
	log.Log("AnalyzeOriginalFunctions\n")
	result.AnalyzeOriginalFunctions(pass)

	for _, fn := range result.FunctionsWithErrors {
		log.Log("Found function %s(%t): %s\n", fn.Name, fn.IsWrapping, fn.Pos.String())
		for _, callee := range fn.CalledBy {
			log.Log("Function is called by %s(%t): %s\n", callee.Name, callee.IsWrapping, callee.Pos.String())
		}
		log.Log("\n")
	}

	return result, nil
}

// FindFunctionsWithErrors finds all functions that return errors in the AST.
func (res *Result) FindFunctionsWithErrors(pass *analysis.Pass) {
	inspectResult := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	cfgs := pass.ResultOf[ctrlflow.Analyzer].(*ctrlflow.CFGs)
	info := model.NewInfo(pass)

	nodeFilter := []ast.Node{
		(*ast.FuncDecl)(nil),
		(*ast.FuncLit)(nil),
	}

	inspectResult.Preorder(nodeFilter, func(n ast.Node) {
		if n == nil {
			return
		}
		var fn *model.Function
		switch f := n.(type) {
		case *ast.FuncDecl:
			fn = res.TryAddFunction(info, cfgs, f)
		case *ast.FuncLit:
			fn = res.TryAddFunction(info, cfgs, f)
		}
		if fn != nil {
			res.OriginalFunctions = append(res.OriginalFunctions, fn)
		}
	})

	visited := make(map[*model.Function]bool)
	stack := make(model.Stack[*FunctionWithDepth], 0, 64)

	// Initialize stack with original functions at depth 0
	for _, fn := range res.OriginalFunctions {
		stack.Push(&FunctionWithDepth{Function: fn, Depth: 0})
	}

	for item := stack.Pop(); item != nil; item = stack.Pop() {
		fnWithDepth := *item
		function := fnWithDepth.Function
		currentDepth := fnWithDepth.Depth

		log.Log("Populating %s (depth %d): %s\n", function.Name, currentDepth, function.Pos.String())
		if visited[function] {
			continue
		}
		visited[function] = true

		// Check MaxDepth limit (ignore if MaxDepth <= 0)
		if res.conf.MaxDepth > 0 && currentDepth >= res.conf.MaxDepth {
			log.Log("Reached max depth %d for function %s, stopping traversal\n", res.conf.MaxDepth, function.Name)
			continue
		}

		ast.Inspect(function.Node, func(n ast.Node) bool {
			if n == nil {
				return false
			}
			_, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			fn := res.TryAddCallExpr(function.Info, cfgs, n)
			if fn != nil {
				fn.CalledBy.AddUnique(function)
				// Push with incremented depth
				stack.Push(&FunctionWithDepth{Function: fn, Depth: currentDepth + 1})
			}
			return true
		})
	}
}

// MarkTaintedFunctions marks functions that return wrapped errors.
func (res *Result) MarkTaintedFunctions() {
	matchClean := res.conf.CleanFunctions.Match
	matchWrapper := res.conf.WrapperFunctions.Match

	for _, function := range res.FunctionsWithErrors {
		if matchClean(function.Pkg, function.Name) {
			log.Log("Function %s.%s is clean, marking with '%t': %s\n", function.Pkg, function.Name, false, function.Pos.String())
			function.IsWrapping = false
			continue
		}
		if matchWrapper(function.Pkg, function.Name) {
			log.Log("Function %s.%s is taint, marking with '%t': %s\n", function.Pkg, function.Name, true, function.Pos.String())
			function.IsWrapping = true
			continue
		}
	}
	var visited = make(map[*model.Function]bool)
	for _, function := range res.FunctionsWithErrors {
		res.propagateWrapping(visited, function)
	}
}

// propagateWrapping propagates wrapping information from the given function to all its callers.
func (res *Result) propagateWrapping(visited map[*model.Function]bool, function *model.Function) {
	log.Log("Propagating function %s.%s: %s\n", function.Pkg, function.Name, function.Pos.String())
	if !function.IsWrapping || res.conf.CleanFunctions.Match(function.Pkg, function.Name) {
		log.Log("Function %s.%s is not wrapping, skipping function\n", function.Pkg, function.Name)
		return
	}

	// Use depth-aware traversal
	stack := make(model.Stack[*FunctionWithDepth], 0, len(function.CalledBy))
	for _, caller := range function.CalledBy {
		stack.Push(&FunctionWithDepth{Function: caller, Depth: 0})
	}

	for v := stack.Pop(); v != nil; v = stack.Pop() {
		fnWithDepth := *v
		fn := fnWithDepth.Function
		currentDepth := fnWithDepth.Depth

		if visited[fn] {
			continue
		}
		visited[fn] = true
		if res.conf.CleanFunctions.Match(fn.Pkg, fn.Name) {
			continue
		}

		// Check MaxDepth limit (ignore if MaxDepth <= 0)
		if res.conf.MaxDepth > 0 && currentDepth >= res.conf.MaxDepth {
			log.Log("Reached max depth %d for propagation to function %s.%s, stopping\n", res.conf.MaxDepth, fn.Pkg, fn.Name)
			continue
		}

		log.Log("Taint function %s.%s (depth %d): %s\n", fn.Pkg, fn.Name, currentDepth, fn.Pos.String())
		fn.IsWrapping = true

		// Push callers with incremented depth
		for _, caller := range fn.CalledBy {
			stack.Push(&FunctionWithDepth{Function: caller, Depth: currentDepth + 1})
		}
	}
}

// AnalyzeOriginalFunctions walks over originally found functions CFG and reports if unnecessary wrapping is used.
func (res *Result) AnalyzeOriginalFunctions(pass *analysis.Pass) {
	cfgs := pass.ResultOf[ctrlflow.Analyzer].(*ctrlflow.CFGs)

	var visited = make(map[*cfg.Block]bool)
	var variables = make(map[token.Position]bool)
	for _, v := range res.OriginalFunctions {
		if !v.IsWrapping {
			continue
		}
		clear(visited)
		clear(variables)
		res.analyzeOriginalFunctionBlock(pass, cfgs, v.Block, visited, variables, 0)
	}
}

type StackCall struct {
	Fn   *model.Function
	Call *ast.CallExpr
}

type FunctionWithDepth struct {
	Function *model.Function
	Depth    int
}

// analyzeOriginalFunctionBlock walks over the CFG of the original function and
// traces all error variables and finds errors that are unnecessarily wrapped.
func (res *Result) analyzeOriginalFunctionBlock(
	pass *analysis.Pass,
	cfgs *ctrlflow.CFGs,
	block *cfg.Block,
	visited map[*cfg.Block]bool,
	variables map[token.Position]bool,
	depth int,
) {
	if block == nil || visited[block] {
		return
	}

	// Check MaxDepth limit (ignore if MaxDepth <= 0)
	if res.conf.MaxDepth > 0 && depth >= res.conf.MaxDepth {
		log.Log("Reached max depth %d in CFG traversal, stopping\n", res.conf.MaxDepth)
		return
	}
	info := model.NewInfo(pass)
	matchWrapping := res.conf.WrapperFunctions.Match
	replaceWith := res.conf.WrapperFunctions.ReplaceWith
	replaceWithFunction := res.conf.WrapperFunctions.ReplaceWithFunction

	visited[block] = true
	log.Log("Visiting block %v\n", block)

	for _, item := range block.Nodes {
		ast.Inspect(item, func(n ast.Node) bool {
			if n == nil {
				return false
			}
			log.Log("Visiting node %s\n", info.FormatNode(n))
			switch node := n.(type) {
			case *ast.CallExpr:
				fn := res.TryAddCallExpr(info, cfgs, node)
				if fn == nil || !matchWrapping(fn.Pkg, fn.Name) {
					return true
				}
				var wrapping bool
				for _, arg := range node.Args {
					result := res.analyzeCallStack(pass, cfgs, info, arg, variables)
					if result != nil {
						wrapping = wrapping || *result
					}
				}
				if wrapping {
					fn.IsWrapping = true
					log.Log("Node unnecessarily wraps error with stacktrace %s\n", info.FormatNode(node))
					errorArgument := res.getErrorArgument(cfgs, info, node)
					var fixes []analysis.SuggestedFix
					if errorArgument != nil {
						if len(node.Args) == 1 {
							fixes = []analysis.SuggestedFix{
								{
									Message: "Remove unnecessary error wrapping",
									TextEdits: []analysis.TextEdit{
										{
											Pos:     node.Pos(),
											End:     node.End(),
											NewText: []byte(info.FormatNode(errorArgument)),
										},
									},
								},
							}
						} else {
							message := "Replace unnecessary error wrapping"
							newText := info.FormatNode(node)
							if len(node.Args) == 2 {
								newText = replaceWith(fn.Pkg, fn.Name, newText)
							} else {
								newText = replaceWithFunction(fn.Pkg, fn.Name, newText)
							}
							if newText != "" {
								fixes = []analysis.SuggestedFix{
									{
										Message: message,
										TextEdits: []analysis.TextEdit{
											{
												Pos:     node.Pos(),
												End:     node.End(),
												NewText: []byte(newText),
											},
										},
									},
								}
							}
						}
					}
					pass.Report(analysis.Diagnostic{
						Pos:            node.Pos(),
						End:            node.End(),
						Category:       "",
						Message:        fmt.Sprintf("%s call unnecessarily wraps error with stacktrace. Replace with errors.WithMessage() or fmt.Errorf()", fn.Name),
						URL:            "",
						SuggestedFixes: fixes,
						Related:        nil,
					})
				}
				return true
			}
			return true
		})
		// Propagate wrapping information to new assignments
		ast.Inspect(item, func(node ast.Node) bool {
			if node == nil {
				return false
			}
			assignStmt, ok := node.(*ast.AssignStmt)
			if !ok {
				return true
			}
			lhs := make([]*token.Position, len(assignStmt.Lhs))
			found := false
			for i, expr := range assignStmt.Lhs {
				if id, idOk := expr.(*ast.Ident); idOk && id != nil {
					obj := info.Types.ObjectOf(id)
					if !isObjectError(obj) {
						continue
					}
					objPos := info.Fset.Position(obj.Pos())
					lhs[i] = &objPos
					found = true
				}
			}
			if !found {
				return true
			}
			log.Log("AssignStmt %s\n", info.FormatNode(assignStmt))

			if len(assignStmt.Rhs) == 1 {
				log.Log("AssignStmt Rhs[0] %s\n", info.FormatNode(assignStmt.Rhs[0]))
				callStackWrapping := res.analyzeCallStack(pass, cfgs, info, assignStmt.Rhs[0], variables)
				if callStackWrapping == nil {
					log.Log("AssignStmt Rhs[0] is nil\n")
					return true
				}
				log.Log("AssignStmt Rhs[0] is %t\n", *callStackWrapping)
				for _, lh := range lhs {
					if lh == nil {
						continue
					}
					log.Log("Updating %s as %t\n", lh.String(), *callStackWrapping)
					variables[*lh] = *callStackWrapping
				}
			} else {
				log.Log("AssignStmt Rhs %d\n", len(assignStmt.Rhs))
				for i, lh := range lhs {
					if lh == nil {
						continue
					}
					result := res.analyzeCallStack(pass, cfgs, info, assignStmt.Rhs[i], variables)
					if result != nil {
						log.Log("AssignStmt Rhs[%d] is %t\n", i, *result)
						log.Log("Updating %s as %t\n", lh.String(), *result)
						variables[*lh] = *result
					} else {
						log.Log("AssignStmt Rhs[%d] is nil\n", i)
					}
				}
			}

			return true
		})
	}
	for _, branch := range block.Succs {
		res.analyzeOriginalFunctionBlock(pass, cfgs, branch, visited, variables, depth+1)
	}
}

var trueValue = true
var falseValue = false

func (res *Result) analyzeCallStack(
	pass *analysis.Pass,
	cfgs *ctrlflow.CFGs,
	info *model.Info,
	n ast.Node,
	variables map[token.Position]bool,
) *bool {
	if n == nil {
		return nil
	}
	log.Log("Analyze call stack %s\n", info.FormatNode(n))
	switch node := n.(type) {
	case *ast.CallExpr:
		if node.Fun == nil {
			return nil
		}
		log.Log("CallExpr %s\n", info.FormatNode(node.Fun))
		fn := res.TryAddCallExpr(info, cfgs, node)
		if fn == nil {
			// If we can't find the function, assume it returns a clean error (not wrapped)
			// This handles standard library functions like json.Marshal properly
			return &falseValue
		}
		log.Log("CallExpr Function %s\n", fn.Name)
		if fn.IsWrapping {
			log.Log("CallExpr Function is wrapping\n")
			return &trueValue
		}
		for i, arg := range node.Args {
			log.Log("CallExpr Arg[%d] %s\n", i, info.FormatNode(arg))
			result := res.analyzeCallStack(pass, cfgs, info, arg, variables)
			if result != nil {
				return result
			}
		}
		return &falseValue
	case *ast.Ident:
		log.Log("Ident %s\n", info.FormatNode(node))
		if obj := info.Types.ObjectOf(node); obj != nil {
			log.Log("Ident Object error\n")
			if isObjectError(obj) {
				log.Log("Ident Object is error\n")
				if variables[info.Fset.Position(obj.Pos())] {
					log.Log("Ident Object is error and variables[%t]\n", variables[info.Fset.Position(obj.Pos())])
					return &trueValue
				} else {
					log.Log("Ident Object is error and variables[%t]\n", variables[info.Fset.Position(obj.Pos())])
					return &falseValue
				}
			}
		}
		return nil
	case *ast.StarExpr:
		log.Log("StarExpr %s\n", info.FormatNode(node))
		return res.analyzeCallStack(pass, cfgs, info, node.X, variables)
	case *ast.ParenExpr:
		log.Log("ParenExpr %s\n", info.FormatNode(node))
		return res.analyzeCallStack(pass, cfgs, info, node.X, variables)
	}
	return nil
}

func (res *Result) getErrorArgument(cfgs *ctrlflow.CFGs, info *model.Info, call *ast.CallExpr) ast.Expr {
	if len(call.Args) == 0 {
		return nil
	}
	for _, rootArg := range call.Args {
		untypedArg := rootArg
		for {
			switch arg := untypedArg.(type) {
			case *ast.Ident:
				obj := info.Types.ObjectOf(arg)
				if isObjectError(obj) {
					return rootArg
				}
			case *ast.CallExpr:
				fn := res.TryAddCallExpr(info, cfgs, arg)
				if fn != nil {
					return rootArg
				}
			case *ast.StarExpr:
				untypedArg = arg.X
			case *ast.ParenExpr:
				untypedArg = arg.X
			case *ast.SelectorExpr:
				untypedArg = arg.Sel
			case *ast.IndexExpr:
				untypedArg = arg.X
			default:
				break
			}
		}
	}

	return nil
}

func isErrorType(typ types.Type) bool {
	if typ == nil {
		return false
	}

	// Check if it's the built-in error interface
	if typ.String() == "error" {
		return true
	}

	// Check if it implements the error interface
	errorInterface := types.NewInterfaceType([]*types.Func{
		types.NewFunc(
			token.NoPos,
			nil,
			"Error",
			types.NewSignatureType(
				nil,
				nil,
				nil,
				types.NewTuple(),
				types.NewTuple(types.NewVar(token.NoPos, nil, "", types.Typ[types.String])),
				false,
			),
		),
	}, nil)

	return types.Implements(typ, errorInterface)
}

func isObjectError(obj types.Object) bool {
	if obj == nil {
		return false
	}
	return isErrorType(obj.Type())
}
