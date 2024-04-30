package mlog

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/go-logr/logr"
	"github.com/go-logr/zapr"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/duration"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/klog/v2"
	"k8s.io/klog/v2/textlogger"
)

func newLogr(ctx context.Context, encoding string, klogLevel klog.Level) (logr.Logger, func(), error) {
	if encoding == "text" {
		var w io.Writer = os.Stderr
		flush := func() { _ = os.Stderr.Sync() }

		// allow tests to override klog config (but cheat and re-use the zap override key)
		if overrides, ok := ctx.Value(zapOverridesKey).(*testOverrides); ok {
			if overrides.w != nil {
				w = newSink(overrides.w) // make sure the value is safe for concurrent use
				flush = func() {}
			}
		}

		w = &trimWriter{w: w}

		return textlogger.NewLogger(textlogger.NewConfig(textlogger.Verbosity(int(klogLevel)), textlogger.Output(w))), flush, nil
	}

	path := "stderr" // this is how zap refers to os.Stderr
	f := func(config *zap.Config) {
		if encoding == "console" {
			config.EncoderConfig.LevelKey = zapcore.OmitKey
			config.EncoderConfig.EncodeCaller = zapcore.ShortCallerEncoder
			config.EncoderConfig.EncodeTime = humanTimeEncoder
			config.EncoderConfig.EncodeDuration = humanDurationEncoder
		}
	}
	var opts []zap.Option

	// allow tests to override zap config
	if overrides, ok := ctx.Value(zapOverridesKey).(*testOverrides); ok {
		if overrides.w != nil {
			// use a per invocation random string as the key into the global map
			testKey := "/" + base64.RawURLEncoding.EncodeToString([]byte(rand.String(32)))

			// tell zap to use our custom sink registry to find the writer
			path = "monis.app-mlog://" + testKey

			// the registry may be called multiple times so make sure the value is safe for concurrent use
			sink := newSink(overrides.w)

			// store the test's buffer where we can find it globally
			actual, loaded := sinkMap.LoadOrStore(testKey, sink)
			require.False(overrides.t, loaded)
			require.Equal(overrides.t, sink, actual)

			defer func() {
				// delete buffer from the global map to prevent a memory leak
				value, loaded := sinkMap.LoadAndDelete(testKey)
				require.True(overrides.t, loaded)
				require.Equal(overrides.t, sink, value)
			}()
		}
		if overrides.f != nil {
			f = overrides.f
		}
		if overrides.opts != nil {
			opts = overrides.opts
		}
	}

	// when using the trace or all log levels, an error log will contain the full stack.
	// this is too noisy for regular use because things like leader election conflicts
	// result in transient errors and we do not want all of that noise in the logs.
	// this check is performed dynamically on the global log level.
	return newZapr(globalLevel, LevelTrace, encoding, path, f, opts...)
}

func newZapr(level zap.AtomicLevel, addStack zapcore.LevelEnabler, encoding, path string, f func(config *zap.Config), opts ...zap.Option) (logr.Logger, func(), error) {
	opts = append([]zap.Option{zap.WrapCore(func(core zapcore.Core) zapcore.Core {
		return &trimCore{core: core}
	})}, opts...)

	if encoding == "json" { // stack traces are too noisy otherwise
		opts = append([]zap.Option{zap.AddStacktrace(addStack)}, opts...)
	}

	config := zap.Config{
		Level:             level,
		Development:       false,
		DisableCaller:     false,
		DisableStacktrace: true, // handled via the AddStacktrace call above
		Sampling:          nil,  // keep all logs for now
		Encoding:          encoding,
		EncoderConfig: zapcore.EncoderConfig{
			MessageKey:     "message",
			LevelKey:       "level",
			TimeKey:        "timestamp",
			NameKey:        "logger",
			CallerKey:      "caller",
			FunctionKey:    zapcore.OmitKey, // included in caller
			StacktraceKey:  "stacktrace",
			SkipLineEnding: false,
			LineEnding:     zapcore.DefaultLineEnding,
			EncodeLevel:    levelEncoder,
			// human-readable and machine parsable with microsecond precision (same as klog, kube audit event, etc)
			EncodeTime:          zapcore.TimeEncoderOfLayout(metav1.RFC3339Micro),
			EncodeDuration:      zapcore.StringDurationEncoder,
			EncodeCaller:        callerEncoder,
			EncodeName:          nil,
			NewReflectedEncoder: nil,
			ConsoleSeparator:    "  ",
		},
		OutputPaths:      []string{path},
		ErrorOutputPaths: []string{path},
		InitialFields:    nil,
	}

	f(&config)

	log, err := config.Build(opts...)
	if err != nil {
		return logr.Logger{}, nil, fmt.Errorf("failed to build zap logger: %w", err)
	}

	return zapr.NewLogger(log), func() { _ = log.Sync() }, nil
}

func levelEncoder(l zapcore.Level, enc zapcore.PrimitiveArrayEncoder) {
	mlogLevel := zapLevelToMlogLevel(l)

	if len(mlogLevel) == 0 {
		return // this tells zap that it should handle encoding the level itself because we do not know the mapping
	}

	enc.AppendString(string(mlogLevel))
}

func zapLevelToMlogLevel(l zapcore.Level) LogLevel {
	if l > 0 {
		// best effort mapping, the zap levels do not really translate to klog
		// but this is correct for "error" level which is all we need for logr
		return LogLevel(l.String())
	}

	// klog levels are inverted when zap handles them
	switch {
	case -l >= klogLevelAll:
		return LevelAll
	case -l >= klogLevelTrace:
		return LevelTrace
	case -l >= klogLevelDebug:
		return LevelDebug
	case -l >= klogLevelInfo:
		return LevelInfo
	default:
		return "" // warning is handled via a custom key since klog level 0 is ambiguous
	}
}

func callerEncoder(caller zapcore.EntryCaller, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString(caller.String() + funcEncoder(caller))
}

func funcEncoder(caller zapcore.EntryCaller) string {
	funcName := caller.Function
	if idx := strings.LastIndexByte(funcName, '/'); idx != -1 {
		funcName = funcName[idx+1:] // keep everything after the last /
	}
	return "$" + funcName
}

func humanDurationEncoder(d time.Duration, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString(duration.HumanDuration(d))
}

func humanTimeEncoder(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString(t.Local().Format(time.RFC1123))
}

var _ zapcore.Core = &trimCore{}

type trimCore struct {
	core zapcore.Core
}

func (t *trimCore) Enabled(level zapcore.Level) bool {
	return t.core.Enabled(level)
}

func (t *trimCore) With(fields []zapcore.Field) zapcore.Core {
	return &trimCore{core: t.core.With(fields)}
}

func (t *trimCore) Check(ent zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if strings.HasSuffix(ent.Message, "\n") {
		ent.Message = ent.Message[:len(ent.Message)-1]
	}

	if downstream := t.core.Check(ent, ce); downstream != nil {
		return downstream
	}

	return ce
}

func (t *trimCore) Write(ent zapcore.Entry, fields []zapcore.Field) error {
	return t.core.Write(ent, fields)
}

func (t *trimCore) Sync() error {
	return t.core.Sync()
}

var _ io.Writer = &trimWriter{}

type trimWriter struct {
	w io.Writer
}

var brokenNewlineThenQuoteThenActualNewline = []byte(`\n"
`)

func (t *trimWriter) Write(p []byte) (int, error) {
	if bytes.HasSuffix(p, brokenNewlineThenQuoteThenActualNewline) {
		// overwrite the broken newline with the quote and correct newline
		p[len(p)-4] = '"'
		p[len(p)-3] = '\n'
		p = p[:len(p)-2] // then just slice off the original quote and newline
	}
	return t.w.Write(p)
}
