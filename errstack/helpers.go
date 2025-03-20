package errstack

import (
	"bytes"
	"go/ast"
	"go/format"
	"go/token"
	"go/types"
	"path/filepath"
	"slices"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/ctrlflow"
	"golang.org/x/tools/go/cfg"
)

type Pass struct {
	Fset      *token.FileSet
	TypesInfo *types.Info
	Files     []*ast.File
	ResultOf  map[*analysis.Analyzer]interface{}
}

func NewPass(analysisPass *analysis.Pass) *Pass {
	return &Pass{
		Fset:      analysisPass.Fset,
		TypesInfo: analysisPass.TypesInfo,
		Files:     analysisPass.Files,
		ResultOf:  analysisPass.ResultOf,
	}
}

// formatNode returns string representation of node as code.
func formatNode(pass *Pass, node any) string {
	var buf bytes.Buffer
	_ = format.Node(&buf, pass.Fset, node)
	return buf.String()
}

// matchFunc returns true if function matches any of package functions.
func matchFunc(l []PkgFunctions, pkg, name string) bool {
	for _, item := range l {
		if item.Pkg == pkg && slices.Contains(item.Names, name) {
			return true
		}
	}

	return false
}

// getCFGBlock returns the first block of the CFG for the given node.
func getCFGBlock(cfgs *ctrlflow.CFGs, node ast.Node) *cfg.Block {
	defer func() {
		recover()
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

func (es *ErrStack) getPkgPath(dir string) string {
	return es.getDirPkgPath(filepath.Dir(dir))
}

func (es *ErrStack) getDirPkgPath(dir string) string {
	settings := es.settings.Get()
	if strings.HasPrefix(dir, settings.WorkDir) {
		dir = strings.TrimPrefix(dir, settings.WorkDir)
		dir = strings.TrimPrefix(dir, "vendor/")
		return dir
	}
	if strings.HasPrefix(dir, settings.GoRoot) {
		return strings.TrimPrefix(dir, settings.GoRoot)
	}

	return dir
}
