package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
)

func main() {
	_ = testBuiltin()
	_ = testJson()
}

func testBuiltin() error {
	err := fmt.Errorf("error")
	a := []int{1, 2, 3}
	a = append(a, 4)
	_ = a

	return errors.Wrap(err, "wrapped")
}

func testJson() error {
	w := bufio.NewWriter(nil)
	err := json.NewEncoder(w).Encode(struct{}{})

	return errors.Wrap(err, "wrapped")
}
