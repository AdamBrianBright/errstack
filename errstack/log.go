//go:build debug

package errstack

import (
	"fmt"
	"io"
	"os"
	"sync"
)

var w *os.File
var wm sync.Mutex

func init() {
	f, err := os.OpenFile("errstack.log", os.O_RDWR|os.O_SYNC|os.O_CREATE, 0644)
	if err != nil {
		panic(err)
	}
	w = f
	_, _ = f.Seek(0, io.SeekStart)
	_ = f.Truncate(0)
	_ = f.Sync()
}

func log(format string, args ...any) {
	wm.Lock()
	defer wm.Unlock()
	_, _ = w.WriteString(fmt.Sprintf(format, args...))
}
