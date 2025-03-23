package preload_packages

import (
	"go/ast"
	"go/token"
	"go/types"
	"strings"
	"sync"

	"github.com/AdamBrianBright/errstack/internal/config"
	"github.com/AdamBrianBright/errstack/internal/log"
	"github.com/AdamBrianBright/errstack/internal/model"

	"golang.org/x/tools/go/ast/inspector"
	"golang.org/x/tools/go/packages"
)

type NodeInfo struct {
	Pass *model.Info
	Node ast.Node
}

type Result struct {
	m    sync.Mutex
	conf *config.Config
	Pkgs map[string]*packages.Package
	Objs map[token.Position]NodeInfo
}

// LoadSelector loads the package containing the given selector and returns its AST.
func (lp *Result) LoadSelector(info *model.Info, x, sel string) (*model.Info, ast.Node) {
	var retInfo *model.Info
	var retNode ast.Node

	lp.m.Lock()
	defer lp.m.Unlock()
	for pkgName, pkg := range lp.Pkgs {
		if pkgName != x && !strings.HasSuffix(pkgName, "/"+x) {
			continue
		}
		retInfo = &model.Info{
			Fset:  pkg.Fset,
			Files: pkg.Syntax,
			Types: pkg.TypesInfo,
		}

		nodes := []ast.Node{
			(*ast.FuncDecl)(nil),
		}
		inspector.New(info.Files).Nodes(nodes, func(n ast.Node, push bool) bool {
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
			return retInfo, retNode
		}
	}

	if retNode != nil {
		return retInfo, retNode
	}
	return info, nil
}

// LoadObject loads the package containing the given object and returns its AST.
func (lp *Result) LoadObject(info *model.Info, obj types.Object) (*model.Info, ast.Node) {
	lp.m.Lock()
	defer lp.m.Unlock()

	objPos := info.Fset.Position(obj.Pos())
	if existing, ok := lp.Objs[objPos]; ok {
		return existing.Pass, existing.Node
	}
	objPkg := objPos.Filename
	pkgPath := lp.conf.GetPkgPath(objPkg)
	pkg, ok := lp.Pkgs[pkgPath]
	if !ok {
		log.Log("Package %s not found\n", pkgPath)
		return info, nil
	}
	info = &model.Info{
		Fset:  pkg.Fset,
		Types: pkg.TypesInfo,
		Files: pkg.Syntax,
	}

	var found ast.Node
	defer func() {
		log.Log("Loaded object %q:\n %s\n\n", objPos.String(), obj.String())
		lp.Objs[objPos] = NodeInfo{
			Pass: info,
			Node: found,
		}
	}()

	for _, f := range info.Files {
		if info.Fset.Position(f.Pos()).Filename != objPos.Filename {
			continue
		}
		ast.Inspect(f, func(n ast.Node) bool {
			if n == nil || found != nil {
				return false
			}
			switch node := n.(type) {
			case *ast.FuncDecl:
				if info.Fset.Position(node.Name.Pos()) == objPos {
					found = node
					return true
				}
			}
			return true
		})
		if found != nil {
			return info, found
		}
	}

	return info, found
}
