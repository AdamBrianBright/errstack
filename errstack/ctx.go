package errstack

import (
	"go/token"
	"go/types"
)

type Ctx struct {
	Info *types.Info
	Fset *token.FileSet
}

func (c *Ctx) Pos(pos token.Pos) string {
	return c.Fset.Position(pos).String()
}
