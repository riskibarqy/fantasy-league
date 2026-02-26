package logging

import (
	"context"
	"os"
	"sync/atomic"

	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Level = zapcore.Level

const (
	LevelDebug = zapcore.DebugLevel
	LevelInfo  = zapcore.InfoLevel
	LevelWarn  = zapcore.WarnLevel
	LevelError = zapcore.ErrorLevel
)

type Logger struct {
	zap    *zap.Logger
	sugar  *zap.SugaredLogger
	closed atomic.Bool
}

var defaultLogger atomic.Pointer[Logger]

func init() {
	defaultLogger.Store(NewNop())
}

func NewJSON(level Level) *Logger {
	encoderCfg := zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		FunctionKey:    zapcore.OmitKey,
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.CapitalLevelEncoder,
		EncodeTime:     zapcore.RFC3339NanoTimeEncoder,
		EncodeDuration: zapcore.StringDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderCfg),
		zapcore.Lock(os.Stdout),
		level,
	)

	return FromZap(zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel)))
}

func NewNop() *Logger {
	return FromZap(zap.NewNop())
}

func FromZap(z *zap.Logger) *Logger {
	if z == nil {
		z = zap.NewNop()
	}
	return &Logger{
		zap:   z,
		sugar: z.Sugar(),
	}
}

func Default() *Logger {
	if logger := defaultLogger.Load(); logger != nil {
		return logger
	}
	return NewNop()
}

func SetDefault(logger *Logger) {
	if logger == nil {
		logger = NewNop()
	}
	defaultLogger.Store(logger)
}

func (l *Logger) Zap() *zap.Logger {
	if l == nil || l.zap == nil {
		return zap.NewNop()
	}
	return l.zap
}

func (l *Logger) Sync() error {
	if l == nil || l.zap == nil {
		return nil
	}
	if l.closed.CompareAndSwap(false, true) {
		return l.zap.Sync()
	}
	return nil
}

func (l *Logger) With(args ...any) *Logger {
	if l == nil {
		return NewNop()
	}
	return &Logger{
		zap:   l.zap.With(zapFields(args)...),
		sugar: l.sugar.With(args...),
	}
}

func (l *Logger) Debug(msg string, args ...any) {
	l.log(zap.DebugLevel, msg, args...)
}

func (l *Logger) Info(msg string, args ...any) {
	l.log(zap.InfoLevel, msg, args...)
}

func (l *Logger) Warn(msg string, args ...any) {
	l.log(zap.WarnLevel, msg, args...)
}

func (l *Logger) Error(msg string, args ...any) {
	l.log(zap.ErrorLevel, msg, args...)
}

func (l *Logger) DebugContext(ctx context.Context, msg string, args ...any) {
	l.logContext(ctx, zap.DebugLevel, msg, args...)
}

func (l *Logger) InfoContext(ctx context.Context, msg string, args ...any) {
	l.logContext(ctx, zap.InfoLevel, msg, args...)
}

func (l *Logger) WarnContext(ctx context.Context, msg string, args ...any) {
	l.logContext(ctx, zap.WarnLevel, msg, args...)
}

func (l *Logger) ErrorContext(ctx context.Context, msg string, args ...any) {
	l.logContext(ctx, zap.ErrorLevel, msg, args...)
}

func (l *Logger) log(level zapcore.Level, msg string, args ...any) {
	logger := l
	if logger == nil {
		logger = Default()
	}
	fields := zapFields(args)
	if ce := logger.zap.Check(level, msg); ce != nil {
		ce.Write(fields...)
	}
}

func (l *Logger) logContext(ctx context.Context, level zapcore.Level, msg string, args ...any) {
	logger := l
	if logger == nil {
		logger = Default()
	}
	fields := make([]zap.Field, 0, len(args)/2+2)
	fields = append(fields, zapFields(args)...)
	fields = append(fields, traceFields(ctx)...)
	if ce := logger.zap.Check(level, msg); ce != nil {
		ce.Write(fields...)
	}
}

func traceFields(ctx context.Context) []zap.Field {
	if ctx == nil {
		return nil
	}
	spanCtx := trace.SpanContextFromContext(ctx)
	if !spanCtx.IsValid() {
		return nil
	}
	return []zap.Field{
		zap.String("trace_id", spanCtx.TraceID().String()),
		zap.String("span_id", spanCtx.SpanID().String()),
	}
}

func zapFields(args []any) []zap.Field {
	if len(args) == 0 {
		return nil
	}

	out := make([]zap.Field, 0, (len(args)+1)/2)
	for i := 0; i < len(args); i += 2 {
		key, ok := args[i].(string)
		if !ok || key == "" {
			key = "arg"
		}

		if i+1 >= len(args) {
			out = append(out, zap.Any(key, nil))
			break
		}

		value := args[i+1]
		if err, ok := value.(error); ok {
			out = append(out, zap.NamedError(key, err))
			continue
		}
		out = append(out, zap.Any(key, value))
	}

	return out
}

