package simple

import "go.uber.org/zap"

var (
	password = "secret"
	apiKey   = "abcd"
	token    = "tok"
)

func BadExamples() {
	logger, _ := zap.NewProduction()
	defer func() { _ = logger.Sync() }()

	logger.Info("Failed to start service") // want "log message should start"

	logger.Error("ошибка подключения") // want "should contain only English"

	logger.Debug("service started 🚀")   // want "contains disallowed symbol or emoji"
	logger.Debug("service started! 🚀")  // want "contains disallowed symbol or emoji"
	logger.Warn("connection failed!!!") // want "contains disallowed symbol or emoji"

	logger.Info("user password: " + password) // want "may contain sensitive data"
	logger.Info("token: " + token)            // want "may contain sensitive data"

	logger.Info("hello", zap.String("token", "12345"))  // want "may contain sensitive data"
	logger.Info("hello", zap.Int64("token", 12345))     // want "may contain sensitive data"
	logger.Debug("debug", zap.Any("password", "12345")) // want "may contain sensitive data"
	logger.Debug("debug", zap.Any("password", 12345))   // want "may contain sensitive data"

	logger.Info("info", zap.String("user", "bob"), zap.String("api_key", apiKey), zap.Int64("count", 1)) // want "may contain sensitive data"
}
