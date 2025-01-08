package main

import (
	"fmt"
	"git.elewise.com/elma365-libraries/json"
	"github.com/pkg/errors"
)

func main() {
	_ = testWrapMixed()
	_ = testWrapReduced()
	_ = testJsonDecoder()
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

func testJsonDecoder() error {
	var v any
	err := json.NewDecoder(nil).Decode(v)

	return errors.WithStack(err)
}
