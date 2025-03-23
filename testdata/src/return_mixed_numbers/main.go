package main

import (
	"fmt"

	"github.com/0xJacky/partialzip"
	"github.com/pkg/errors"
)

func main() {
	_ = testWrapMixed()
	_ = testWrapReduced()
	_ = testPkgError()
}

func returnSomethingAndError() (int, error) {
	return 1, fmt.Errorf("error")
}

func returnMixedNumbers() (int, int, error) {
	n, err := returnSomethingAndError()

	return 0, n, err
}

func returnReducedNumbers() (int, error) {
	_, _, err := returnMixedNumbers()
	return 0, err
}

func testWrapMixed() error {
	a, b, err := returnMixedNumbers()
	_ = a
	_ = b
	return errors.Wrap(err, "wrapped")
}

func testWrapReduced() error {
	a, err := returnReducedNumbers()
	_ = a
	return errors.Wrap(err, "wrapped")
}

func testPkgError() error {
	_, err := partialzip.New("-x-x-")

	return errors.WithStack(err) // want `WithStack call unnecessarily wraps error with stacktrace\. Replace with errors\.WithMessage\(\) or fmt\.Errorf\(\)`
}
