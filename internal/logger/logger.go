package logger

import (
    "os"
    "strings"

    "github.com/sirupsen/logrus"
)

var log *logrus.Logger

// Init initializes the logger
func Init() {
    log = logrus.New()
    
    // Set output to stdout
    log.SetOutput(os.Stdout)
    
    // Set JSON format for structured logging
    log.SetFormatter(&logrus.JSONFormatter{
        TimestampFormat: "2006-01-02T15:04:05.000Z",
        FieldMap: logrus.FieldMap{
            logrus.FieldKeyTime:  "timestamp",
            logrus.FieldKeyLevel: "level",
            logrus.FieldKeyMsg:   "message",
        },
    })
    
    // Set log level from environment variable
    logLevel := os.Getenv("LOG_LEVEL")
    if logLevel == "" {
        logLevel = "info"
    }
    
    SetLevel(logLevel)
}

// SetLevel sets the log level
func SetLevel(level string) error {
    switch strings.ToLower(level) {
    case "debug":
        log.SetLevel(logrus.DebugLevel)
    case "info":
        log.SetLevel(logrus.InfoLevel)
    case "warn", "warning":
        log.SetLevel(logrus.WarnLevel)
    case "error":
        log.SetLevel(logrus.ErrorLevel)
    case "fatal":
        log.SetLevel(logrus.FatalLevel)
    case "panic":
        log.SetLevel(logrus.PanicLevel)
    default:
        log.SetLevel(logrus.InfoLevel)
        log.WithFields(logrus.Fields{
            "invalid_level": level,
            "message": "Invalid log level, using info",
            "valid_levels": "debug, info, warn, error, fatal, panic",
        }).Warn("Invalid log level provided")
        return nil
    }
    
    log.WithField("log_level", level).Info("Log level set")
    return nil
}

// GetLogger returns the logger instance
func GetLogger() *logrus.Logger {
    if log == nil {
        Init()
    }
    return log
}

// WithField creates an entry with a field
func WithField(key string, value interface{}) *logrus.Entry {
    return GetLogger().WithField(key, value)
}

// WithFields creates an entry with multiple fields
func WithFields(fields map[string]interface{}) *logrus.Entry {
    return GetLogger().WithFields(fields)
}

// WithError creates an entry with an error field
func WithError(err error) *logrus.Entry {
    return GetLogger().WithError(err)
}

// Debug logs a debug message
func Debug(args ...interface{}) {
    GetLogger().Debug(args...)
}

// Info logs an info message
func Info(args ...interface{}) {
    GetLogger().Info(args...)
}

// Warn logs a warning message
func Warn(args ...interface{}) {
    GetLogger().Warn(args...)
}

// Error logs an error message
func Error(args ...interface{}) {
    GetLogger().Error(args...)
}

// Fatal logs a fatal message and exits
func Fatal(args ...interface{}) {
    GetLogger().Fatal(args...)
}

// Debugf logs a formatted debug message
func Debugf(format string, args ...interface{}) {
    GetLogger().Debugf(format, args...)
}

// Infof logs a formatted info message
func Infof(format string, args ...interface{}) {
    GetLogger().Infof(format, args...)
}

// Warnf logs a formatted warning message
func Warnf(format string, args ...interface{}) {
    GetLogger().Warnf(format, args...)
}

// Errorf logs a formatted error message
func Errorf(format string, args ...interface{}) {
    GetLogger().Errorf(format, args...)
}

// Fatalf logs a formatted fatal message and exits
func Fatalf(format string, args ...interface{}) {
    GetLogger().Fatalf(format, args...)
}
