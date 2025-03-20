package main

import (
	"fmt"

	"github.com/pkg/errors"
)

func main() {
	_ = testSameNameOnePkg()
}

type Foo struct{}

func (f Foo) Method() (int, error) {
	return 0, fmt.Errorf("error")
}

type Bar struct {
	Foo
}

func (f Bar) Method() error {
	_, err := f.Foo.Method()
	return err
}

func testSameNameOnePkg() error {
	err := Bar{Foo{}}.Method()
	err = errors.Wrap(err, "wrapped")
	return errors.Wrap(err, "wrapped") // want `Wrap call unnecessarily wraps error with stacktrace\. Replace with errors\.WithMessage\(\) or fmt\.Errorf\(\)`
}
