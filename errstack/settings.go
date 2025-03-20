package errstack

type PkgFunctions struct {
	Pkg   string   `mapstructure:"pkg" yaml:"pkg"`
	Names []string `mapstructure:"names" yaml:"names"`
}

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
	WrapperFunctions []PkgFunctions `mapstructure:"wrapperFunctions" yaml:"wrapperFunctions"`
	// CleanFunctions - list of functions that are considered to clean errors without stacktrace.
	CleanFunctions []PkgFunctions `mapstructure:"cleanFunctions" yaml:"cleanFunctions"`
}

func NewDefaultConfig() Config {
	return Config{
		WrapperFunctions: DefaultWrapperFunctions,
		CleanFunctions:   DefaultCleanFunctions,
	}
}
