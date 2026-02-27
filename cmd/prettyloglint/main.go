package main

import (
	"github.com/danyarmarkin/prettyloglint/internal/analyzer"

	"golang.org/x/tools/go/analysis/singlechecker"
)

func main() {
	singlechecker.Main(analyzer.Analyzer)
}
