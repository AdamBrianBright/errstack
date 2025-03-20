package errstack

import (
	"go/ast"
	"go/token"
	"go/types"
	"os"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/ctrlflow"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
	"golang.org/x/tools/go/cfg"
	"golang.org/x/tools/go/packages"
)

type Settings struct {
	WrappedFunctions []PkgFunctions
	CleanFunctions   []PkgFunctions
	GoRoot           string
	WorkDir          string
}

type Function struct {
	Name       string                // Name of the function
	Node       ast.Node              // AST node of the function
	Block      *cfg.Block            // Control flow graph of the function
	Pos        token.Position        // Position of the function declaration
	IsWrapping bool                  // Is true if this function returns wrapped errors
	CalledBy   *List[token.Position] // Functions that call this function
	Pkg        string                // Package containing the function
	Pass       *Pass                 // Pass used to load the function
}

type NodeInfo struct {
	Pass *Pass
	Node ast.Node
}
type ErrStack struct {
	pass                *Locked[*analysis.Pass]
	settings            *Locked[Settings]
	originalFunctions   *List[token.Position]
	functionsWithErrors *Map[token.Position, *Function]
	loadedPkgs          *Map[string, *packages.Package]
	loadedObjects       *Map[token.Position, NodeInfo]
}

func NewErrStack(config Config) *ErrStack {
	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	goroot := os.Getenv("GOROOT")

	defer w.Sync()

	settings := Settings{
		WrappedFunctions: config.WrapperFunctions,
		CleanFunctions:   config.CleanFunctions,
		GoRoot:           goroot + "/src/",
		WorkDir:          wd + "/",
	}

	return &ErrStack{
		settings:            NewLocked(settings),
		pass:                NewLocked[*analysis.Pass](nil),
		originalFunctions:   NewList[token.Position](0, 64),
		functionsWithErrors: NewMap[token.Position, *Function](64),
		loadedPkgs:          NewMap[string, *packages.Package](64),
		loadedObjects:       NewMap[token.Position, NodeInfo](64),
	}
}

func (es *ErrStack) Run(analysisPass *analysis.Pass) (any, error) {
	es.pass.Set(analysisPass)
	pass := NewPass(analysisPass)

	es.loadPackages()

	log("findFunctionsWithErrors\n")
	es.findFunctionsWithErrors(pass)
	log("populateFunctionDependencies\n")
	es.populateFunctionDependencies(pass)
	log("markTaintedFunctions\n")
	es.markTaintedFunctions()
	log("analyzeOriginalFunctions\n")
	es.analyzeOriginalFunctions(pass)

	for _, fn := range es.functionsWithErrors.Clone() {
		log("Found function %s(%t): %s\n", fn.Name, fn.IsWrapping, fn.Pos.String())
		for _, position := range fn.CalledBy.Clone() {
			callee, _ := es.functionsWithErrors.Get(position)
			log("Function is called by %s(%t): %s\n", callee.Name, callee.IsWrapping, position.String())
		}
		log("\n")
	}

	return nil, nil
}

func (es *ErrStack) loadPackages() {
	settings := es.settings.Get()
	pkgs, err := packages.Load(&packages.Config{
		Mode: packages.LoadAllSyntax,
	}, settings.GoRoot+"/...", settings.WorkDir+"/...", settings.WorkDir+"vendor/...")
	if err != nil {
		panic(err)
	}
	for _, pkg := range pkgs {
		es.loadedPkgs.Set(es.getDirPkgPath(pkg.Dir), pkg)
	}
}

// findFunctionsWithErrors finds all functions that returns errors in the AST.
func (es *ErrStack) findFunctionsWithErrors(pass *Pass) {
	inspectResult := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	cfgs := pass.ResultOf[ctrlflow.Analyzer].(*ctrlflow.CFGs)

	nodeFilter := []ast.Node{
		(*ast.FuncDecl)(nil),
		(*ast.FuncLit)(nil),
	}

	inspectResult.Preorder(nodeFilter, func(n ast.Node) {
		if n == nil {
			return
		}
		switch f := n.(type) {
		case *ast.FuncDecl:
			es.tryAddFunction(pass, cfgs, f)
		case *ast.FuncLit:
			es.tryAddFunction(pass, cfgs, f)
		}
	})

	es.originalFunctions = NewList[token.Position](0, es.functionsWithErrors.Len())
	for pos := range es.functionsWithErrors.Clone() {
		es.originalFunctions.Push(pos)
	}
}

// populateFunctionDependencies walks through functions with errors, finds function calls and adds them to the list.
// Repeats the same process until no new functions are added.
func (es *ErrStack) populateFunctionDependencies(pass *Pass) {
	cfgs := pass.ResultOf[ctrlflow.Analyzer].(*ctrlflow.CFGs)

	visited := make(map[token.Position]bool)
	stack := make(Stack[token.Position], 0, 64)
	for position := range es.functionsWithErrors.Clone() {
		stack.Push(position)
	}
	for item := stack.Pop(); item != nil; item = stack.Pop() {
		if visited[*item] {
			continue
		}
		visited[*item] = true
		function, _ := es.functionsWithErrors.Get(*item)

		ast.Inspect(function.Node, func(n ast.Node) bool {
			if n == nil {
				return false
			}
			_, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			pos := es.tryAddCallExpr(function.Pass, cfgs, n)
			if pos != nil {
				fn, _ := es.functionsWithErrors.Get(*pos)
				fn.CalledBy.AddUnique(function.Pos)
				stack.Push(*pos)
			}
			return true
		})
	}
}

// markTaintedFunctions marks functions that return wrapped errors.
func (es *ErrStack) markTaintedFunctions() {
	settings := es.settings.Get()
	for _, function := range es.functionsWithErrors.Clone() {
		if matchFunc(settings.CleanFunctions, function.Pkg, function.Name) {
			log("Function %s.%s is clean, marking with '%t': %s\n", function.Pkg, function.Name, false, function.Pos.String())
			function.IsWrapping = false
			continue
		}
		if matchFunc(settings.WrappedFunctions, function.Pkg, function.Name) {
			log("Function %s.%s is taint, marking with '%t': %s\n", function.Pkg, function.Name, true, function.Pos.String())
			function.IsWrapping = true
			continue
		}
	}
	var visited = make(map[token.Position]bool)
	for _, function := range es.functionsWithErrors.Clone() {
		es.propagateWrapping(visited, function)
	}
}

// propagateWrapping propagates wrapping information from the given function to all its callers.
func (es *ErrStack) propagateWrapping(visited map[token.Position]bool, function *Function) {
	log("Propagating function %s.%s: %s\n", function.Pkg, function.Name, function.Pos.String())
	if !function.IsWrapping {
		log("Function %s.%s is not wrapping, skipping function\n", function.Pkg, function.Name)
		return
	}
	var stack = make(Stack[token.Position], function.CalledBy.Len())
	stack.Push(function.CalledBy.Clone()...)
	for pos := stack.Pop(); pos != nil; pos = stack.Pop() {
		if visited[*pos] {
			continue
		}
		visited[*pos] = true
		if v, ok := es.functionsWithErrors.Get(*pos); ok {
			log("Taint function %s.%s: %s\n", v.Pkg, v.Name, v.Pos.String())
			v.IsWrapping = true
			stack.Push(v.CalledBy.Clone()...)
		}
	}
}

// analyzeOriginalFunctions walks over originally found functions CFG and reports if unnecessary wrapping is used.
func (es *ErrStack) analyzeOriginalFunctions(pass *Pass) {
	cfgs := pass.ResultOf[ctrlflow.Analyzer].(*ctrlflow.CFGs)

	for _, pos := range es.originalFunctions.Clone() {
		v, ok := es.functionsWithErrors.Get(pos)
		if !ok {
			continue
		}
		es.analyzeOriginalFunction(v.Pass, cfgs, v)
		continue
	}
}

func (es *ErrStack) analyzeOriginalFunction(pass *Pass, cfgs *ctrlflow.CFGs, v *Function) {
	var wrapping bool
	var visited = make(map[*cfg.Block]bool)
	es.analyzeOriginalFunctionBlock(pass, cfgs, v.Block, wrapping, visited)
}

func (es *ErrStack) analyzeOriginalFunctionBlock(pass *Pass, cfgs *ctrlflow.CFGs, block *cfg.Block, wrapping bool, visited map[*cfg.Block]bool) {
	if block == nil || visited[block] {
		return
	}
	settings := es.settings.Get()
	esPass := es.pass.Get()

	visited[block] = true
	log("Visiting block %v\n", block)
	for _, item := range block.Nodes {
		ast.Inspect(item, func(n ast.Node) bool {
			log("Visiting node %s\n", formatNode(pass, n))
			call, ok := n.(*ast.CallExpr)
			if !ok || call == nil {
				return true
			}
			type StackCall struct {
				Fn   *Function
				Call *ast.CallExpr
			}
			var stack = Stack[StackCall]{}
			var callStack = Stack[*ast.CallExpr]{call}
			for callItem := callStack.Pop(); callItem != nil; callItem = callStack.Pop() {
				if pos := es.tryAddCallExpr(pass, cfgs, *callItem); pos != nil {
					value, _ := es.functionsWithErrors.Get(*pos)
					stack.Push(StackCall{
						Fn:   value,
						Call: *callItem,
					})
				}
				for _, arg := range (*callItem).Args {
					if callArg, ok2 := arg.(*ast.CallExpr); ok2 {
						log("Found additional call in argument %s\n", formatNode(pass, callArg))
						callStack.Push(callArg)
					}
				}
			}
			if len(stack) == 0 {
				log("Node is not a proper call %s\n", formatNode(pass, call))
				return true
			}
			ln := len(stack)
			log("Traversing call stack(%d):\n", len(stack))
			for fnItem := stack.Pop(); fnItem != nil; fnItem = stack.Pop() {
				fn := fnItem.Fn
				call := fnItem.Call
				log("Call stack item %d/%d: %s.%s\n", ln-len(stack), ln, fn.Pkg, fn.Name)
				if matchFunc(settings.WrappedFunctions, fn.Pkg, fn.Name) {
					log("Node is a wrapped call %s\n", formatNode(pass, call))
					if wrapping {
						log("Node unnecessarily wraps error with stacktrace %s\n", formatNode(pass, call))
						esPass.Reportf(call.Pos(), "%s call unnecessarily wraps error with stacktrace. Replace with errors.WithMessage() or fmt.Errorf()", fn.Name)
						return false
					}
					wrapping = true
				}
				if fn.IsWrapping {
					log("Node is a call to a function that returns wrapped errors %s\n", formatNode(pass, call))
					wrapping = true
				}
			}
			return true
		})
		for _, branch := range block.Succs {
			es.analyzeOriginalFunctionBlock(pass, cfgs, branch, wrapping, visited)
		}
	}
}

// loadApproxSelector loads the package containing the given selector and returns its AST.
func (es *ErrStack) loadApproxSelector(pass *Pass, x, sel string) (*Pass, ast.Node) {
	var retPass *Pass
	var retNode ast.Node

	for pkgName, pkg := range es.loadedPkgs.Clone() {
		if pkgName != x && !strings.HasSuffix(pkgName, "/"+x) {
			continue
		}
		retPass = &Pass{
			Fset:      pkg.Fset,
			Files:     pkg.Syntax,
			TypesInfo: pkg.TypesInfo,
			ResultOf:  pass.ResultOf,
		}

		nodes := []ast.Node{
			(*ast.FuncDecl)(nil),
		}
		inspector.New(pass.Files).Nodes(nodes, func(n ast.Node, push bool) bool {
			decl, ok := n.(*ast.FuncDecl)
			if n == nil || !ok {
				return false
			}
			if decl.Name.Name == sel {
				retNode = decl
				return false
			}
			return true
		})
		if retNode != nil {
			return retPass, retNode
		}
	}

	if retNode != nil {
		return retPass, retNode
	}
	return pass, nil
}

// loadObject loads the package containing the given object and returns its AST.
func (es *ErrStack) loadObject(pass *Pass, obj types.Object) (*Pass, ast.Node) {
	objPos := pass.Fset.Position(obj.Pos())
	if existing, ok := es.loadedObjects.Get(objPos); ok {
		return existing.Pass, existing.Node
	}
	objPkg := objPos.Filename
	pkgPath := es.getPkgPath(objPkg)
	pkg, ok := es.loadedPkgs.Get(pkgPath)
	if !ok {
		log("Package %s not found\n", pkgPath)
		return pass, nil
	}
	pass = &Pass{
		Fset:      pkg.Fset,
		TypesInfo: pkg.TypesInfo,
		Files:     pkg.Syntax,
		ResultOf:  pass.ResultOf,
	}

	var found ast.Node
	defer func() {
		log("Loaded object %#v: %s\n", found, objPos.String())
		es.loadedObjects.Set(objPos, NodeInfo{
			Pass: pass,
			Node: found,
		})
	}()

	for _, f := range pass.Files {
		if pass.Fset.Position(f.Pos()).Filename != objPos.Filename {
			continue
		}
		ast.Inspect(f, func(n ast.Node) bool {
			if n == nil || found != nil {
				return false
			}
			switch node := n.(type) {
			case *ast.FuncDecl:
				if pass.Fset.Position(node.Name.Pos()) == objPos {
					found = node
					return true
				}
			}
			return true
		})
		if found != nil {
			return pass, found
		}
	}

	return pass, found
}

// tryAddCallExpr tries to parse AST node as a function call and add its decl to the list of functions with errors.
// Returns the position of the function declaration if it was added successfully, nil otherwise.
func (es *ErrStack) tryAddCallExpr(pass *Pass, cfgs *ctrlflow.CFGs, call ast.Node) *token.Position {
	if call == nil {
		log("Trying to add nil call\n")
	}
	switch fun := call.(type) {
	case *ast.CallExpr:
		if fun.Fun == nil {
			return nil
		}
		return es.tryAddCallExpr(pass, cfgs, fun.Fun)
	case *ast.Ident:
		if fun == nil {
			return nil
		}
		if fun.Obj != nil && fun.Obj.Decl != nil {
			return es.tryAddFunction(pass, cfgs, fun.Obj.Decl)
		}
		return nil
	case *ast.SelectorExpr:
		if fun.Sel == nil {
			return nil
		}
		if fun.Sel.Obj != nil && fun.Sel.Obj.Decl != nil {
			return es.tryAddFunction(pass, cfgs, fun.Sel.Obj.Decl)
		}
		var decl ast.Node
		obj := pass.TypesInfo.ObjectOf(fun.Sel)
		if obj != nil {
			pass, decl = es.loadObject(pass, obj)
		} else {
			pass, decl = es.loadApproxSelector(pass, formatNode(pass, fun.X), fun.Sel.String())
		}
		if decl == nil {
			return nil
		}
		return es.tryAddFunction(pass, cfgs, decl)
	case *ast.StarExpr:
		if fun.X == nil {
			return nil
		}
		return es.tryAddCallExpr(pass, cfgs, fun.X)
	case *ast.ParenExpr:
		if fun.X == nil {
			return nil
		}
		return es.tryAddCallExpr(pass, cfgs, fun.X)
	case *ast.IndexExpr:
		if fun.X == nil || fun.Index == nil {
			return nil
		}
		return es.tryAddCallExpr(pass, cfgs, fun.X)
	}
	return nil
}

// tryAddFunction tries to parse AST node as a function and add it to the list of functions with errors.
// Returns the position of the function declaration if it was added successfully, nil otherwise.
// If function is already in the list, returns existing position.
func (es *ErrStack) tryAddFunction(pass *Pass, cfgs *ctrlflow.CFGs, fun any) *token.Position {
	switch decl := fun.(type) {
	case *ast.FuncDecl:
		if decl.Type.Results == nil {
			return nil
		}
		pos := pass.Fset.Position(decl.Pos())
		if _, ok := es.functionsWithErrors.Get(pos); ok {
			return &pos
		}

		var foundError bool
		for _, f := range decl.Type.Results.List {
			if ident, ok := f.Type.(*ast.Ident); ok && ident != nil && ident.Name == "error" {
				foundError = true
				break
			}
		}
		if !foundError {
			return nil
		}
		es.functionsWithErrors.Set(pos, &Function{
			Name:       decl.Name.Name,
			Node:       decl,
			Block:      getCFGBlock(cfgs, decl),
			Pos:        pos,
			IsWrapping: false,
			CalledBy:   NewList[token.Position](0, 8),
			Pkg:        es.getPkgPath(pass.Fset.Position(decl.Pos()).Filename),
			Pass:       pass,
		})
		return &pos
	case *ast.FuncLit:
		if decl.Type.Results == nil {
			return nil
		}
		pos := pass.Fset.Position(decl.Pos())
		if _, ok := es.functionsWithErrors.Get(pos); ok {
			return nil
		}

		var foundError bool
		for _, f := range decl.Type.Results.List {
			if ident, ok := f.Type.(*ast.Ident); ok && ident != nil && ident.Name == "error" {
				foundError = true
				break
			}
		}
		if !foundError {
			return nil
		}
		es.functionsWithErrors.Set(pos, &Function{
			Name:       "anonymous",
			Node:       decl,
			Block:      getCFGBlock(cfgs, decl),
			Pos:        pos,
			IsWrapping: false,
			CalledBy:   NewList[token.Position](0, 8),
			Pkg:        es.getPkgPath(pass.Fset.Position(decl.Pos()).Filename),
			Pass:       pass,
		})
		return &pos
	}

	return nil
}
