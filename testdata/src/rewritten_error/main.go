package main

import (
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"
)

func main() {
	_ = testRewritten()
	_ = testRewrittenReallocated()
	_ = testRewrittenLocalScope()
	_ = testRewrittenReallocatedLocalScope()
}

func returnSomethingAndError() (int, error) {
	return 1, errors.WithStack(fmt.Errorf("error"))
}

func testRewritten() error {
	_, err := returnSomethingAndError()
	if err != nil {
		return errors.WithMessagef(err, "message")
	}

	var data []byte
	data, err = json.Marshal(map[string]any{})
	if err != nil {
		return errors.WithStack(err)
	}

	_ = data

	return nil
}

func testRewrittenReallocated() error {
	_, err := returnSomethingAndError()
	if err != nil {
		return errors.WithMessagef(err, "message")
	}

	data, err := json.Marshal(map[string]any{})
	if err != nil {
		return errors.WithStack(err)
	}

	_ = data

	return nil
}

func testRewrittenLocalScope() error {
	_, err := returnSomethingAndError()
	if err != nil {
		return errors.WithMessagef(err, "message")
	}

	if _, err = json.Marshal(map[string]any{}); err != nil {
		return errors.WithStack(err)
	}

	return nil
}

func testRewrittenReallocatedLocalScope() error {
	_, err := returnSomethingAndError()
	if err != nil {
		return errors.WithMessagef(err, "message")
	}

	if _, err := json.Marshal(map[string]any{}); err != nil {
		return errors.WithStack(err)
	}

	return nil
}
