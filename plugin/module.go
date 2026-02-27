package plugin

import (
	"github.com/danyarmarkin/prettyloglint/internal/analyzer"
	"github.com/golangci/plugin-module-register/register"

	"golang.org/x/tools/go/analysis"
)

func init() {
	register.Plugin("prettyloglint", New)
}

type analyzerPlugin struct{}

func (p *analyzerPlugin) BuildAnalyzers() ([]*analysis.Analyzer, error) {
	return []*analysis.Analyzer{
		analyzer.Analyzer,
	}, nil
}

func (p *analyzerPlugin) GetLoadMode() string {
	return register.LoadModeSyntax
}

func New(conf any) (register.LinterPlugin, error) {
	return &analyzerPlugin{}, nil
}
