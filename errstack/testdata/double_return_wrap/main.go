package main

import (
	"github.com/pkg/errors"
)

func main() {
	_ = testDoubleReturnWrapStack()
	_ = testDoubleReturnWrapfStack()
	_ = testDoubleReturnStackStack()
	_ = testDoubleReturnWrapWrap()
	_ = testDoubleReturnWrapfWrap()
	_ = testDoubleReturnStackWrap()
	_ = testDoubleReturnWrapWrapf()
	_ = testDoubleReturnWrapfWrapf()
	_ = testDoubleReturnStackWrapf()
	_ = testSingleReturnWrap()
	_ = testSingleReturnWrapf()
	_ = testSingleReturnStack()
	_ = testErrorsNew()
}

func testDoubleReturnWrapStack() error {
	return errors.Wrap(errors.WithStack(nil), "wrapped") // want `errors\.Wrap call unnecessarily wraps error with stacktrace\. Replace with errors\.WithMessage\(\) or fmt\.Errorf\(\)`
}

func testDoubleReturnWrapfStack() error {
	return errors.Wrapf(errors.WithStack(nil), "wrapped %s", "arg") // want `errors\.Wrapf call unnecessarily wraps error with stacktrace\. Replace with errors\.WithMessage\(\) or fmt\.Errorf\(\)`
}
func testDoubleReturnStackStack() error {
	return errors.WithStack(errors.WithStack(nil)) // want `errors\.WithStack call unnecessarily wraps error with stacktrace\. Replace with errors\.WithMessage\(\) or fmt\.Errorf\(\)`
}

func testDoubleReturnWrapWrap() error {
	return errors.Wrap(errors.Wrap(nil, "wrapped"), "wrapped") // want `errors\.Wrap call unnecessarily wraps error with stacktrace\. Replace with errors\.WithMessage\(\) or fmt\.Errorf\(\)`
}

func testDoubleReturnWrapfWrap() error {
	return errors.Wrapf(errors.Wrap(nil, "wrapped"), "wrapped %s", "arg") // want `errors\.Wrapf call unnecessarily wraps error with stacktrace\. Replace with errors\.WithMessage\(\) or fmt\.Errorf\(\)`
}
func testDoubleReturnStackWrap() error {
	return errors.WithStack(errors.Wrap(nil, "wrapped")) // want `errors\.WithStack call unnecessarily wraps error with stacktrace\. Replace with errors\.WithMessage\(\) or fmt\.Errorf\(\)`
}

func testDoubleReturnWrapWrapf() error {
	return errors.Wrap(errors.Wrapf(nil, "wrapped %s", "arg"), "wrapped") // want `errors\.Wrap call unnecessarily wraps error with stacktrace\. Replace with errors\.WithMessage\(\) or fmt\.Errorf\(\)`
}

func testDoubleReturnWrapfWrapf() error {
	return errors.Wrapf(errors.Wrapf(nil, "wrapped %s", "arg"), "wrapped %s", "arg") // want `errors\.Wrapf call unnecessarily wraps error with stacktrace\. Replace with errors\.WithMessage\(\) or fmt\.Errorf\(\)`
}
func testDoubleReturnStackWrapf() error {
	return errors.WithStack(errors.Wrapf(nil, "wrapped %s", "arg")) // want `errors\.WithStack call unnecessarily wraps error with stacktrace\. Replace with errors\.WithMessage\(\) or fmt\.Errorf\(\)`
}

func testSingleReturnWrap() error {
	return errors.Wrap(nil, "wrapped")
}

func testSingleReturnWrapf() error {
	return errors.Wrapf(nil, "wrapped %s", "arg")
}

func testSingleReturnStack() error {
	return errors.WithStack(nil)
}

func testErrorsNew() error {
	return errors.New("error")
}
