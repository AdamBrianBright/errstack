package main

import (
	"fmt"
	"github.com/pkg/errors"
)

func main() {
	_ = testWrapVariadic()
	_ = testWrapWrappedVariadic()
}

func returnSomethingAndError() (int, error) {
	return 1, fmt.Errorf("error")
}

func returnVariadic() (int, error) {
	return returnSomethingAndError()
}

func testWrapVariadic() error {
	n, err := returnVariadic()
	_ = n
	return errors.Wrap(err, "wrapped")
}

func testWrapWrappedVariadic() error {
	n, err := returnVariadic()
	_ = n
	err = errors.Wrap(err, "wrapped")
	return errors.Wrap(err, "wrapped") // want `Wrap call unnecessarily wraps error with stacktrace\. Replace with errors\.WithMessage\(\) or fmt\.Errorf\(\)`
}
