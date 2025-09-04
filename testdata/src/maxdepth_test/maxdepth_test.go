package maxdepth_test

import (
	"github.com/pkg/errors"
)

// This creates a deep call chain to test MaxDepth functionality
// The chain is: level0 -> level1 -> level2 -> level3 -> level4 -> level5

func level0() error {
	return level1() // depth 0
}

func level1() error {
	return level2() // depth 1
}

func level2() error {
	return level3() // depth 2
}

func level3() error {
	return level4() // depth 3
}

func level4() error {
	return level5() // depth 4
}

func level5() error {
	// This should be found when MaxDepth allows deep traversal
	// But may be missed when MaxDepth limits the analysis depth
	return errors.Wrap(errors.New("base error"), "wrapped") // want "Wrap call unnecessarily wraps error"
}
