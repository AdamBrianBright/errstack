package model

import (
	"bytes"
	"go/ast"
	"go/format"
	"go/token"
	"go/types"

	"golang.org/x/tools/go/analysis"
)

type Info struct {
	Fset  *token.FileSet
	Types *types.Info
	Files []*ast.File
}

func NewInfo(analysisPass *analysis.Pass) *Info {
	return &Info{
		Fset:  analysisPass.Fset,
		Types: analysisPass.TypesInfo,
		Files: analysisPass.Files,
	}
}

// FormatNode returns string representation of a node as code.
func (pass *Info) FormatNode(node any) string {
	var buf bytes.Buffer
	_ = format.Node(&buf, pass.Fset, node)
	return buf.String()
}
