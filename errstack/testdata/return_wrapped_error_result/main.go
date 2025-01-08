package main

import (
	"fmt"
	"github.com/pkg/errors"
)

func main() {
	_ = testWrapErrorResult()
	_ = testWrapfErrorResult()
	_ = testWithStackErrorResult()
	_ = testWrapUnwrappedErrorResult()
	_ = testWrapfUnwrappedErrorResult()
	_ = testWithStackUnwrappedErrorResult()
	_ = testNoWrapErrorResult()
	_ = testWithMessageErrorResult()
	_ = testWithMessagefErrorResult()
}

func testWrapErrorResult() error {
	err := returnsWrappedError()
	return errors.Wrap(err, "wrapped") // want `Wrap call unnecessarily wraps error with stacktrace\. Replace with errors\.WithMessage\(\) or fmt\.Errorf\(\)`
}
func testWrapfErrorResult() error {
	err := returnsWrappedError()
	return errors.Wrapf(err, "wrapped") // want `Wrapf call unnecessarily wraps error with stacktrace\. Replace with errors\.WithMessage\(\) or fmt\.Errorf\(\)`
}

func testWithStackErrorResult() error {
	err := returnsWrappedError()
	return errors.WithStack(err) // want `WithStack call unnecessarily wraps error with stacktrace\. Replace with errors\.WithMessage\(\) or fmt\.Errorf\(\)`
}

func testWrapUnwrappedErrorResult() error {
	err := returnsError()
	return errors.Wrap(err, "wrapped")
}
func testWrapfUnwrappedErrorResult() error {
	err := returnsError()
	return errors.Wrapf(err, "wrapped")
}

func testWithStackUnwrappedErrorResult() error {
	err := returnsError()
	return errors.WithStack(err)
}

func testNoWrapErrorResult() error {
	err := returnsWrappedError()
	return err
}

func testWithMessageErrorResult() error {
	err := returnsWrappedError()
	return errors.WithMessage(err, "message")
}

func testWithMessagefErrorResult() error {
	err := returnsWrappedError()
	return errors.WithMessagef(err, "message")
}

func returnsError() error {
	return fmt.Errorf("error")
}

func returnsWrappedError() error {
	return errors.Wrap(returnsError(), "wrapped")
}
