package main

import (
	"fmt"

	"github.com/pkg/errors"
)

func main() {
	_ = testWrapWrapVariable()
	_ = testWrapfWrapVariable()
	_ = testWithStackWrapVariable()
	_ = testWrapWrapfVariable()
	_ = testWrapfWrapfVariable()
	_ = testWithStackWrapfVariable()
	_ = testWrapWithStackVariable()
	_ = testWrapfWithStackVariable()
	_ = testWithStackWithStackVariable()
	_ = testWrapUnwrappedVariable()
	_ = testWrapfUnwrappedVariable()
	_ = testWithStackUnwrappedVariable()
	_ = testNoWrapVariable()
	_ = testWithMessageVariable()
	_ = testWithMessagefVariable()
}

func testWrapWrapVariable() error {
	err := fmt.Errorf("error")
	err = errors.Wrap(err, "wrapped")
	return errors.Wrap(err, "wrapped") // want `Wrap call unnecessarily wraps error with stacktrace\. Replace with errors\.WithMessage\(\) or fmt\.Errorf\(\)`
}
func testWrapfWrapVariable() error {
	err := fmt.Errorf("error")
	err = errors.Wrap(err, "wrapped")
	return errors.Wrapf(err, "wrapped") // want `Wrapf call unnecessarily wraps error with stacktrace\. Replace with errors\.WithMessage\(\) or fmt\.Errorf\(\)`
}

func testWithStackWrapVariable() error {
	err := fmt.Errorf("error")
	err = errors.Wrap(err, "wrapped")
	return errors.WithStack(err) // want `WithStack call unnecessarily wraps error with stacktrace\. Replace with errors\.WithMessage\(\) or fmt\.Errorf\(\)`
}
func testWrapWrapfVariable() error {
	err := fmt.Errorf("error")
	err = errors.WithStack(err)
	return errors.Wrap(err, "wrapped") // want `Wrap call unnecessarily wraps error with stacktrace\. Replace with errors\.WithMessage\(\) or fmt\.Errorf\(\)`
}
func testWrapfWrapfVariable() error {
	err := fmt.Errorf("error")
	err = errors.WithStack(err)
	return errors.Wrapf(err, "wrapped") // want `Wrapf call unnecessarily wraps error with stacktrace\. Replace with errors\.WithMessage\(\) or fmt\.Errorf\(\)`
}

func testWithStackWrapfVariable() error {
	err := fmt.Errorf("error")
	err = errors.WithStack(err)
	return errors.WithStack(err) // want `WithStack call unnecessarily wraps error with stacktrace\. Replace with errors\.WithMessage\(\) or fmt\.Errorf\(\)`
}

func testWrapWithStackVariable() error {
	err := fmt.Errorf("error")
	err = errors.WithStack(err)
	return errors.Wrap(err, "wrapped") // want `Wrap call unnecessarily wraps error with stacktrace\. Replace with errors\.WithMessage\(\) or fmt\.Errorf\(\)`
}
func testWrapfWithStackVariable() error {
	err := fmt.Errorf("error")
	err = errors.WithStack(err)
	return errors.Wrapf(err, "wrapped") // want `Wrapf call unnecessarily wraps error with stacktrace\. Replace with errors\.WithMessage\(\) or fmt\.Errorf\(\)`
}

func testWithStackWithStackVariable() error {
	err := fmt.Errorf("error")
	err = errors.WithStack(err)
	return errors.WithStack(err) // want `WithStack call unnecessarily wraps error with stacktrace\. Replace with errors\.WithMessage\(\) or fmt\.Errorf\(\)`
}

func testWrapUnwrappedVariable() error {
	err := fmt.Errorf("error")
	return errors.Wrap(err, "wrapped")
}
func testWrapfUnwrappedVariable() error {
	err := fmt.Errorf("error")
	return errors.Wrapf(err, "wrapped")
}

func testWithStackUnwrappedVariable() error {
	err := fmt.Errorf("error")
	return errors.WithStack(err)
}

func testNoWrapVariable() error {
	err := fmt.Errorf("error")
	err = errors.WithStack(err)
	return err
}

func testWithMessageVariable() error {
	err := fmt.Errorf("error")
	err = errors.WithStack(err)
	return errors.WithMessage(err, "message")
}

func testWithMessagefVariable() error {
	err := fmt.Errorf("error")
	err = errors.WithStack(err)
	return errors.WithMessagef(err, "message")
}
