package config

import "slices"

type PkgFunctions struct {
	Pkg   string   `mapstructure:"pkg" yaml:"pkg"`
	Names []string `mapstructure:"names" yaml:"names"`
}

type PkgsFunctions []PkgFunctions

// Match returns true if function matches any of package functions.
func (pkgFunctions PkgsFunctions) Match(pkg, name string) bool {
	for _, item := range pkgFunctions {
		if item.Pkg == pkg && slices.Contains(item.Names, name) {
			return true
		}
	}

	return false
}
