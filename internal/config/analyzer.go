package config

import (
	"flag"
	"fmt"
	"os"
	"reflect"

	"github.com/AdamBrianBright/errstack/internal/helpers"
	"github.com/AdamBrianBright/errstack/internal/log"

	"golang.org/x/tools/go/analysis"
	"gopkg.in/yaml.v3"
)

const _doc = `errstack_config analyzer is responsible to take configurations (flags) for ErrStack execution.
It does not run any analysis and is only meant to be used as a dependency for the sub-analyzers of 
ErrStack to share the same configurations. 
`

var Analyzer = &analysis.Analyzer{
	Name:       "errstack_config",
	Doc:        _doc,
	Run:        helpers.WrapRun(run),
	Flags:      newFlagSet(),
	ResultType: reflect.TypeOf((*helpers.Result[*Config])(nil)),
}

const (
	// YamlConfig is the flag for loading all packages.
	YamlConfig = "yaml-config"
	// Debug is the flag for debug logging.
	Debug = "debug"
)

// newFlagSet returns a flag set to be used in the nilaway config analyzer.
func newFlagSet() flag.FlagSet {
	fs := flag.NewFlagSet("errstack_config", flag.ExitOnError)

	// We do not keep the returned pointer to the flags because we will not use them directly here.
	// Instead, we will use the flags through the analyzer's Flags field later.
	_ = fs.String(YamlConfig, "", "Full config in yaml format")

	_ = fs.Bool(Debug, false, "Debug logging")

	return *fs
}

func run(pass *analysis.Pass) (*Config, error) {
	// Set up default values for the config.
	conf := NewDefaultConfig()
	defer func() {
		wd, err := os.Getwd()
		if err != nil {
			panic(err)
		}
		goroot := os.Getenv("GOROOT")
		conf.WorkDir = wd + "/"
		conf.GoRoot = goroot + "/src/"
	}()

	if yamlConfig, ok := pass.Analyzer.Flags.Lookup(YamlConfig).Value.(flag.Getter).Get().(string); ok {
		if len(yamlConfig) > 0 {
			err := yaml.Unmarshal([]byte(yamlConfig), conf)
			if err != nil {
				return nil, fmt.Errorf("unmarshal config: %w", err)
			}
			return conf, nil
		}
	}

	// Override default values if the user provides flags.
	if debug, ok := pass.Analyzer.Flags.Lookup(Debug).Value.(flag.Getter).Get().(bool); ok {
		conf.Debug = debug
	}
	log.EnableDebug(conf.Debug)

	return conf, nil
}
