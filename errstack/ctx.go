package errstack

import (
	"go/ast"
	"go/token"
	"go/types"
	"golang.org/x/tools/go/packages"
)

type Ctx struct {
	Path   string
	Info   *types.Info
	Fset   *token.FileSet
	Syntax []*ast.File
}

func (es *ErrStack) NewCtx(pkgPath string) *Ctx {
	if pkgPath == "" {
		return &Ctx{
			Path:   pkgPath,
			Info:   es.pass.TypesInfo,
			Fset:   es.pass.Fset,
			Syntax: es.pass.Files,
		}
	}
	// if run with `errstack ./file.go`, all nodes (and pass) will be in the package `command-line-arguments`.
	// this is a workaround for that.
	if pkgPath == "command-line-arguments" {
		pkgPath = es.dir
	}
	if pkgPath[:2] == "_/" {
		pkgPath = pkgPath[1:]
	}

	es.m.RLock()
	pkg, ok := es.pkgs[pkgPath]
	es.m.RUnlock()
	if !ok {
		es.m.Lock()
		defer es.m.Unlock()

		pkgs, err := packages.Load(&packages.Config{
			Mode: packages.NeedSyntax | packages.NeedTypes | packages.NeedTypesInfo | packages.NeedImports | packages.NeedDeps | packages.NeedModule | packages.NeedCompiledGoFiles | packages.NeedFiles,
		}, pkgPath)
		if err != nil || len(pkgs) == 0 {
			return nil
		}
		es.pkgs[pkgPath] = pkgs[0]
		pkg = pkgs[0]
	}

	return &Ctx{
		Path:   pkgPath,
		Info:   pkg.TypesInfo,
		Fset:   pkg.Fset,
		Syntax: pkg.Syntax,
	}
}
func (c *Ctx) Pos(pos token.Pos) string {
	return c.Fset.Position(pos).String()
}
