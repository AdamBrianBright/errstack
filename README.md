# 🛡️ ErrStack

**ErrStack** is a powerful Go linter that detects unnecessary error wrapping using `errors.Wrap`, `errors.Wrapf`, and
`errors.WithStack`, helping you maintain clean and efficient error handling in your codebase.

Created as the perfect complement to the [wrapcheck](https://github.com/tomarrell/wrapcheck) linter.

## 📦 Installation

### Requirements

- Go `>= v1.23.12`

### Standalone Installation

```bash
go install github.com/AdamBrianBright/errstack/cmd/errstack@latest
```

### 🔧 golangci-lint Integration

ErrStack can be used as a module for [golangci-lint](https://golangci-lint.run/usage/linters/#modules).

`.custom-gcl.yml`

```yaml .custom-gcl.yml
version: v1.64.8

destination: ./testdata/src

plugins:
  - module: 'github.com/AdamBrianBright/errstack'
    import: 'github.com/AdamBrianBright/errstack/cmd/gclplugin'
    version: v0.3.4
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
          - pkg: fmt
            names: [ Errorf ]
          - pkg: github.com/pkg/errors
            names: [ WithMessage, WithMessagef ]

linters:
  disable-all: true
  enable:
    - errstack
```

## ⚙️ Configuration

You can configure ErrStack using the `.errstack.yaml` file in your project root, or in your home directory.

```yaml
# List of functions that are considered to wrap errors.
# If you're using some fancy error wrapping library like github.com/pkg/errors,
# you may want to add it to this list.
# If you want to ignore some functions, simply don't add them to the list.
wrapperFunctions:
  - pkg: github.com/pkg/errors
    names: [ New, Errorf, Wrap, Wrapf, WithStack ]
    replaceWith: WithMessage     # Optional. Attempts to replace errors.Wrap like functions with errors.WithMessage.
    replaceWithFormat: WithMessagef  # Optional. Attempts to replace errors.Wrapf like functions with errors.WithMessagef.

# List of functions that are considered to clean errors without stacktrace.
cleanFunctions:
  - pkg: errors
    names: [ New ]
  - pkg: fmt
    names: [ Errorf ]
  - pkg: github.com/pkg/errors
    names: [ WithMessage, WithMessagef ]

# Performance tuning options
includeVendor: true      # Whether to include vendor packages in analysis
maxDepth: 100           # Maximum depth for analysis to prevent infinite loops
excludePatterns: [ ]     # Patterns to exclude from analysis (e.g., ["test", "mock"])
```

## 🚀 Usage

To lint all the packages in your project, run:

```bash
errstack ./...
```

You can also target specific packages:

```bash
errstack ./internal/...
errstack ./pkg/mypackage
```

## 🧪 Testing

This linter is thoroughly tested using `analysistest`. You can view all the test cases under the `testdata` directory to
understand various scenarios and edge cases.

## 🤔 Why ErrStack?

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

This linter helps you identify such cases and remove unnecessary wrapping, keeping your error handling clean and
efficient.

## 🔍 How Does It Work?

ErrStack uses advanced static analysis to detect redundant error wrapping:

1. **📦 Package Analysis** - Preloads all packages and parses their ASTs
2. **🎯 Function Discovery** - Finds all functions that return errors
3. **📞 Call Tracking** - Identifies all calls to error-returning functions
4. **🏷️ Wrapping Detection** - Marks functions that return wrapped errors
5. **🔬 Flow Analysis** - Analyzes control flow graphs and reports unnecessary wrapping

### 💡 Example

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