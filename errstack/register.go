package errstack

import (
	"github.com/golangci/plugin-module-register/register"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
)

func init() {
	// Регистрируем кастомный линтер в реестре плагинов golangci-lint.
	register.Plugin("errstack", New)
}

type Plugin struct {
	settings Config
}

func New(settings any) (register.LinterPlugin, error) {
	s, err := register.DecodeSettings[Config](settings)
	if err != nil {
		return nil, err
	}

	return &Plugin{settings: s}, nil
}

func (p *Plugin) GetLoadMode() string {
	return register.LoadModeTypesInfo
}

func (p *Plugin) BuildAnalyzers() ([]*analysis.Analyzer, error) {
	return []*analysis.Analyzer{
		NewAnalyzer(p.settings),
	}, nil
}

func NewAnalyzer(config Config) *analysis.Analyzer {
	return &analysis.Analyzer{
		Name:             "errstack",
		Doc:              "Checks for unnecessary error wrapping using errors.Wrap, errors.Wrapf, and errors.WithStack",
		Run:              NewErrStack(config).Run,
		Requires:         []*analysis.Analyzer{inspect.Analyzer},
		RunDespiteErrors: true,
	}
}
