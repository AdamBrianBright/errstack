package log

import (
	"fmt"
	"io"
	"os"
	"sync"
)

var (
	w    *os.File
	wm   sync.Mutex
	once sync.Once
	Sync = func() {}
	Log  = func(format string, args ...any) {}
)

func EnableDebug(debug bool) {
	once.Do(func() {
		if !debug {
			return
		}

		f, err := os.OpenFile("errstack.log", os.O_RDWR|os.O_SYNC|os.O_CREATE, 0644)
		if err != nil {
			panic(err)
		}
		w = f
		_, _ = f.Seek(0, io.SeekStart)
		_ = f.Truncate(0)
		_ = f.Sync()

		Sync = func() {
			wm.Lock()
			defer wm.Unlock()
			_ = w.Sync()
		}

		Log = func(format string, args ...any) {
			wm.Lock()
			defer wm.Unlock()
			_, _ = w.WriteString(fmt.Sprintf(format, args...))
		}
	})
}
