package simple

import "go.uber.org/zap"

func GoodExamples() {
	logger, _ := zap.NewProduction()
	defer func() { _ = logger.Sync() }()

	logger.Info("starting service on port 8080")
	logger.Error("failed to connect to database")
	logger.Debug("operation completed successfully")
	logger.Warn("something went wrong")

	logger.Info("user info", zap.String("user", "alice"))
	logger.Info("metrics", zap.Int64("count", 42), zap.String("env", "prod"))
	logger.Debug("payload", zap.Any("data", map[string]interface{}{"ok": true}))

	logger.Info("batch processed", zap.String("batch_id", "b1"), zap.Int64("items", 100), zap.String("source", "svc"))
	
	logger.Info("misc", zap.String("note", "all good"), zap.Int64("duration_ms", 120))
}
