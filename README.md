# ErrStack

**ErrStack** is a linter for Go that checks for unnecessary error wrapping using `errors.Wrap`, `errors.Wrapf`, and `errors.WithStack`.

It is created as a complement to the [wrapcheck](https://github.com/tomarrell/wrapcheck) linter.

## Installation

Go `>= v1.21`

```bash
go install github.com/AdamBrianBright/errstack/cmd/errstack@latest
```

ErrStack can be used as a module for [golangci-lint](https://golangci-lint.run/usage/linters/#modules).

## Configuration

You can configure ErrStack using the `.errstack.yaml` file in your project root, or in your home directory.

```yaml
# List of functions that are considered to wrap errors.
# If you're using some fancy error wrapping library like github.com/pkg/errors,
# you may want to add it to this list.
# If you want to ignore some functions, simply don't add them to the list.
wrapperFunctions:
  - pkg: github.com/pkg/errors
    names:
      - New
      - Errorf
      - Wrap
      - Wrapf
      - WithStack
cleanFunctions:
  - pkg: errors
    names:
      - New
  - pkg: fmt
    names:
      - Errorf
  - pkg: github.com/pkg/errors
    names:
      - WithMessage
      - WithMessagef
```

## Usage

To lint all the packages in your project, run:

```bash
errstack ./...
```

## Testing

This linter is tested using `analysistest`, you can view all the test cases under `testdata` directory.

## Why?

If you're using some fancy error wrapping library like [github.com/pkg/errors](https://pkg.go.dev/github.com/pkg/errors), you may have stumbled upon doubling or tripling the amount of stacktrace duplicates in your logs.

This happens because the library wraps errors in context style, hiding stacktraces from the user in unexported structs and fields like russian dolls.

When doing so, libraries don't check for stacktraces already present in the error, since it is usually not necessary and only slows down your code.

However, if you're using libraries out of your control, you may not be able to easily identify whether some functions may return wrapped errors or not, and just wrap errors from external packages like [wrapcheck](https://github.com/tomarrell/wrapcheck) suggests anyways.

This linter helps you to identify such cases, and help you remove unnecessary wrapping.

## How does it work?

ErrStack finds all calls to configured list of wrapping functions in your code and finds the source of the error.

When the source of an error is located up to it's root (assigment statement, or return statement with errors.New() passed as an argument), it check if the error was wrapped, excluding nil errors.

!!! This linter doesn't verify the actual types as it's almost impossible to do so and usually pointless.

Linter calculates the amount of non-nil branches and prints a warning if it's greater than the configured threshold.

### Example

```go
package main

import (
	"github.com/pkg/errors"
)

func main() {
	_ = testDoubleReturnWrapStack()
}

func testDoubleReturnWrapStack() error {
	return errors.Wrap(errors.WithStack(nil), "wrapped") // want `WithStack call unnecessarily wraps error with stacktrace. Replace with errors.WithMessage()`
}
```