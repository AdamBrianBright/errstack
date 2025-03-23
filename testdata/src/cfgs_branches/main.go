package main

import (
	"fmt"

	"github.com/pkg/errors"
)

func main() {
	_ = testErrReassign()
	_ = testErrPointer()
	_ = testErrShadow()
	_ = testErrSecondVariable()
}

func testErrReassign() error {
	err := errors.New("-x-x-")
	err = fmt.Errorf("error")
	return errors.Wrap(err, "wrapped")
}

func testErrPointer() error {
	var err *error
	if err2 := errors.New("-x-x-"); err2 != nil {
		err = &err2
	}
	if err != nil {
		return errors.Wrap(*err, "wrapped")
	}
	return nil
}

func testErrShadow() error {
	err := errors.New("-x-x-")
	_ = err
	if err := fmt.Errorf("error"); err != nil {
		return errors.Wrap(err, "wrapped")
	}
	return nil
}

func testErrSecondVariable() error {
	err := errors.New("-x-x-")
	if err != nil {
		return err
	}
	if err2 := fmt.Errorf("error"); err2 != nil {
		return errors.Wrap(err2, "wrapped")
	}
	return nil
}
