package core

import (
	"context"
	"log/slog"
	"os"
)

// Logger provides enhanced logging capabilities for The Ark
type Logger struct {
	*slog.Logger
	features map[string]*slog.Logger
}

// NewLogger creates a new logger instance
func NewLogger() *Logger {
	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})

	logger := &Logger{
		Logger:   slog.New(handler),
		features: make(map[string]*slog.Logger),
	}

	return logger
}

// ForFeature returns a logger specific to a feature
func (l *Logger) ForFeature(featureName string) *Logger {
	if featureLogger, exists := l.features[featureName]; exists {
		return &Logger{
			Logger:   featureLogger,
			features: l.features,
		}
	}

	// Create feature-specific logger with feature name in context
	featureLogger := l.Logger.With("feature", featureName)
	l.features[featureName] = featureLogger

	return &Logger{
		Logger:   featureLogger,
		features: l.features,
	}
}

// WithContext returns a logger with request context
func (l *Logger) WithContext(ctx context.Context) *Logger {
	if ctx == nil {
		return l
	}

	// Extract request ID from context if available
	if requestID, ok := ctx.Value("request_id").(string); ok {
		return &Logger{
			Logger:   l.Logger.With("request_id", requestID),
			features: l.features,
		}
	}

	return l
}

// WithUser returns a logger with user context
func (l *Logger) WithUser(userID int, email string) *Logger {
	return &Logger{
		Logger:   l.Logger.With("user_id", userID, "user_email", email),
		features: l.features,
	}
}

// SetLevel sets the logging level
func (l *Logger) SetLevel(level slog.Level) {
	// This would require recreating the handler, which is more complex
	// For now, we'll use the default level
}

// LogFeatureEvent logs a feature-specific event
func (l *Logger) LogFeatureEvent(featureName, event string, attrs ...any) {
	featureLogger := l.ForFeature(featureName)
	featureLogger.Info("Feature event", append([]any{"event", event}, attrs...)...)
}

// LogFeatureError logs a feature-specific error
func (l *Logger) LogFeatureError(featureName, message string, err error, attrs ...any) {
	featureLogger := l.ForFeature(featureName)
	allAttrs := append([]any{"error", err}, attrs...)
	featureLogger.Error(message, allAttrs...)
}
