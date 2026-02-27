package simple

import (
	"log/slog"

	"go.uber.org/zap"
)

func GoodExamples() {
	slog.Info("starting server on port 8080")
	slog.Error("failed to connect to database")
	slog.Warn("something went wrong")

	logger, _ := zap.NewProduction()
	defer logger.Sync()
	logger.Info("server started")
	logger.Debug("api request completed")
}
