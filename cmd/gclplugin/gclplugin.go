// Package gclplugin implements the golangci-lint's module plugin interface for ErrStack to be used
// as a private linter in golangci-lint. See more details at
// https://golangci-lint.run/plugins/module-plugins/.
package gclplugin

import (
	"fmt"

	"github.com/AdamBrianBright/errstack/internal/config"
	"github.com/AdamBrianBright/errstack/internal/passes/errstack"

	"github.com/golangci/plugin-module-register/register"
	"golang.org/x/tools/go/analysis"
	"gopkg.in/yaml.v3"
)

func init() {
	// Регистрируем кастомный линтер в реестре плагинов golangci-lint.
	register.Plugin("errstack", New)
}

// New returns the golangci-lint plugin that wraps the ErrStack analyzer.
func New(settings any) (register.LinterPlugin, error) {
	conf, err := register.DecodeSettings[*config.Config](settings)
	if err != nil {
		return nil, err
	}

	return &ErrStackPlugin{conf: conf}, nil
}

// ErrStackPlugin is the ErrStack plugin wrapper for golangci-lint.
type ErrStackPlugin struct {
	conf *config.Config
}

// BuildAnalyzers builds the ErrStack analyzer with the configurations applied to the config analyzer.
func (p *ErrStackPlugin) BuildAnalyzers() ([]*analysis.Analyzer, error) {
	conf, err := yaml.Marshal(p.conf)
	if err != nil {
		return nil, fmt.Errorf("marshal config: %w", err)
	}
	err = config.Analyzer.Flags.Set(config.YamlConfig, string(conf))
	if err != nil {
		return nil, fmt.Errorf("set config flag: %w", err)
	}

	return []*analysis.Analyzer{errstack.Analyzer}, nil
}

// GetLoadMode returns the load mode of the ErrStack plugin (requiring types info).
func (p *ErrStackPlugin) GetLoadMode() string {
	return register.LoadModeTypesInfo
}
