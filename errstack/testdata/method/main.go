package main

import (
	"fmt"
	"github.com/pkg/errors"
)

func main() {
	_ = testWrapMethodWrapped()
	_ = testWrapMethodUnwrapped()
}

type TestStruct struct{}

func (t TestStruct) TestMethod() error {
	return errors.New("error")
}
func (t TestStruct) TestMethodUnwrapped() error {
	return fmt.Errorf("error")
}

func testWrapMethodWrapped() error {
	tester := TestStruct{}
	err := tester.TestMethod()

	return errors.Wrap(err, "wrapped") // want `Wrap call unnecessarily wraps error with stacktrace\. Replace with errors\.WithMessage\(\) or fmt\.Errorf\(\)`
}

func testWrapMethodUnwrapped() error {
	tester := TestStruct{}
	err := tester.TestMethodUnwrapped()

	return errors.Wrap(err, "wrapped")
}
