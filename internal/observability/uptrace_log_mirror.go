package observability

import (
	"context"
	"fmt"
	"math"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/riskibarqy/fantasy-league/internal/platform/logging"
	otellog "go.opentelemetry.io/otel/log"
	otelglobal "go.opentelemetry.io/otel/log/global"
	"go.uber.org/zap/zapcore"
)

const (
	uptraceLogInstrumentation = "fantasy-league/internal/platform/logging"
	healthPath                = "/healthz"
	maxLogValueDepth          = 3
)

func newUptraceLogMirror(serviceVersion string) logging.MirrorFunc {
	otelLogger := otelglobal.Logger(
		uptraceLogInstrumentation,
		otellog.WithInstrumentationVersion(serviceVersion),
	)

	return func(ctx context.Context, level logging.Level, msg string, args ...any) {
		if shouldSkipUptraceLog(msg, args) {
			return
		}

		if ctx == nil {
			ctx = context.Background()
		}
		severity := toOTelSeverity(level)
		if !otelLogger.Enabled(ctx, otellog.EnabledParameters{
			Severity:  severity,
			EventName: msg,
		}) {
			return
		}

		now := time.Now().UTC()
		record := otellog.Record{}
		record.SetTimestamp(now)
		record.SetObservedTimestamp(now)
		record.SetSeverity(severity)
		record.SetSeverityText(strings.ToUpper(level.String()))
		record.SetEventName(msg)
		record.SetBody(otellog.StringValue(msg))

		attributes := buildOTelLogAttributes(args)
		if len(attributes) > 0 {
			record.AddAttributes(attributes...)
		}

		otelLogger.Emit(ctx, record)
	}
}

func shouldSkipUptraceLog(msg string, args []any) bool {
	if msg != "http_request" {
		return false
	}
	for i := 0; i+1 < len(args); i += 2 {
		key, ok := args[i].(string)
		if !ok || key != "http_path" {
			continue
		}
		path, ok := args[i+1].(string)
		return ok && path == healthPath
	}
	return false
}

func buildOTelLogAttributes(args []any) []otellog.KeyValue {
	if len(args) == 0 {
		return nil
	}

	attrs := make([]otellog.KeyValue, 0, (len(args)+1)/2)
	for i := 0; i < len(args); i += 2 {
		key := fmt.Sprintf("arg_%d", i/2)
		if k, ok := args[i].(string); ok && strings.TrimSpace(k) != "" {
			key = k
		}
		if i+1 >= len(args) {
			attrs = append(attrs, otellog.Empty(key))
			continue
		}
		attrs = append(attrs, otellog.KeyValue{
			Key:   key,
			Value: toOTelLogValue(args[i+1], 0),
		})
	}

	return attrs
}

func toOTelSeverity(level zapcore.Level) otellog.Severity {
	switch {
	case level <= zapcore.DebugLevel:
		return otellog.SeverityDebug
	case level == zapcore.InfoLevel:
		return otellog.SeverityInfo
	case level == zapcore.WarnLevel:
		return otellog.SeverityWarn
	case level >= zapcore.DPanicLevel:
		return otellog.SeverityFatal
	default:
		return otellog.SeverityError
	}
}

func toOTelLogValue(value any, depth int) otellog.Value {
	if depth >= maxLogValueDepth {
		return otellog.StringValue(fmt.Sprint(value))
	}
	if value == nil {
		return otellog.Value{}
	}

	switch v := value.(type) {
	case string:
		return otellog.StringValue(v)
	case bool:
		return otellog.BoolValue(v)
	case int:
		return otellog.IntValue(v)
	case int8:
		return otellog.Int64Value(int64(v))
	case int16:
		return otellog.Int64Value(int64(v))
	case int32:
		return otellog.Int64Value(int64(v))
	case int64:
		return otellog.Int64Value(v)
	case uint:
		if uint64(v) > math.MaxInt64 {
			return otellog.StringValue(fmt.Sprint(v))
		}
		return otellog.Int64Value(int64(v))
	case uint8:
		return otellog.Int64Value(int64(v))
	case uint16:
		return otellog.Int64Value(int64(v))
	case uint32:
		return otellog.Int64Value(int64(v))
	case uint64:
		if v > math.MaxInt64 {
			return otellog.StringValue(fmt.Sprint(v))
		}
		return otellog.Int64Value(int64(v))
	case float32:
		return otellog.Float64Value(float64(v))
	case float64:
		return otellog.Float64Value(v)
	case []byte:
		cp := append([]byte(nil), v...)
		return otellog.BytesValue(cp)
	case time.Time:
		return otellog.StringValue(v.UTC().Format(time.RFC3339Nano))
	case time.Duration:
		return otellog.StringValue(v.String())
	case error:
		return otellog.StringValue(v.Error())
	case fmt.Stringer:
		return otellog.StringValue(v.String())
	}

	rv := reflect.ValueOf(value)
	switch rv.Kind() {
	case reflect.Pointer, reflect.Interface:
		if rv.IsNil() {
			return otellog.Value{}
		}
		return toOTelLogValue(rv.Elem().Interface(), depth+1)
	case reflect.Slice, reflect.Array:
		if rv.Kind() == reflect.Slice && rv.Type().Elem().Kind() == reflect.Uint8 {
			out := make([]byte, rv.Len())
			reflect.Copy(reflect.ValueOf(out), rv)
			return otellog.BytesValue(out)
		}
		items := make([]otellog.Value, 0, rv.Len())
		for i := 0; i < rv.Len(); i++ {
			items = append(items, toOTelLogValue(rv.Index(i).Interface(), depth+1))
		}
		return otellog.SliceValue(items...)
	case reflect.Map:
		if rv.Type().Key().Kind() != reflect.String {
			return otellog.StringValue(fmt.Sprint(value))
		}
		keys := rv.MapKeys()
		sort.Slice(keys, func(i, j int) bool {
			return keys[i].String() < keys[j].String()
		})
		kvs := make([]otellog.KeyValue, 0, len(keys))
		for _, key := range keys {
			kvs = append(kvs, otellog.KeyValue{
				Key:   key.String(),
				Value: toOTelLogValue(rv.MapIndex(key).Interface(), depth+1),
			})
		}
		return otellog.MapValue(kvs...)
	default:
		return otellog.StringValue(fmt.Sprint(value))
	}
}
