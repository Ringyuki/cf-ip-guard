package logging

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/buffer"
	"go.uber.org/zap/zapcore"
)

type Logger = *zap.SugaredLogger

var std Logger = zap.NewExample().Sugar()

func Init(level, format, color string) (Logger, error) {
	lvl, err := parseLevel(level)
	if err != nil {
		return nil, err
	}

	lowerFmt := strings.ToLower(strings.TrimSpace(format))
	if lowerFmt == "" {
		lowerFmt = "text"
	}

	var encoder zapcore.Encoder
	switch lowerFmt {
	case "text":
		encoder = newPrettyEncoder("cf-ip-guard", shouldColor(lowerFmt, color))
	case "json":
		encCfg := zap.NewProductionEncoderConfig()
		encCfg.TimeKey = "time"
		encCfg.EncodeTime = func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
			enc.AppendString(t.Format(time.RFC3339))
		}
		encCfg.EncodeDuration = zapcore.StringDurationEncoder
		encCfg.EncodeCaller = zapcore.ShortCallerEncoder
		encCfg.NameKey = "logger"
		encCfg.CallerKey = "caller"
		encCfg.EncodeLevel = zapcore.CapitalLevelEncoder
		encoder = zapcore.NewJSONEncoder(encCfg)
	default:
		return nil, fmt.Errorf("unsupported log format: %s", format)
	}

	ws := zapcore.Lock(os.Stderr)
	core := zapcore.NewCore(encoder, ws, lvl)

	base := zap.New(core,
		zap.AddCaller(),
		zap.AddCallerSkip(1),
		zap.Fields(zap.String("logger", "cf-ip-guard"), zap.Int("pid", os.Getpid())),
	)
	logger := base.Sugar()
	std = logger
	return logger, nil
}

func L() Logger {
	return std
}

func Set(l Logger) {
	if l != nil {
		std = l
	}
}

func parseLevel(level string) (zapcore.LevelEnabler, error) {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "", "info":
		return zapcore.InfoLevel, nil
	case "debug":
		return zapcore.DebugLevel, nil
	case "warn", "warning":
		return zapcore.WarnLevel, nil
	case "error":
		return zapcore.ErrorLevel, nil
	default:
		return zapcore.InfoLevel, fmt.Errorf("unsupported log level: %s", level)
	}
}

func shouldColor(format, color string) bool {
	switch strings.ToLower(strings.TrimSpace(color)) {
	case "on":
		return true
	case "off":
		return false
	}

	if format == "json" {
		return false
	}
	if _, ok := os.LookupEnv("NO_COLOR"); ok {
		return false
	}
	term := os.Getenv("TERM")
	return term != "" && term != "dumb"
}

// [cf-ip-guard] 12345 2025/12/11 13:44:46 ERROR [daemon] message k=v ...
type prettyEncoder struct {
	zapcore.Encoder
	app   string
	color bool
}

var prettyBufPool = buffer.NewPool()

func newPrettyEncoder(app string, color bool) zapcore.Encoder {
	encCfg := zap.NewProductionEncoderConfig()
	encCfg.TimeKey = ""
	encCfg.LevelKey = ""
	encCfg.NameKey = ""
	encCfg.CallerKey = ""
	encCfg.MessageKey = ""
	encCfg.StacktraceKey = ""
	inner := zapcore.NewConsoleEncoder(encCfg)
	return &prettyEncoder{Encoder: inner, app: app, color: color}
}

func (e *prettyEncoder) Clone() zapcore.Encoder {
	return &prettyEncoder{
		Encoder: e.Encoder.Clone(),
		app:     e.app,
		color:   e.color,
	}
}

func (e *prettyEncoder) EncodeEntry(ent zapcore.Entry, fields []zapcore.Field) (*buffer.Buffer, error) {
	m := zapcore.NewMapObjectEncoder()
	for _, f := range fields {
		f.AddTo(m)
	}

	pid := os.Getpid()
	if v, ok := m.Fields["pid"]; ok {
		switch val := v.(type) {
		case int:
			pid = val
		case int64:
			pid = int(val)
		}
		delete(m.Fields, "pid")
	}
	delete(m.Fields, "logger")

	ts := ent.Time.Format(time.RFC3339)
	level := strings.ToUpper(ent.Level.String())
	if e.color {
		level = colorize(ent.Level, level)
	}

	buf := prettyBufPool.Get()
	fmt.Fprintf(buf, "[%s] %d %s %-5s", e.app, pid, ts, level)
	if ent.LoggerName != "" {
		fmt.Fprintf(buf, " [%s]", ent.LoggerName)
	}
	if ent.Message != "" {
		fmt.Fprintf(buf, " %s", ent.Message)
	}

	if len(m.Fields) > 0 {
		keys := make([]string, 0, len(m.Fields))
		for k := range m.Fields {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			fmt.Fprintf(buf, " %s=%v", k, m.Fields[k])
		}
	}

	buf.AppendByte('\n')
	return buf, nil
}

// colorize level
const (
	colReset  = "\033[0m"
	colRed    = "\033[31m"
	colYellow = "\033[33m"
	colBlue   = "\033[34m"
	colGreen  = "\033[32m"
)

func colorize(lvl zapcore.Level, s string) string {
	switch lvl {
	case zapcore.DebugLevel:
		return colBlue + s + colReset
	case zapcore.InfoLevel:
		return colGreen + s + colReset
	case zapcore.WarnLevel:
		return colYellow + s + colReset
	default:
		return colRed + s + colReset
	}
}
