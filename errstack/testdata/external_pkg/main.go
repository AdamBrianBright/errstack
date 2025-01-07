package main

import (
	"git.elewise.com/elma365-libraries/json"
	"github.com/pkg/errors"
)

func main() {
	_ = testWrapExternal()
}

func testWrapExternal() error {
	_, err := json.Marshal(nil)
	return errors.Wrap(err, "wrapped") // want `errors\.Wrap call unnecessarily wraps error with stacktrace\. Replace with errors\.WithMessage\(\) or fmt\.Errorf\(\)`
}
