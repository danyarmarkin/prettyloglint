package simple

import (
	"log/slog"
)

func GoodExamples() {
	slog.Info("starting server on port 8080")
	slog.Error("failed to connect to database")
	slog.Warn("something went wrong")
	slog.Debug("api request completed")

	logger := slog.New(nil)
	logger.Info("starting instance")
	logger.Debug("operation completed")
}
