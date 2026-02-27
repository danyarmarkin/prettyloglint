package plugin

import (
	"github.com/danyarmarkin/prettyloglint/internal/analyzer"
	"github.com/golangci/plugin-module-register/register"

	"golang.org/x/tools/go/analysis"
)

func init() {
	register.Plugin("prettyloglint", New)
}

type analyzerPlugin struct {
	cfg analyzer.Config
}

func (p *analyzerPlugin) BuildAnalyzers() ([]*analysis.Analyzer, error) {
	return []*analysis.Analyzer{
		analyzer.NewAnalyzer(p.cfg),
	}, nil
}

func (p *analyzerPlugin) GetLoadMode() string {
	return register.LoadModeSyntax
}

func New(conf any) (register.LinterPlugin, error) {
	cfg := analyzer.Config{
		AllowedPunctuation:      ",-/:()",
		CustomSensitivePatterns: []string{},
		IgnoreZapFields:         false,
	}
	if confMap, ok := conf.(map[string]interface{}); ok {
		if ap, ok := confMap["allowed-punctuation"].(string); ok {
			cfg.AllowedPunctuation = ap
		}
		if csp, ok := confMap["custom-sensitive-patterns"].([]interface{}); ok {
			for _, v := range csp {
				if s, ok := v.(string); ok {
					cfg.CustomSensitivePatterns = append(cfg.CustomSensitivePatterns, s)
				}
			}
		}
		if izf, ok := confMap["ignore-zap-fields"].(bool); ok {
			cfg.IgnoreZapFields = izf
		}
	}
	return &analyzerPlugin{cfg: cfg}, nil
}
