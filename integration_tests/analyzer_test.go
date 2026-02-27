package integration_tests

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"prettyloglint/internal/analyzer"
)

func TestAnalyzerSimple(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, analyzer.Analyzer, "simple")
}

func TestAnalyzerSlog(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, analyzer.Analyzer, "slog")
}

func TestAnalyzerZap(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, analyzer.Analyzer, "zap")
}
