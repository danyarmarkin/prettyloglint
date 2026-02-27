package main

import (
	"prettyloglint/internal/analyzer"

	"go.uber.org/zap"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/singlechecker"
)

func main() {
	_, _ = zap.NewDevelopment()
	singlechecker.Main(analyzer.Analyzer)
}

func AnalyzerPlugin() map[string]*analysis.Analyzer {
	return map[string]*analysis.Analyzer{
		analyzer.Analyzer.Name: analyzer.Analyzer,
	}
}
