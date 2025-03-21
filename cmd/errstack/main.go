package main

import (
	"errors"
	"log"

	"github.com/AdamBrianBright/errstack/errstack"

	"github.com/spf13/viper"
	"golang.org/x/tools/go/analysis/singlechecker"
)

func main() {
	viper.SetConfigName(".errstack")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("$HOME/.errstack")
	viper.AddConfigPath(".")

	viper.SetDefault("wrapperFunctions", errstack.DefaultWrapperFunctions)
	viper.SetDefault("cleanFunctions", errstack.DefaultCleanFunctions)

	// Read in config, ignore if the file isn't found and use defaults.
	if err := viper.ReadInConfig(); err != nil {
		var configFileNotFoundError viper.ConfigFileNotFoundError
		if !errors.As(err, &configFileNotFoundError) {
			log.Fatalf("failed to parse config: %v", err)
		}
	}

	var cfg errstack.Config
	if err := viper.Unmarshal(&cfg); err != nil {
		log.Fatalf("failed to unmarshal config: %v", err)
	}

	singlechecker.Main(errstack.NewAnalyzer(cfg))
}
