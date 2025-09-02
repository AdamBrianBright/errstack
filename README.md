# ErrStack

**ErrStack** is a linter for Go that checks for unnecessary error wrapping using `errors.Wrap`, `errors.Wrapf`, and
`errors.WithStack`.

It is created as a complement to the [wrapcheck](https://github.com/tomarrell/wrapcheck) linter.

## Installation

Go `>= v1.23.12`

```bash
go install github.com/AdamBrianBright/errstack/cmd/errstack@latest
```

ErrStack can be used as a module for [golangci-lint](https://golangci-lint.run/usage/linters/#modules).

`.custom-gcl.yml`
```yaml .custom-gcl.yml
version: v1.64.8

destination: ./testdata/src

plugins:
  - module: 'github.com/AdamBrianBright/errstack'
    import: 'github.com/AdamBrianBright/errstack/cmd/gclplugin'
    version: v0.3.3
```

`.golangci.yml`
```yaml .golangci.yml
linters-settings:
  custom:
    errstack:
      type: "module"
      description: Finds unnecessary error wraps with stacktraces.
      settings:
        wrapperFunctions:
          - pkg: github.com/pkg/errors
            names: [ New, Errorf, Wrap, Wrapf, WithStack ] 
            replaceWith: WithMessage
            replaceWithFormat: WithMessagef
        cleanFunctions:
          - pkg: errors
            names: [ New ]
          - pkg: github.com/pkg/errors
            names: [ WithMessage, WithMessagef ]

linters:
  disable-all: true
  enable:
    - errstack
```

## Configuration

You can configure ErrStack using the `.errstack.yaml` file in your project root, or in your home directory.

```yaml
# List of functions that are considered to wrap errors.
# If you're using some fancy error wrapping library like github.com/pkg/errors,
# you may want to add it to this list.
# If you want to ignore some functions, simply don't add them to the list.
wrapperFunctions:
  - pkg: github.com/pkg/errors
    names: [ New, Errorf, Wrap, Wrapf, WithStack ]
    replaceWith: WithMessage # Optional. Attempts to replace errors.Wrap like functions with errors.WithMessage.
    replaceWithFormat: WithMessagef # Optional. Attempts to replace errors.Wrapf like functions with errors.WithMessagef.
# List of functions that are considered to clean errors without stacktrace.
cleanFunctions:
    - pkg: errors
      names: [ New ]
    - pkg: github.com/pkg/errors
      names: [ WithMessage, WithMessagef ]
```

## Usage

To lint all the packages in your project, run:

```bash
errstack ./...
```

## Testing

This linter is tested using `analysistest`, you can view all the test cases under `testdata` directory.

## Why?

If you're using some fancy error wrapping library
like [github.com/pkg/errors](https://pkg.go.dev/github.com/pkg/errors), you may have stumbled upon doubling or tripling
the amount of stacktrace duplicates in your logs.

This happens because the library wraps errors in context style, hiding stacktraces from the user in unexported structs
and fields like russian dolls.

When doing so, libraries don't check for stacktraces already present in the error, since it is usually not necessary and
only slows down your code.

However, if you're using libraries out of your control, you may not be able to easily identify whether some functions
may return wrapped errors or not, and just wrap errors from external packages
like [wrapcheck](https://github.com/tomarrell/wrapcheck) suggests anyways.

This linter helps you to identify such cases, and help you remove unnecessary wrapping.

## How does it work?

1. Preloads all packages and parses their ASTs.
2. Finds all functions that return errors.
3. Finds all calls to functions that return errors.
4. Marks functions that return wrapped errors.
5. Analyzes original function CFG and reports if unnecessary wrapping is used.

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
	return errors.Wrap(errors.WithStack(nil), "wrapped") // want `Wrap call unnecessarily wraps error with stacktrace\. Replace with errors\.WithMessage\(\) or fmt\.Errorf\(\)`
}
```