package main

import (
	"github.com/0xJacky/partialzip"
	"github.com/pkg/errors"
)

func main() {
	_ = testWrapExternal()
}

func testWrapExternal() error {
	_, err := partialzip.New("-x-x-")
	return errors.Wrap(err, "wrapped") // want `Wrap call unnecessarily wraps error with stacktrace\. Replace with errors\.WithMessage\(\) or fmt\.Errorf\(\)`
}
