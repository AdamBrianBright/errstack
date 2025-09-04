package config

import (
	"slices"
	"strings"
)

type PkgFunctions struct {
	Pkg               string   `mapstructure:"pkg" yaml:"pkg"`
	Names             []string `mapstructure:"names" yaml:"names"`
	ReplaceWith       string   `mapstructure:"replaceWith" yaml:"replaceWith"`
	ReplaceWithFormat string   `mapstructure:"replaceWithFormat" yaml:"replaceWithFormat"`
}

type PkgsFunctions []PkgFunctions

// Match returns true if a function matches any of the package functions.
func (pkgFunctions PkgsFunctions) Match(pkg, name string) bool {
	for _, item := range pkgFunctions {
		if item.Pkg == pkg && slices.Contains(item.Names, name) {
			return true
		}
	}

	return false
}

// ReplaceWith returns new formatted node with replaced function name.
func (pkgFunctions PkgsFunctions) ReplaceWith(pkg, name, text string) string {
	for _, item := range pkgFunctions {
		if item.Pkg == pkg && slices.Contains(item.Names, name) {
			if item.ReplaceWith == "" {
				return ""
			}
			return strings.Replace(text, name, item.ReplaceWith, 1)
		}
	}

	return ""
}

// ReplaceWithFunction returns new formatted node with replaced function name.
func (pkgFunctions PkgsFunctions) ReplaceWithFunction(pkg, name, text string) string {
	for _, item := range pkgFunctions {
		if item.Pkg == pkg && slices.Contains(item.Names, name) {
			if item.ReplaceWithFormat == "" {
				return ""
			}
			return strings.Replace(text, name, item.ReplaceWithFormat, 1)
		}
	}

	return ""
}
