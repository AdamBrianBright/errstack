package main

import (
	"fmt"
)

func main() {
	_ = testFuncAsVar()
}

func testFuncAsVar() error {
	f := func() error {
		return fmt.Errorf("error")
	}
	err := f()

	return err
}
