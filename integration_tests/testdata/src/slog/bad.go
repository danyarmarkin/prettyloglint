package simple

import (
	"log/slog"
)

var (
	password = "secret"
	apiKey   = "abcd"
)

func BadExamples() {
	slog.Info("Starting server on port 8080") // want "log message should start"
	slog.Error("запуск сервера")              // want "should contain only English"
	slog.Warn("server started! 🚀")            // want "contains disallowed symbol or emoji"
	slog.Info("user password: " + password)   // want "may contain sensitive data"

	logger := slog.New(nil)
	logger.Info("Starting instance")  // want "log message should start"
	logger.Debug("api_key=" + apiKey) // want "may contain sensitive data"
	logger.Info("ok!")                // want "contains disallowed symbol or emoji"
}
