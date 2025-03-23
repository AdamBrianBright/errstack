package config

var (
	DefaultWrapperFunctions = []PkgFunctions{
		{Pkg: "github.com/pkg/errors", Names: []string{
			"New", "Errorf", "Wrap", "Wrapf", "WithStack",
		}},
	}
	DefaultCleanFunctions = []PkgFunctions{
		{Pkg: "github.com/pkg/errors", Names: []string{
			"WithMessage", "WithMessagef",
		}},
		{Pkg: "errors", Names: []string{
			"New", "Wrapf", "WithStack",
		}},
		{Pkg: "fmt", Names: []string{
			"Errorf",
		}},
	}
)

type Config struct {
	// WrapperFunctions - list of functions that are considered to wrap errors.
	// If you're using some fancy error wrapping library like github.com/pkg/errors,
	// you may want to add it to this list.
	// If you want to ignore some functions, simply don't add them to the list.
	WrapperFunctions PkgsFunctions `mapstructure:"wrapperFunctions" yaml:"wrapperFunctions,omitempty"`
	// CleanFunctions - list of functions that are considered to clean errors without stacktrace.
	CleanFunctions PkgsFunctions `mapstructure:"cleanFunctions" yaml:"cleanFunctions,omitempty"`

	GoRoot  string `mapstructure:"-" yaml:"-"`
	WorkDir string `mapstructure:"-" yaml:"-"`
	Debug   bool   `mapstructure:"__debug" yaml:"__debug,omitempty"`
}

func NewDefaultConfig() *Config {
	return &Config{
		WrapperFunctions: DefaultWrapperFunctions,
		CleanFunctions:   DefaultCleanFunctions,
	}
}
