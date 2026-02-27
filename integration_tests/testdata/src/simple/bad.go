package simple

import (
	"log/slog"

	"go.uber.org/zap"
)

var (
	password = "secret"
	apiKey   = "abcd"
)

func BadExamples() {
	slog.Info("Starting server on port 8080") // want "log message should start"

	slog.Error("запуск сервера") // want "should contain only English"

	slog.Warn("server started! 🚀")     // want "contains disallowed symbol or emoji"
	slog.Error("connection failed!!!") // want "contains disallowed symbol or emoji"

	slog.Info("user password: " + password) // want "may contain sensitive data"
	slog.Debug("api_key=" + apiKey)         // want "may contain sensitive data"

	logger, _ := zap.NewProduction()
	defer logger.Sync()
	logger.Info("Failed to connect to DB") // want "log message should start"
	logger.Info("token: 12345")            // want "may contain sensitive data"

	logger.Info("hello there", zap.String("token", "12345")) // want "may contain sensitive data"
	logger.Info("hello there", zap.Int64("token", 12345))    // want "may contain sensitive data"
	logger.Debug("debug", zap.Any("password", "12345"))      // want "may contain sensitive data"
	logger.Debug("debug", zap.Any("password", 12345))        // want "may contain sensitive data"
}
