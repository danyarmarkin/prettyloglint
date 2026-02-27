package zap

// Minimal stub of go.uber.org/zap used only for analysistest.
// Keeps signatures needed by testdata/simple/bad.go so type-based detection works.

type Logger struct{}

type Field struct{}

func NewProduction() (*Logger, error) {
	return &Logger{}, nil
}

func (l *Logger) Sync() error { return nil }

// Logger methods
func (l *Logger) Info(msg string, args ...interface{})  {}
func (l *Logger) Debug(msg string, args ...interface{}) {}
func (l *Logger) Error(msg string, args ...interface{}) {}
func (l *Logger) Warn(msg string, args ...interface{})  {}

// Field constructors used in tests
func String(key, val string) Field          { return Field{} }
func Int64(key string, val int64) Field     { return Field{} }
func Any(key string, val interface{}) Field { return Field{} }
