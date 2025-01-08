package main

import (
	"fmt"
	"github.com/pkg/errors"
	"time"
)

func main() {
	_ = testNestedSelector()
	_ = testTypeMethodDirect()
}

type Foo struct{}

func (f Foo) Method() (int, error) {
	return 0, fmt.Errorf("error")
}

type Bar struct {
	Foo Foo
}

type Baz struct {
	Bar Bar
}

func testNestedSelector() error {
	baz := Baz{Bar{Foo{}}}
	_, err := baz.Bar.Foo.Method()

	err = errors.Wrap(err, "wrapped")
	return errors.Wrap(err, "wrapped") // want `Wrap call unnecessarily wraps error with stacktrace\. Replace with errors\.WithMessage\(\) or fmt\.Errorf\(\)`
}

func testTypeMethodDirect() error {
	_ = time.Time.UTC(time.Time{})

	return errors.New("error")
}
