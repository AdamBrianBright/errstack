package main

import (
	"errors"
	"log"

	"github.com/AdamBrianBright/errstack/internal/config"
	"github.com/AdamBrianBright/errstack/internal/passes/errstack"

	"github.com/spf13/viper"
	"golang.org/x/tools/go/analysis/singlechecker"
	"gopkg.in/yaml.v3"
)

func main() {
	viper.SetConfigName(".errstack")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("$HOME/.errstack")
	viper.AddConfigPath(".")

	viper.SetDefault("wrapperFunctions", config.DefaultWrapperFunctions)
	viper.SetDefault("cleanFunctions", config.DefaultCleanFunctions)

	// Read in config, ignore if the file isn't found and use defaults.
	if err := viper.ReadInConfig(); err != nil {
		var configFileNotFoundError viper.ConfigFileNotFoundError
		if !errors.As(err, &configFileNotFoundError) {
			log.Fatalf("failed to parse config: %v", err)
		}
	}

	var cfg config.Config
	if err := viper.Unmarshal(&cfg); err != nil {
		log.Fatalf("failed to unmarshal config: %v", err)
	}
	configYaml, err := yaml.Marshal(cfg)
	if err != nil {
		log.Fatalf("failed to marshal config: %v", err)
	}
	err = config.Analyzer.Flags.Set(config.YamlConfig, string(configYaml))
	if err != nil {
		log.Fatalf("failed to set config flag: %v", err)
	}

	singlechecker.Main(errstack.Analyzer)
}
