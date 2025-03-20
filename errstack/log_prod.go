//go:build !debug

package errstack

import (
	"os"
	"sync"
)

var w *os.File
var wm sync.Mutex

func log(format string, args ...any) {}
