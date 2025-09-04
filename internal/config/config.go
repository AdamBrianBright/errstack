package config

var (
	DefaultWrapperFunctions = []PkgFunctions{
		{
			Pkg: "github.com/pkg/errors",
			Names: []string{
				"New", "Errorf", "Wrap", "Wrapf", "WithStack",
			},
			ReplaceWith:       "WithMessage",
			ReplaceWithFormat: "WithMessagef",
		},
	}
	DefaultCleanFunctions = []PkgFunctions{
		{Pkg: "github.com/pkg/errors", Names: []string{
			"WithMessage", "WithMessagef",
		}},
		{Pkg: "errors", Names: []string{
			"New",
		}},
		{Pkg: "fmt", Names: []string{
			"Errorf",
		}},
	}
	DefaultExcludePatterns []string
)

const (
	DefaultMaxDepth      = 100
	DefaultIncludeVendor = true
)

type Config struct {
	// WrapperFunctions - a list of functions that are considered to wrap errors.
	// If you're using some fancy error wrapping library like github.com/pkg/errors,
	// you may want to add it to this list.
	// If you want to ignore some functions, don't add them to the list.
	WrapperFunctions PkgsFunctions `mapstructure:"wrapperFunctions" yaml:"wrapperFunctions,omitempty"`
	// CleanFunctions - a list of functions that are considered to clean errors without stacktrace.
	CleanFunctions PkgsFunctions `mapstructure:"cleanFunctions" yaml:"cleanFunctions,omitempty"`

	// Performance tuning options
	IncludeVendor   bool     `mapstructure:"includeVendor" yaml:"includeVendor,omitempty"`
	ExcludePatterns []string `mapstructure:"excludePatterns" yaml:"excludePatterns,omitempty"`
	MaxDepth        int      `mapstructure:"maxDepth" yaml:"maxDepth,omitempty"`

	GoRoot  string `mapstructure:"-" yaml:"-"`
	WorkDir string `mapstructure:"-" yaml:"-"`
	Debug   bool   `mapstructure:"__debug" yaml:"__debug,omitempty"`
}

func NewDefaultConfig() *Config {
	return &Config{
		WrapperFunctions: DefaultWrapperFunctions,
		CleanFunctions:   DefaultCleanFunctions,
		IncludeVendor:    DefaultIncludeVendor,
		ExcludePatterns:  DefaultExcludePatterns,
		MaxDepth:         DefaultMaxDepth,
	}
}
