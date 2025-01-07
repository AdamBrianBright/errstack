package main

import (
	"fmt"
	"github.com/pkg/errors"
)

func main() {
	_ = testWrapFuncVariable()
}

func testWrapFuncVariable() error {
	f := func() error {
		return fmt.Errorf("error")
	}
	err := f()
	err = errors.Wrap(err, "wrapped")
	return errors.Wrap(err, "wrapped") // want `errors\.Wrap call unnecessarily wraps error with stacktrace\. Replace with errors\.WithMessage\(\) or fmt\.Errorf\(\)`
}
