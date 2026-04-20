package logger

import (
	"context"
	"errors"
	"os"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	gormlogger "gorm.io/gorm/logger"
	"gorm.io/gorm/utils"
)

type GormZapLogger struct {
	ZapLogger                 *zap.Logger
	LogLevel                  gormlogger.LogLevel
	SlowThreshold             time.Duration
	SkipCallerLookup          bool
	IgnoreRecordNotFoundError bool
}

func LoggerCustom(env string) *zap.Logger {

	var (
		logger *zap.Logger
		err    error
	)

	if env == "production" {
		logger, err = zap.NewProduction()
		if err != nil {
			panic("Failed to initialize logger!")
		}
	} else {
		logger, err = zap.NewDevelopment()
		if err != nil {
			panic("Failed to initialize logger!")
		}
	}
	return logger
}

func ProductionLog() (*zap.Logger, error) {
	cfg := zap.NewProductionEncoderConfig()
	cfg.TimeKey = "timestamp"
	cfg.EncodeTime = zapcore.ISO8601TimeEncoder

	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(cfg),
		zapcore.AddSync(os.Stdout),
		zapcore.InfoLevel,
	)
	return zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel)), nil
}

func DevelopmentLog() (*zap.Logger, error) {
	cfg := zap.NewDevelopmentEncoderConfig()
	cfg.EncodeLevel = zapcore.CapitalColorLevelEncoder

	core := zapcore.NewCore(
		zapcore.NewConsoleEncoder(cfg),
		zapcore.AddSync(os.Stdout),
		zapcore.DebugLevel,
	)
	return zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel)), nil
}

func NewGormZapLogger(zapLog *zap.Logger) *GormZapLogger {
	return &GormZapLogger{
		ZapLogger:                 zapLog,
		LogLevel:                  gormlogger.Info,
		SlowThreshold:             200 * time.Millisecond,
		SkipCallerLookup:          false,
		IgnoreRecordNotFoundError: true,
	}
}

func (l *GormZapLogger) LogMode(level gormlogger.LogLevel) gormlogger.Interface {
	newLogger := *l
	newLogger.LogLevel = level
	return &newLogger
}

func (l *GormZapLogger) Info(ctx context.Context, msg string, args ...interface{}) {
	if l.LogLevel >= gormlogger.Info {
		l.ZapLogger.Sugar().Infof(msg, args...)
	}
}

func (l *GormZapLogger) Warn(ctx context.Context, msg string, args ...interface{}) {
	if l.LogLevel >= gormlogger.Warn {
		l.ZapLogger.Sugar().Warnf(msg, args...)
	}
}

func (l *GormZapLogger) Error(ctx context.Context, msg string, args ...interface{}) {
	if l.LogLevel >= gormlogger.Error {
		l.ZapLogger.Sugar().Errorf(msg, args...)
	}
}

func (l *GormZapLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	if l.LogLevel <= gormlogger.Silent {
		return
	}

	elapsed := time.Since(begin)
	sql, rows := fc()

	fields := []zap.Field{
		zap.String("sql", sql),
		zap.Int64("rows", rows),
		zap.Duration("elapsed", elapsed),
		zap.String("caller", utils.FileWithLineNum()),
	}

	switch {
	case err != nil && l.LogLevel >= gormlogger.Error:
		if !(errors.Is(err, gormlogger.ErrRecordNotFound) && l.IgnoreRecordNotFoundError) {
			l.ZapLogger.Error("query error",
				append(fields, zap.Error(err))...,
			)
		}

	case l.SlowThreshold != 0 && elapsed > l.SlowThreshold && l.LogLevel >= gormlogger.Warn:
		l.ZapLogger.Warn("slow query",
			append(fields, zap.Duration("threshold", l.SlowThreshold))...,
		)

	case l.LogLevel >= gormlogger.Info:
		l.ZapLogger.Debug("query", fields...)
	}
}
