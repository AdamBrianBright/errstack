package model

import (
	"go/ast"
	"go/token"

	"golang.org/x/tools/go/cfg"
)

type Function struct {
	Name       string           // Name of the function
	Node       ast.Node         // AST node of the function
	Type       *ast.FuncType    // Type of the function
	Body       *ast.BlockStmt   // Body of the function
	Block      *cfg.Block       // Control flow graph of the function
	Pos        token.Position   // Position of the function declaration
	IsWrapping bool             // Is true if this function returns wrapped errors
	CalledBy   Stack[*Function] // Functions that call this function
	Pkg        string           // Package containing the function
	Info       *Info            // Info used to load the function
}
