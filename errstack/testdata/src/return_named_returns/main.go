package main

import (
	"fmt"

	"github.com/pkg/errors"
)

func main() {
	_ = testWrapNamedReturns()
}

func returnSomethingAndError() (int, error) {
	return 1, fmt.Errorf("error")
}

func returnNamedReturns() (n int, err error) {
	n, err = returnSomethingAndError()
	return
}

func testWrapNamedReturns() error {
	n, err := returnNamedReturns()
	_ = n
	return errors.Wrap(err, "wrapped")
}

func testWrapWrappedNamedReturns() error {
	n, err := returnNamedReturns()
	_ = n
	err = errors.Wrap(err, "wrapped")
	return errors.Wrap(err, "wrapped") // want `Wrap call unnecessarily wraps error with stacktrace\. Replace with errors\.WithMessage\(\) or fmt\.Errorf\(\)`
}
