package main

import (
	"github.com/pkg/errors"
)

func main() {
	_ = testInterface(Bar{Foo{}})
}

type FooI interface {
	Method() error
}

type Foo struct{}

func (f Foo) Method() error {
	return errors.New("error")
}

type Bar struct {
	Foo FooI
}

func testInterface(bar Bar) error {
	return bar.Foo.Method()
}
